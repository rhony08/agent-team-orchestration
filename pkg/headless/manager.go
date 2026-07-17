package headless

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Instance represents a headless OpenCode instance
type Instance struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Port   int    `json:"port"`
	PID    int    `json:"pid"`
	Status string `json:"status"`
	cmd    *exec.Cmd
}

// Manager manages headless OpenCode instances
type Manager struct {
	instances map[string]*Instance
	mu        sync.RWMutex
	nextPort  int
}

// NewManager creates a new headless manager
func NewManager(startPort int) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		nextPort:  startPort,
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

	// Find available port if requested port is in use
	if !IsPortAvailable(port) {
		availablePort, err := FindAvailablePort(port + 1)
		if err != nil {
			return fmt.Errorf("port %d in use and no alternatives found", port)
		}
		log.Printf("Port %d in use, using %d instead for %s", port, availablePort, name)
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
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENCODE_PORT=%d", port),
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

	if status == "stopped" || status == "stopping" {
		return
	}

	if err != nil {
		log.Printf("✗ %s exited: %v", inst.Name, err)
	} else {
		log.Printf("✗ %s exited", inst.Name)
	}

	m.mu.Lock()
	inst.Status = "stopped"
	m.mu.Unlock()
}

// Stop stops an instance
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.instances[name]
	if !exists {
		return fmt.Errorf("instance not found: %s", name)
	}

	inst.Status = "stopping"

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

// GetInstancePort returns the port for an instance
func (m *Manager) GetInstancePort(name string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.instances[name]
	if !ok {
		return 0, false
	}
	return inst.Port, true
}
