package headless

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Instance represents the single headless OpenCode instance
type Instance struct {
	Port     int    `json:"port"`
	PID      int    `json:"pid"`
	Status   string `json:"status"`
	Username string `json:"username"`
	Password string `json:"password"`
	cmd      *exec.Cmd
}

// Session represents a session for a specific project
type Session struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Agent  string `json:"agent"`
	Status string `json:"status"`
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

// Start starts a single headless OpenCode instance and captures credentials
func (m *Manager) Start(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.instance != nil {
		return fmt.Errorf("instance already running")
	}

	if !IsPortAvailable(port) {
		availablePort, err := FindAvailablePort(port + 1)
		if err != nil {
			return fmt.Errorf("port %d in use and no alternatives found", port)
		}
		log.Printf("Port %d in use, using %d instead", port, availablePort)
		port = availablePort
	}

	opencodePath, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found in PATH: %w", err)
	}

	cmd := exec.Command(opencodePath, "serve",
		"--port", fmt.Sprintf("%d", port),
		"--hostname", "127.0.0.1",
	)
	cmd.Env = os.Environ()

	// Capture stderr to get credentials
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	m.instance = &Instance{
		Port:   port,
		PID:    cmd.Process.Pid,
		Status: "starting",
		cmd:    cmd,
	}

	// Read stderr to capture credentials
	go m.captureCredentials(stderr)

	go m.monitor()

	return nil
}

// captureCredentials reads stderr and captures credentials from child process
func (m *Manager) captureCredentials(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[opencode] %s", line)
	}

	// Process has started, now capture credentials from /proc/PID/environ
	time.Sleep(500 * time.Millisecond)

	m.mu.RLock()
	inst := m.instance
	m.mu.RUnlock()

	if inst == nil {
		return
	}

	environPath := fmt.Sprintf("/proc/%d/environ", inst.PID)
	data, err := os.ReadFile(environPath)
	if err != nil {
		log.Printf("Warning: Could not read process environment: %v", err)
	} else {
		envs := strings.Split(string(data), "\000")
		for _, env := range envs {
			if strings.HasPrefix(env, "OPENCODE_SERVER_USERNAME=") {
				m.mu.Lock()
				if m.instance != nil {
					m.instance.Username = strings.TrimPrefix(env, "OPENCODE_SERVER_USERNAME=")
				}
				m.mu.Unlock()
			}
			if strings.HasPrefix(env, "OPENCODE_SERVER_PASSWORD=") {
				m.mu.Lock()
				if m.instance != nil {
					m.instance.Password = strings.TrimPrefix(env, "OPENCODE_SERVER_PASSWORD=")
				}
				m.mu.Unlock()
			}
		}
	}

	m.mu.RLock()
	username := ""
	port := 0
	if m.instance != nil {
		username = m.instance.Username
		port = m.instance.Port
	}
	m.mu.RUnlock()

	if username != "" {
		log.Printf("✓ Credentials captured (user: %s)", username)
	} else {
		log.Printf("⚠ No credentials found, API calls may fail")
	}

	// Mark as running
	m.mu.Lock()
	if m.instance != nil {
		m.instance.Status = "running"
	}
	m.mu.Unlock()

	if port > 0 {
		log.Printf("✓ OpenCode ready on port %d", port)
	}
}

func (m *Manager) monitor() {
	// Capture instance reference before waiting
	m.mu.RLock()
	inst := m.instance
	m.mu.RUnlock()

	if inst == nil || inst.cmd == nil {
		return
	}

	err := inst.cmd.Wait()

	// Check if instance was stopped intentionally
	m.mu.RLock()
	if m.instance == nil {
		m.mu.RUnlock()
		return
	}
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
	if m.instance != nil {
		m.instance.Status = "stopped"
	}
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

// apiCall makes an authenticated API call
func (m *Manager) apiCall(method, path string, body interface{}) (*http.Response, error) {
	m.mu.RLock()
	inst := m.instance
	m.mu.RUnlock()

	if inst == nil || inst.Status != "running" {
		return nil, fmt.Errorf("no instance running")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", inst.Port, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add auth if credentials available
	if inst.Username != "" && inst.Password != "" {
		req.SetBasicAuth(inst.Username, inst.Password)
	}

	return http.DefaultClient.Do(req)
}

// CreateSession creates a new session
func (m *Manager) CreateSession(name, path, agent string) (*Session, error) {
	resp, err := m.apiCall("POST", "/session", map[string]interface{}{
		"title": fmt.Sprintf("%s - %s", name, agent),
	})
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

// SendPrompt sends a prompt to a session (non-blocking)
func (m *Manager) SendPrompt(sessionName, prompt string) (string, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionName)
	}

	// Send prompt async (non-blocking)
	resp, err := m.apiCall("POST", fmt.Sprintf("/session/%s/prompt_async", session.ID), map[string]interface{}{
		"parts": []map[string]string{
			{"type": "text", "text": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	return "Message sent. Use 'crush-orchestrator sessions' to check status.", nil
}

// GetMessages gets messages from a session
func (m *Manager) GetMessages(sessionName string) ([]Message, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionName)
	}

	resp, err := m.apiCall("GET", fmt.Sprintf("/session/%s/message", session.ID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rawMessages []struct {
		Info struct {
			ID        string `json:"id"`
			Role      string `json:"role"`
			SessionID string `json:"sessionID"`
			Time      struct {
				Created int64 `json:"created"`
			} `json:"time"`
		} `json:"info"`
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawMessages); err != nil {
		return nil, err
	}

	var messages []Message
	for _, msg := range rawMessages {
		for _, part := range msg.Parts {
			if part.Type == "text" && part.Text != "" {
				messages = append(messages, Message{
					ID:      msg.Info.ID,
					Role:    msg.Info.Role,
					Content: part.Text,
					Time:    time.Unix(msg.Info.Time.Created/1000, 0),
				})
			}
		}
	}

	return messages, nil
}

// Message represents a chat message
type Message struct {
	ID      string    `json:"id"`
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}
