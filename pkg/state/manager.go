package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager handles state persistence with atomic writes
type Manager struct {
	basePath string
	mu       sync.RWMutex
}

// NewManager creates a new state manager
func NewManager(basePath string) (*Manager, error) {
	dirs := []string{
		"tasks/active",
		"tasks/completed",
		"messages/inbox",
		"messages/archive",
		"checkpoints/pending",
		"checkpoints/resolved",
		"agents",
	}

	for _, dir := range dirs {
		path := filepath.Join(basePath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &Manager{basePath: basePath}, nil
}

// GetBasePath returns the state base path
func (m *Manager) GetBasePath() string {
	return m.basePath
}

// atomicWrite writes data to a file atomically using temp file + rename
func (m *Manager) atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// --- Task Operations ---

// CreateTask creates a new task
func (m *Manager) CreateTask(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Status == "" {
		task.Status = TaskStatusPending
	}

	// Validate dependencies exist
	for _, dep := range task.Dependencies {
		if err := m.taskExists(dep.TaskID); err != nil {
			return fmt.Errorf("dependency %s: %w", dep.TaskID, err)
		}
	}

	// Check for circular dependencies
	if err := m.checkCircularDeps(task); err != nil {
		return err
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	path := filepath.Join(m.basePath, "tasks", "active", task.ID+".json")
	if err := m.atomicWrite(path, data); err != nil {
		return err
	}

	return m.updateSummary()
}

// GetTask returns a task by ID
func (m *Manager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try active first
	path := filepath.Join(m.basePath, "tasks", "active", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Try completed
		path = filepath.Join(m.basePath, "tasks", "completed", id+".json")
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s", id)
		}
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// ListTasks returns tasks with optional filters
func (m *Manager) ListTasks(status TaskStatus, assignee string) ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*Task

	// Scan active tasks
	activeDir := filepath.Join(m.basePath, "tasks", "active")
	if err := m.scanTaskDir(activeDir, &tasks, status, assignee); err != nil {
		return nil, err
	}

	// Scan completed tasks if requested
	if status == TaskStatusCompleted || status == TaskStatusCancelled || status == "" {
		completedDir := filepath.Join(m.basePath, "tasks", "completed")
		if err := m.scanTaskDir(completedDir, &tasks, status, assignee); err != nil {
			return nil, err
		}
	}

	return tasks, nil
}

func (m *Manager) scanTaskDir(dir string, tasks *[]*Task, status TaskStatus, assignee string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}

		if status != "" && task.Status != status {
			continue
		}
		if assignee != "" && task.Assignee != assignee {
			continue
		}

		*tasks = append(*tasks, &task)
	}

	return nil
}

// UpdateTask updates a task
func (m *Manager) UpdateTask(id string, update TaskUpdate) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, err := m.loadTaskUnsafe(id)
	if err != nil {
		return nil, err
	}

	if update.Status != nil {
		task.Status = *update.Status
		if *update.Status == TaskStatusCompleted {
			now := time.Now()
			task.CompletedAt = &now
		}
	}
	if update.Assignee != nil {
		task.Assignee = *update.Assignee
	}
	if update.Priority != nil {
		task.Priority = *update.Priority
	}

	// Move to completed if done
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusCancelled {
		activePath := filepath.Join(m.basePath, "tasks", "active", id+".json")
		completedPath := filepath.Join(m.basePath, "tasks", "completed", id+".json")
		os.Remove(activePath)

		data, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := m.atomicWrite(completedPath, data); err != nil {
			return nil, err
		}
	} else {
		data, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return nil, err
		}
		path := filepath.Join(m.basePath, "tasks", "active", id+".json")
		if err := m.atomicWrite(path, data); err != nil {
			return nil, err
		}
	}

	if err := m.updateSummary(); err != nil {
		return nil, err
	}

	return task, nil
}

func (m *Manager) loadTaskUnsafe(id string) (*Task, error) {
	path := filepath.Join(m.basePath, "tasks", "active", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(m.basePath, "tasks", "completed", id+".json")
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s", id)
		}
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (m *Manager) taskExists(id string) error {
	path := filepath.Join(m.basePath, "tasks", "active", id+".json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	path = filepath.Join(m.basePath, "tasks", "completed", id+".json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return fmt.Errorf("task not found")
}

func (m *Manager) checkCircularDeps(task *Task) error {
	visited := make(map[string]bool)
	var check func(id string) bool
	check = func(id string) bool {
		if visited[id] {
			return true
		}
		visited[id] = true

		t, err := m.loadTaskUnsafe(id)
		if err != nil {
			return false
		}
		for _, dep := range t.Dependencies {
			if dep.TaskID == task.ID {
				return true
			}
			if check(dep.TaskID) {
				return true
			}
		}
		return false
	}

	for _, dep := range task.Dependencies {
		if check(dep.TaskID) {
			return fmt.Errorf("circular dependency detected")
		}
	}
	return nil
}

// --- Message Operations ---

// SendMessage sends a message to an agent
func (m *Manager) SendMessage(msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Sanitize content
	msg.Content = sanitizeContent(msg.Content)

	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return err
	}

	if msg.To == "all" {
		// Broadcast to all agents
		agents, err := m.listAgentsUnsafe()
		if err != nil {
			return err
		}
		for _, agent := range agents {
			path := filepath.Join(m.basePath, "messages", "inbox", agent.ID, msg.ID+".json")
			if err := m.atomicWrite(path, data); err != nil {
				return err
			}
		}
	} else {
		path := filepath.Join(m.basePath, "messages", "inbox", msg.To, msg.ID+".json")
		if err := m.atomicWrite(path, data); err != nil {
			return err
		}
	}

	return m.updateSummary()
}

// GetMessages returns messages for an agent
func (m *Manager) GetMessages(agentID string, limit int, ack bool) ([]*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if limit <= 0 {
		limit = 50
	}

	dir := filepath.Join(m.basePath, "messages", "inbox", agentID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var messages []*Message
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		messages = append(messages, &msg)

		if len(messages) >= limit {
			break
		}
	}

	// Acknowledge (archive) messages if requested
	if ack && len(messages) > 0 {
		archiveDir := filepath.Join(m.basePath, "messages", "archive", agentID)
		os.MkdirAll(archiveDir, 0755)

		for _, msg := range messages {
			src := filepath.Join(dir, msg.ID+".json")
			dst := filepath.Join(archiveDir, msg.ID+".json")
			os.Rename(src, dst)
		}
	}

	return messages, nil
}

// --- Checkpoint Operations ---

// CreateCheckpoint creates a new checkpoint
func (m *Manager) CreateCheckpoint(cp *Checkpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cp.ID == "" {
		cp.ID = uuid.New().String()
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	if cp.Status == "" {
		cp.Status = CheckpointStatusPending
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(m.basePath, "checkpoints", "pending", cp.ID+".json")
	if err := m.atomicWrite(path, data); err != nil {
		return err
	}

	return m.updateSummary()
}

// GetCheckpoint returns a checkpoint by ID
func (m *Manager) GetCheckpoint(id string) (*Checkpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.basePath, "checkpoints", "pending", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(m.basePath, "checkpoints", "resolved", id+".json")
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("checkpoint not found: %s", id)
		}
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

// ListCheckpoints returns pending checkpoints
func (m *Manager) ListCheckpoints() ([]*Checkpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dir := filepath.Join(m.basePath, "checkpoints", "pending")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var checkpoints []*Checkpoint
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}

		checkpoints = append(checkpoints, &cp)
	}

	return checkpoints, nil
}

// ResolveCheckpoint resolves a checkpoint (approve or deny)
func (m *Manager) ResolveCheckpoint(id string, approved bool, reason string) (*Checkpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pendingPath := filepath.Join(m.basePath, "checkpoints", "pending", id+".json")
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		return nil, fmt.Errorf("checkpoint not found: %s", id)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}

	if cp.Status != CheckpointStatusPending {
		return nil, fmt.Errorf("checkpoint already resolved")
	}

	now := time.Now()
	cp.ResolvedAt = &now
	if approved {
		cp.Status = CheckpointStatusApproved
	} else {
		cp.Status = CheckpointStatusDenied
		cp.Reason = reason
	}

	// Move to resolved
	resolvedPath := filepath.Join(m.basePath, "checkpoints", "resolved", id+".json")
	newData, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := m.atomicWrite(resolvedPath, newData); err != nil {
		return nil, err
	}
	os.Remove(pendingPath)

	if err := m.updateSummary(); err != nil {
		return nil, err
	}

	return &cp, nil
}

// --- Agent Operations ---

// RegisterAgent registers an agent
func (m *Manager) RegisterAgent(agent *Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(m.basePath, "agents", agent.ID+".json")
	return m.atomicWrite(path, data)
}

// ListAgents returns all registered agents
func (m *Manager) ListAgents() ([]*Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.listAgentsUnsafe()
}

func (m *Manager) listAgentsUnsafe() ([]*Agent, error) {
	dir := filepath.Join(m.basePath, "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []*Agent
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var agent Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

// --- Summary ---

// updateSummary updates the lightweight summary file
func (m *Manager) updateSummary() error {
	summary := Summary{
		Version:   "1.0.0",
		UpdatedAt: time.Now(),
	}

	// Count active tasks
	activeDir := filepath.Join(m.basePath, "tasks", "active")
	if entries, err := os.ReadDir(activeDir); err == nil {
		summary.Stats.ActiveTasks = len(entries)
	}

	// Count completed tasks
	completedDir := filepath.Join(m.basePath, "tasks", "completed")
	if entries, err := os.ReadDir(completedDir); err == nil {
		summary.Stats.CompletedTasks = len(entries)
	}

	// Count pending checkpoints
	cpDir := filepath.Join(m.basePath, "checkpoints", "pending")
	if entries, err := os.ReadDir(cpDir); err == nil {
		summary.Stats.PendingCheckpoints = len(entries)
	}

	// Count agents
	agentDir := filepath.Join(m.basePath, "agents")
	if entries, err := os.ReadDir(agentDir); err == nil {
		summary.Stats.TotalAgents = len(entries)
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(m.basePath, "state.json")
	return m.atomicWrite(path, data)
}

// GetSummary returns the current summary
func (m *Manager) GetSummary() (*Summary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := filepath.Join(m.basePath, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &Summary{Version: "1.0.0"}, nil
	}

	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return &Summary{Version: "1.0.0"}, nil
	}

	return &summary, nil
}

// sanitizeContent prevents prompt injection in messages
func sanitizeContent(content string) string {
	// Wrap in message tags to delineate from instructions
	return fmt.Sprintf("<message>%s</message>", content)
}
