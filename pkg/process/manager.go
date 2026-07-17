package process

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Process represents a managed OpenCode process
type Process struct {
	Name   string
	Path   string
	Port   int
	PID    int
	Cmd    *exec.Cmd
	Status string
}

// Manager manages OpenCode processes
type Manager struct {
	processes map[string]*Process
	mu        sync.Mutex
	nextPort  int
}

// NewManager creates a new process manager
func NewManager(startPort int) *Manager {
	return &Manager{
		processes: make(map[string]*Process),
		nextPort:  startPort,
	}
}

// Spawn starts an OpenCode instance in the given directory
func (m *Manager) Spawn(name, dir string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.processes[name]; exists {
		return fmt.Errorf("process already exists: %s", name)
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

	cmd := exec.Command(opencodePath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENCODE_PORT=%d", port),
		fmt.Sprintf("OPENCODE_HOST=127.0.0.1"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	p := &Process{
		Name:   name,
		Path:   dir,
		Port:   port,
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Status: "starting",
	}

	m.processes[name] = p

	// Wait for process to be ready
	go m.waitForReady(p)

	// Monitor process
	go m.monitor(p)

	return nil
}

// waitForReady waits for the OpenCode instance to be healthy
func (m *Manager) waitForReady(p *Process) {
	client := &http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", p.Port)

	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			m.mu.Lock()
			p.Status = "running"
			m.mu.Unlock()
			log.Printf("OpenCode instance %s is ready on port %d", p.Name, p.Port)
			return
		}
	}

	m.mu.Lock()
	p.Status = "failed"
	m.mu.Unlock()
	log.Printf("OpenCode instance %s failed to become ready", p.Name)
}

// monitor watches a process and restarts if it crashes
func (m *Manager) monitor(p *Process) {
	err := p.Cmd.Wait()

	m.mu.Lock()
	status := p.Status
	m.mu.Unlock()

	if status == "stopped" {
		return
	}

	if err != nil {
		log.Printf("OpenCode instance %s crashed: %v", p.Name, err)
		m.mu.Lock()
		p.Status = "crashed"
		m.mu.Unlock()

		// Restart up to 3 times
		for i := 0; i < 3; i++ {
			log.Printf("Restarting %s (attempt %d/3)...", p.Name, i+1)
			time.Sleep(2 * time.Second)

			if err := m.restart(p.Name); err != nil {
				log.Printf("Failed to restart %s: %v", p.Name, err)
			} else {
				return
			}
		}

		log.Printf("Failed to restart %s after 3 attempts", p.Name)
	}
}

// restart restarts a process
func (m *Manager) restart(name string) error {
	m.mu.Lock()
	p, exists := m.processes[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("process not found: %s", name)
	}

	// Clean up old process
	if p.Cmd != nil && p.Cmd.Process != nil {
		p.Cmd.Process.Kill()
	}
	delete(m.processes, name)
	m.mu.Unlock()

	// Reuse port
	return m.Spawn(name, p.Path, p.Port)
}

// Stop stops a process
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.processes[name]
	if !exists {
		return fmt.Errorf("process not found: %s", name)
	}

	p.Status = "stopped"

	if p.Cmd != nil && p.Cmd.Process != nil {
		// Send SIGTERM
		p.Cmd.Process.Signal(os.Interrupt)

		// Wait up to 10 seconds
		done := make(chan error, 1)
		go func() {
			done <- p.Cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(10 * time.Second):
			// Force kill
			p.Cmd.Process.Kill()
		}
	}

	delete(m.processes, name)
	return nil
}

// StopAll stops all processes
func (m *Manager) StopAll() {
	m.mu.Lock()
	names := make([]string, 0, len(m.processes))
	for name := range m.processes {
		names = append(names, name)
	}
	m.mu.Unlock()

	for _, name := range names {
		m.Stop(name)
	}
}

// HealthCheck checks the health of all processes
func (m *Manager) HealthCheck() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := make(map[string]string)
	client := &http.Client{Timeout: 1 * time.Second}

	for name, p := range m.processes {
		if p.Status != "running" {
			status[name] = p.Status
			continue
		}

		url := fmt.Sprintf("http://127.0.0.1:%d/health", p.Port)
		resp, err := client.Get(url)
		if err != nil || resp.StatusCode != 200 {
			status[name] = "unhealthy"
		} else {
			status[name] = "healthy"
		}
	}

	return status
}

// GetProcess returns a process by name
func (m *Manager) GetProcess(name string) (*Process, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.processes[name]
	return p, ok
}

// AllocatePort allocates the next available port
func (m *Manager) AllocatePort() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	port := m.nextPort
	m.nextPort++
	return port
}

// WritePIDFiles writes PID files for all processes
func (m *Manager) WritePIDFiles(stateDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pidDir := filepath.Join(stateDir, "pids")
	os.MkdirAll(pidDir, 0755)

	for name, p := range m.processes {
		pidFile := filepath.Join(pidDir, name+".pid")
		if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", p.PID)), 0644); err != nil {
			return err
		}
	}

	return nil
}

// CleanupPIDFiles removes PID files
func (m *Manager) CleanupPIDFiles(stateDir string) {
	pidDir := filepath.Join(stateDir, "pids")
	os.RemoveAll(pidDir)
}
