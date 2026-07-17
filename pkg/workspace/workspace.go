// pkg/workspace/workspace.go
// Shared workspace management

package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yourusername/agent-team-orchestration/pkg/types"
)

// Manager handles workspace operations
type Manager struct {
	basePath string
	config   types.WorkspaceConfig
	mu       sync.RWMutex
	files    map[string]types.WorkspaceFile
}

// NewManager creates a new workspace manager
func NewManager(basePath string, config types.WorkspaceConfig) (*Manager, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"agents", "tasks", "messages", "context", "checkpoints"}
	for _, dir := range subdirs {
		if err := os.MkdirAll(filepath.Join(basePath, dir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	m := &Manager{
		basePath: basePath,
		config:   config,
		files:    make(map[string]types.WorkspaceFile),
	}

	// Load existing files
	if err := m.scanFiles(); err != nil {
		return nil, err
	}

	return m, nil
}

// GetBasePath returns the workspace base path
func (m *Manager) GetBasePath() string {
	return m.basePath
}

// WriteFile writes a file to the workspace
func (m *Manager) WriteFile(path string, content []byte, modifiedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullPath := filepath.Join(m.basePath, path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update file record
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	file := types.WorkspaceFile{
		Path:       path,
		Type:       filepath.Ext(path),
		Size:       info.Size(),
		ModifiedBy: modifiedBy,
		ModifiedAt: time.Now(),
	}

	m.files[path] = file

	return nil
}

// ReadFile reads a file from the workspace
func (m *Manager) ReadFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fullPath := filepath.Join(m.basePath, path)
	return os.ReadFile(fullPath)
}

// DeleteFile deletes a file from the workspace
func (m *Manager) DeleteFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullPath := filepath.Join(m.basePath, path)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	delete(m.files, path)
	return nil
}

// ListFiles lists all files in the workspace
func (m *Manager) ListFiles() []types.WorkspaceFile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make([]types.WorkspaceFile, 0, len(m.files))
	for _, f := range m.files {
		files = append(files, f)
	}
	return files
}

// GetFile gets a specific file info
func (m *Manager) GetFile(path string) (types.WorkspaceFile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, ok := m.files[path]
	return file, ok
}

// SaveAgentState saves an agent's state
func (m *Manager) SaveAgentState(agent *types.Agent) error {
	data, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent state: %w", err)
	}

	path := filepath.Join("agents", agent.ID+".json")
	return m.WriteFile(path, data, agent.ID)
}

// LoadAgentState loads an agent's state
func (m *Manager) LoadAgentState(agentID string) (*types.Agent, error) {
	path := filepath.Join("agents", agentID+".json")
	data, err := m.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent state: %w", err)
	}

	var agent types.Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent state: %w", err)
	}

	return &agent, nil
}

// ListAgents lists all agents in the workspace
func (m *Manager) ListAgents() ([]*types.Agent, error) {
	agentsDir := filepath.Join(m.basePath, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	var agents []*types.Agent
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		agentID := entry.Name()[:len(entry.Name())-5] // Remove .json
		agent, err := m.LoadAgentState(agentID)
		if err != nil {
			continue // Skip invalid agents
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// SaveTask saves a task to the workspace
func (m *Manager) SaveTask(task *types.Task) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	var path string
	if task.Status == types.TaskStatusCompleted || task.Status == types.TaskStatusCancelled {
		path = filepath.Join("tasks", "completed", task.ID+".json")
	} else {
		path = filepath.Join("tasks", "active", task.ID+".json")
	}

	return m.WriteFile(path, data, task.Creator)
}

// LoadTask loads a task from the workspace
func (m *Manager) LoadTask(taskID string) (*types.Task, error) {
	// Try active first
	path := filepath.Join("tasks", "active", taskID+".json")
	data, err := m.ReadFile(path)
	if err != nil {
		// Try completed
		path = filepath.Join("tasks", "completed", taskID+".json")
		data, err = m.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
	}

	var task types.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// ListTasks lists all tasks in the workspace
func (m *Manager) ListTasks(status types.TaskStatus) ([]*types.Task, error) {
	var dir string
	if status == types.TaskStatusCompleted || status == types.TaskStatusCancelled {
		dir = filepath.Join(m.basePath, "tasks", "completed")
	} else {
		dir = filepath.Join(m.basePath, "tasks", "active")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	var tasks []*types.Task
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskID := entry.Name()[:len(entry.Name())-5]
		task, err := m.LoadTask(taskID)
		if err != nil {
			continue
		}

		// Filter by status if specified
		if status != "" && task.Status != status {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// scanFiles scans the workspace and builds the file index
func (m *Manager) scanFiles() error {
	return filepath.Walk(m.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(m.basePath, path)
		if err != nil {
			return err
		}

		m.files[relPath] = types.WorkspaceFile{
			Path:       relPath,
			Type:       filepath.Ext(path),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
		}

		return nil
	})
}
