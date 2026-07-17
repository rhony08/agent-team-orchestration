package process

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Process represents a tracked OpenCode instance
type Process struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Port   int    `json:"port"`
	Status string `json:"status"`
}

// Manager tracks OpenCode instances that connect to the API
type Manager struct {
	processes map[string]*Process
	mu        sync.RWMutex
}

// NewManager creates a new process manager
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*Process),
	}
}

// Register registers an OpenCode instance
func (m *Manager) Register(name, path string, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processes[name] = &Process{
		Name:   name,
		Path:   path,
		Port:   port,
		Status: "registered",
	}
}

// SetStatus updates the status of a process
func (m *Manager) SetStatus(name, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.processes[name]; ok {
		p.Status = status
	}
}

// Heartbeat updates the last seen time for a process
func (m *Manager) Heartbeat(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.processes[name]; ok {
		p.Status = "connected"
	}
}

// GetProcess returns a process by name
func (m *Manager) GetProcess(name string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.processes[name]
	return p, ok
}

// ListProcesses returns all registered processes
func (m *Manager) ListProcesses() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	processes := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		processes = append(processes, p)
	}
	return processes
}

// HealthCheck checks connectivity to registered OpenCode instances
func (m *Manager) HealthCheck() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]string)
	client := &http.Client{Timeout: 2 * time.Second}

	for name, p := range m.processes {
		if p.Port == 0 {
			status[name] = "no-port"
			continue
		}

		url := fmt.Sprintf("http://127.0.0.1:%d/health", p.Port)
		resp, err := client.Get(url)
		if err != nil {
			status[name] = "unreachable"
		} else if resp.StatusCode == 200 {
			status[name] = "healthy"
		} else {
			status[name] = "unhealthy"
		}
	}

	return status
}

// Unregister removes a process
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.processes, name)
}
