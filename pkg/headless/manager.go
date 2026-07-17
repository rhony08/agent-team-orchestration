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

// Start starts a single headless OpenCode instance
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

	go m.waitForReady()
	go m.monitor()

	return nil
}

func (m *Manager) waitForReady() {
	client := &http.Client{Timeout: 2 * time.Second}

	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		// Try to access the API - no auth needed for health check
		// Just check if the server is responding
		url := fmt.Sprintf("http://127.0.0.1:%d/session", m.instance.Port)
		req, _ := http.NewRequest("GET", url, nil)

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 401 {
				// Server is up (401 means auth is required but server is running)
				m.mu.Lock()
				m.instance.Status = "running"
				m.mu.Unlock()
				log.Printf("✓ OpenCode ready on port %d", m.instance.Port)
				return
			}
		}
	}

	m.mu.Lock()
	m.instance.Status = "failed"
	m.mu.Unlock()
	log.Printf("✗ OpenCode failed to start")
}

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

// CreateSession creates a new session using opencode run
func (m *Manager) CreateSession(name, path, agent string) (*Session, error) {
	m.mu.RLock()
	inst := m.instance
	m.mu.RUnlock()

	if inst == nil || inst.Status != "running" {
		return nil, fmt.Errorf("no instance running")
	}

	// Use opencode CLI to create session (handles auth automatically)
	opencodePath, err := exec.LookPath("opencode")
	if err != nil {
		return nil, err
	}

	// Run a simple command to create a session
	cmd := exec.Command(opencodePath, "run",
		"--attach", fmt.Sprintf("http://127.0.0.1:%d", inst.Port),
		"--title", fmt.Sprintf("%s - %s", name, agent),
		"--format", "json",
		"--agent", agent,
		"echo initializing",
	)
	cmd.Dir = path
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("CreateSession output: %s", string(output))
		// Try alternative approach - direct API call
		return m.createSessionViaAPI(name, path, agent)
	}

	// Parse session ID from output
	sessionID := parseSessionID(string(output))
	if sessionID == "" {
		return m.createSessionViaAPI(name, path, agent)
	}

	session := &Session{
		ID:     sessionID,
		Name:   name,
		Path:   path,
		Agent:  agent,
		Status: "active",
	}

	m.mu.Lock()
	m.sessions[name] = session
	m.mu.Unlock()

	log.Printf("✓ Session created for %s (ID: %s)", name, sessionID)

	return session, nil
}

// createSessionViaAPI creates a session via direct API call
func (m *Manager) createSessionViaAPI(name, path, agent string) (*Session, error) {
	m.mu.RLock()
	inst := m.instance
	m.mu.RUnlock()

	url := fmt.Sprintf("http://127.0.0.1:%d/session", inst.Port)

	body, _ := json.Marshal(map[string]interface{}{
		"title": fmt.Sprintf("%s - %s", name, agent),
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Try without auth first (if OPENCODE_SERVER_PASSWORD not set)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Need auth - get credentials from environment
		username := os.Getenv("OPENCODE_SERVER_USERNAME")
		password := os.Getenv("OPENCODE_SERVER_PASSWORD")

		if username == "" || password == "" {
			return nil, fmt.Errorf("OpenCode requires auth. Set OPENCODE_SERVER_USERNAME and OPENCODE_SERVER_PASSWORD environment variables")
		}

		req.SetBasicAuth(username, password)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

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

// SendPrompt sends a prompt to a session using opencode run
func (m *Manager) SendPrompt(sessionName, prompt string) (string, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	inst := m.instance
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionName)
	}

	if inst == nil || inst.Status != "running" {
		return "", fmt.Errorf("no instance running")
	}

	// Use opencode CLI to send prompt (handles auth automatically)
	opencodePath, err := exec.LookPath("opencode")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(opencodePath, "run",
		"--attach", fmt.Sprintf("http://127.0.0.1:%d", inst.Port),
		"--session", session.ID,
		"--format", "json",
		prompt,
	)
	cmd.Dir = session.Path
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w\nOutput: %s", err, string(output))
	}

	// Parse response from output
	response := parseResponse(string(output))
	if response == "" {
		return "Message sent", nil
	}

	return response, nil
}

// GetMessages gets messages from a session
func (m *Manager) GetMessages(sessionName string) ([]string, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionName]
	inst := m.instance
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionName)
	}

	if inst == nil || inst.Status != "running" {
		return nil, fmt.Errorf("no instance running")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/session/%s/message", inst.Port, session.ID)

	req, _ := http.NewRequest("GET", url, nil)

	// Try without auth first
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		username := os.Getenv("OPENCODE_SERVER_USERNAME")
		password := os.Getenv("OPENCODE_SERVER_PASSWORD")
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}
	}

	var messages []struct {
		Info struct {
			Role string `json:"role"`
		} `json:"info"`
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}

	var result []string
	for _, msg := range messages {
		for _, part := range msg.Parts {
			if part.Type == "text" && part.Text != "" {
				result = append(result, part.Text)
			}
		}
	}

	return result, nil
}

// parseSessionID extracts session ID from opencode run output
func parseSessionID(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for session ID pattern
		if strings.Contains(line, "ses_") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "ses_") {
					return strings.Trim(part, "\"',.")
				}
			}
		}
		// Look for JSON with id field
		if strings.Contains(line, "\"id\"") && strings.Contains(line, "ses_") {
			var obj struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				return obj.ID
			}
		}
	}
	return ""
}

// parseResponse extracts response text from opencode run output
func parseResponse(output string) string {
	lines := strings.Split(output, "\n")
	var responseLines []string
	inResponse := false

	for _, line := range lines {
		// Look for assistant response markers
		if strings.Contains(line, "\"role\":\"assistant\"") || strings.Contains(line, "\"role\": \"assistant\"") {
			inResponse = true
			continue
		}
		if inResponse && strings.Contains(line, "\"text\"") {
			// Extract text content
			parts := strings.SplitN(line, "\"text\":", 2)
			if len(parts) > 1 {
				text := strings.TrimSpace(parts[1])
				text = strings.Trim(text, "\"")
				text = strings.TrimSuffix(text, "\"")
				responseLines = append(responseLines, text)
			}
		}
	}

	if len(responseLines) > 0 {
		return strings.Join(responseLines, "\n")
	}
	return ""
}
