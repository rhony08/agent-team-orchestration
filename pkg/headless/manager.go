package headless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Instance represents a headless OpenCode instance
type Instance struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Port     int    `json:"port"`
	PID      int    `json:"pid"`
	Status   string `json:"status"`
	cmd      *exec.Cmd
}

// Manager manages headless OpenCode instances
type Manager struct {
	instances map[string]*Instance
	mu        sync.RWMutex
	startPort int
}

// NewManager creates a new headless manager
func NewManager(startPort int) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		startPort: startPort,
	}
}

// Spawn starts a headless OpenCode instance
func (m *Manager) Spawn(name, dir string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.instances[name]; exists {
		return fmt.Errorf("instance already exists: %s", name)
	}

	// Verify directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", dir)
	}

	// Find opencode binary
	opencodePath, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found in PATH: %w", err)
	}

	// Start OpenCode in server mode (headless)
	// OpenCode supports running as a server via its SDK
	cmd := exec.Command(opencodePath, "serve", "--port", fmt.Sprintf("%d", port), "--host", "127.0.0.1")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENCODE_PORT=%d", port),
		"OPENCODE_HEADLESS=true",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	inst := &Instance{
		Name:   name,
		Path:   dir,
		Port:   port,
		PID:    cmd.Process.Pid,
		Status: "starting",
		cmd:    cmd,
	}

	m.instances[name] = inst

	// Wait for ready
	go m.waitForReady(inst)

	// Monitor
	go m.monitor(inst)

	return nil
}

// waitForReady waits for the instance to be healthy
func (m *Manager) waitForReady(inst *Instance) {
	client := &http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", inst.Port)

	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			m.mu.Lock()
			inst.Status = "running"
			m.mu.Unlock()
			log.Printf("✓ %s ready on port %d", inst.Name, inst.Port)
			return
		}
	}

	m.mu.Lock()
	inst.Status = "failed"
	m.mu.Unlock()
	log.Printf("✗ %s failed to start", inst.Name)
}

// monitor watches for crashes
func (m *Manager) monitor(inst *Instance) {
	err := inst.cmd.Wait()

	m.mu.RLock()
	status := inst.Status
	m.mu.RUnlock()

	if status == "stopped" {
		return
	}

	if err != nil {
		log.Printf("✗ %s crashed: %v", inst.Name, err)
		m.mu.Lock()
		inst.Status = "crashed"
		m.mu.Unlock()
	}
}

// Stop stops an instance
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.instances[name]
	if !exists {
		return fmt.Errorf("instance not found: %s", name)
	}

	inst.Status = "stopped"

	if inst.cmd != nil && inst.cmd.Process != nil {
		inst.cmd.Process.Signal(os.Interrupt)

		done := make(chan error, 1)
		go func() {
			done <- inst.cmd.Wait()
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			inst.cmd.Process.Kill()
		}
	}

	delete(m.instances, name)
	return nil
}

// StopAll stops all instances
func (m *Manager) StopAll() {
	m.mu.Lock()
	names := make([]string, 0, len(m.instances))
	for name := range m.instances {
		names = append(names, name)
	}
	m.mu.Unlock()

	for _, name := range names {
		m.Stop(name)
	}
}

// SendMessage sends a prompt to an instance and returns the response
func (m *Manager) SendMessage(instanceName, agentType, message string) (string, error) {
	m.mu.RLock()
	inst, exists := m.instances[instanceName]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("instance not found: %s", instanceName)
	}

	if inst.Status != "running" {
		return "", fmt.Errorf("instance not ready: %s (status: %s)", instanceName, inst.Status)
	}

	// Create session
	sessionID, err := m.createSession(inst, agentType)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Send prompt
	response, err := m.sendPrompt(inst, sessionID, message)
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	return response, nil
}

// createSession creates a new session in the instance
func (m *Manager) createSession(inst *Instance, agentType string) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/session", inst.Port)

	reqBody := map[string]string{
		"agent": agentType,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

// sendPrompt sends a prompt to a session
func (m *Manager) sendPrompt(inst *Instance, sessionID, prompt string) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/session/%s/prompt", inst.Port, sessionID)

	reqBody := map[string]string{
		"prompt": prompt,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

// GetInstance returns an instance by name
func (m *Manager) GetInstance(name string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.instances[name]
	return inst, ok
}

// ListInstances returns all instances
func (m *Manager) ListInstances() []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instances := make([]*Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		instances = append(instances, inst)
	}
	return instances
}

// AllocatePort allocates the next available port
func (m *Manager) AllocatePort() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	port := m.startPort
	m.startPort++
	return port
}
