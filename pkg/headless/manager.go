package headless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Manager connects to an existing OpenCode server
type Manager struct {
	baseURL  string
	username string
	password string
	sessions map[string]*Session
	mu       sync.RWMutex
}

// Session represents a session for a specific project
type Session struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Agent  string `json:"agent"`
	Status string `json:"status"`
}

// Message represents a chat message
type Message struct {
	ID      string    `json:"id"`
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
	Error   string    `json:"error,omitempty"`
}

// NewManager creates a new manager that connects to existing OpenCode
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// Connect connects to an existing OpenCode server
func (m *Manager) Connect(port int) error {
	// Try to get credentials from environment
	m.username = os.Getenv("OPENCODE_SERVER_USERNAME")
	m.password = os.Getenv("OPENCODE_SERVER_PASSWORD")
	m.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	// Test connection
	resp, err := m.apiCall("GET", "/session", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to OpenCode on port %d: %w", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 && (m.username == "" || m.password == "") {
		return fmt.Errorf("OpenCode requires auth. Set OPENCODE_SERVER_USERNAME and OPENCODE_SERVER_PASSWORD")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	log.Printf("✓ Connected to OpenCode at %s (user: %s)", m.baseURL, m.username)
	return nil
}

// ConnectWithCredentials connects with explicit credentials
func (m *Manager) ConnectWithCredentials(port int, username, password string) error {
	m.username = username
	m.password = password
	m.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	resp, err := m.apiCall("GET", "/session", nil)
	if err != nil {
		return fmt.Errorf("cannot connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid credentials")
	}

	log.Printf("✓ Connected to OpenCode at %s", m.baseURL)
	return nil
}

// apiCall makes an authenticated API call
func (m *Manager) apiCall(method, path string, body interface{}) (*http.Response, error) {
	url := m.baseURL + path

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

	if m.username != "" && m.password != "" {
		req.SetBasicAuth(m.username, m.password)
	}

	return http.DefaultClient.Do(req)
}

// IsConnected checks if connected to OpenCode
func (m *Manager) IsConnected() bool {
	return m.baseURL != ""
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

	return "Message sent. Use 'crush-orchestrator messages " + sessionName + "' to check response.", nil
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
			Error     *struct {
				Name string      `json:"name"`
				Data interface{} `json:"data"`
			} `json:"error,omitempty"`
			Time struct {
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
		content := ""
		for _, part := range msg.Parts {
			if part.Type == "text" && part.Text != "" {
				content += part.Text
			}
		}

		errorMsg := ""
		if msg.Info.Error != nil {
			errorMsg = fmt.Sprintf("%v", msg.Info.Error.Data)
		}

		messages = append(messages, Message{
			ID:      msg.Info.ID,
			Role:    msg.Info.Role,
			Content: content,
			Time:    time.Unix(msg.Info.Time.Created/1000, 0),
			Error:   errorMsg,
		})
	}

	return messages, nil
}

// Start is a no-op when connecting to existing server
func (m *Manager) Start(port int) error {
	return m.Connect(port)
}

// Stop is a no-op when connecting to existing server
func (m *Manager) Stop() error {
	return nil
}

// GetPort returns 0 (not managing a server)
func (m *Manager) GetPort() int {
	return 0
}

// IsRunning checks if connected
func (m *Manager) IsRunning() bool {
	return m.IsConnected()
}

// FindAvailablePort finds an available port
func FindAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return true
	}
	resp.Body.Close()
	return false
}
