package headless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Instance represents the single headless OpenCode instance
type Instance struct {
	Port   int    `json:"port"`
	PID    int    `json:"pid"`
	Status string `json:"status"`
	cmd    *exec.Cmd
}

// Session represents a session for a specific project
type Session struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Agent     string `json:"agent"`
	Status    string `json:"status"`
}

// Manager manages the headless OpenCode instance and sessions
type Manager struct {
	instance *Instance
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new headless manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found starting from %d", startPort)
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Start starts a single headless OpenCode instance
func (m *Manager) Start(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance != nil {
		return fmt.Errorf("instance already running")
	}

	// Find available port if requested port is in use
	if !IsPortAvailable(port) {
		availablePort, err := FindAvailablePort(port + 1)
		if err != nil {
			return fmt.Errorf("port %d in use and no alternatives found", port)
		}
		log.Printf("Port %d in use, using %d instead", port, availablePort)
		port = availablePort
	}

	// Find opencode binary
	opencodePath, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found in PATH: %w", err)
	}

	// Start OpenCode in headless server mode
	cmd := exec.Command(opencodePath, "serve",
		"--port", fmt.Sprintf("%d", port),
		"--hostname", "127.0.0.1",
	)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENCODE_PORT=%d", port),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	m.instance = &Instance{
		Port:   port,
		PID:    cmd.Process.Pid,
		Status: "starting",
		cmd:    cmd,
	}

	// Wait for ready
	go m.waitForReady()

	// Monitor
	go m.monitor()

	return nil
}

// waitForReady waits for the instance to be healthy
func (m *Manager) waitForReady() {
	client := &http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", m.instance.Port)

	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			m.mu.Lock()
			m.instance.Status = "running"
			m.mu.Unlock()
			log.Printf("✓ OpenCode ready on port %d", m.instance.Port)
			return
		}
	}

	m.mu.Lock()
	m.instance.Status = "failed"
	m.mu.Unlock()
	log.Printf("✗ OpenCode failed to start")
}

// monitor watches for crashes
func (m *Manager) monitor() {
	err := m.instance.cmd.Wait()

	m.mu.RLock()
	status := m.instance.Status
	m.mu.RUnlock()

	if status == "stopped" || status == "stopping" {
		return
	}

	if err != nil {
		log.Printf("✗ OpenCode exited: %v", err)
	} else {
		log.Printf("✗ OpenCode exited")
	}

	m.mu.Lock()
	m.instance.Status = "stopped"
	m.mu.Unlock()
}

// Stop stops the instance
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance == nil {
		return fmt.Errorf("no instance running")
	}

	m.instance.Status = "stopping"

	if m.instance.cmd != nil && m.instance.cmd.Process != nil {
		m.instance.cmd.Process.Signal(os.Interrupt)

		done := make(chan error, 1)
		go func() {
			done <- m.instance.cmd.Wait()
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			m.instance.cmd.Process.Kill()
		}
	}

	m.instance = nil
	return nil
}

// GetPort returns the instance port
func (m *Manager) GetPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.instance == nil {
		return 0
	}
	return m.instance.Port
}

// IsRunning checks if the instance is running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.instance != nil && m.instance.Status == "running"
}

// CreateSession creates a new session for a project
func (m *Manager) CreateSession(name, path, agent string) (*Session, error) {
	m.mu.RLock()
	port := 0
	if m.instance != nil {
		port = m.instance.Port
	}
	m.mu.RUnlock()

	if port == 0 {
		return nil, fmt.Errorf("no instance running")
	}

	// Create session via OpenCode API
	url := fmt.Sprintf("http://127.0.0.1:%d/session.create", port)

	body, _ := json.Marshal(map[string]interface{}{
		"title": fmt.Sprintf("%s - %s", name, agent),
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	session := &Session{
		ID:     result.ID,
		Name:   name,
		Path:   path,
		Agent:  agent,
		Status: "active",
	}

	m.mu.Lock()
	m.sessions[name] = session
	m.mu.Unlock()

	log.Printf("✓ Session created for %s (ID: %s)", name, result.ID)

	return session, nil
}

// GetSession returns a session by name
func (m *Manager) GetSession(name string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[name]
	return session, ok
}

// ListSessions returns all sessions
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// SendPrompt sends a prompt to a session
func (m *Manager) SendPrompt(sessionName, prompt string) (string, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	port := 0
	if m.instance != nil {
		port = m.instance.Port
	}
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionName)
	}

	if port == 0 {
		return "", fmt.Errorf("no instance running")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/session.prompt", port)

	body, _ := json.Marshal(map[string]interface{}{
		"sessionID": session.ID,
		"parts": []map[string]string{
			{"type": "text", "text": prompt},
		},
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content string `json:"content"`
		Text    string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "Message sent", nil
	}

	if result.Content != "" {
		return result.Content, nil
	}
	return result.Text, nil
}
