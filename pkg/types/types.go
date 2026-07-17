// pkg/types/types.go
// Core types for the orchestration system

package types

import (
	"time"
)

// Orchestrator represents the central coordination hub
type Orchestrator struct {
	ID        string    `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	Workspace string    `json:"workspace" yaml:"workspace"`
	Config    Config    `json:"config" yaml:"config"`
	Agents    []Agent   `json:"agents" yaml:"agents"`
	Tasks     []Task    `json:"tasks" yaml:"tasks"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// Config holds orchestrator configuration
type Config struct {
	Host      string        `json:"host" yaml:"host"`
	Port      int           `json:"port" yaml:"port"`
	Auth      AuthConfig    `json:"auth" yaml:"auth"`
	LogLevel  string        `json:"log_level" yaml:"log_level"`
	Heartbeat time.Duration `json:"heartbeat" yaml:"heartbeat"`
	Timeout   time.Duration `json:"timeout" yaml:"timeout"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	Type   string `json:"type" yaml:"type"`
	APIKey string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
}

// Agent represents an AI coding agent
type Agent struct {
	ID           string            `json:"id" yaml:"id"`
	Name         string            `json:"name" yaml:"name"`
	Role         string            `json:"role" yaml:"role"`
	Status       AgentStatus       `json:"status" yaml:"status"`
	Repo         Repository        `json:"repo" yaml:"repo"`
	Model        string            `json:"model" yaml:"model"`
	MaxTokens    int               `json:"max_tokens" yaml:"max_tokens"`
	Template     string            `json:"template" yaml:"template"`
	Capabilities []Capability      `json:"capabilities" yaml:"capabilities"`
	CurrentTask  string            `json:"current_task,omitempty" yaml:"current_task,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	LastSeen     time.Time         `json:"last_seen" yaml:"last_seen"`
	ConnectedAt  time.Time         `json:"connected_at" yaml:"connected_at"`
}

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
	AgentStatusBusy    AgentStatus = "busy"
	AgentStatusIdle    AgentStatus = "idle"
	AgentStatusError   AgentStatus = "error"
)

// Repository represents a code repository
type Repository struct {
	URL    string `json:"url" yaml:"url"`
	Path   string `json:"path" yaml:"path"`
	Branch string `json:"branch" yaml:"branch"`
}

// Capability represents an agent capability
type Capability struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

// Task represents a unit of work
type Task struct {
	ID           string       `json:"id" yaml:"id"`
	Title        string       `json:"title" yaml:"title"`
	Description  string       `json:"description" yaml:"description"`
	Type         TaskType     `json:"type" yaml:"type"`
	Priority     TaskPriority `json:"priority" yaml:"priority"`
	Status       TaskStatus   `json:"status" yaml:"status"`
	Assignee     string       `json:"assignee,omitempty" yaml:"assignee,omitempty"`
	Creator      string       `json:"creator" yaml:"creator"`
	SubTasks     []Task       `json:"sub_tasks,omitempty" yaml:"sub_tasks,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Checkpoints  []Checkpoint `json:"checkpoints,omitempty" yaml:"checkpoints,omitempty"`
	Context      TaskContext  `json:"context" yaml:"context"`
	Tags         []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
	CreatedAt    time.Time    `json:"created_at" yaml:"created_at"`
	StartedAt    *time.Time   `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	DueDate      *time.Time   `json:"due_date,omitempty" yaml:"due_date,omitempty"`
}

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeFeature  TaskType = "feature"
	TaskTypeBugfix   TaskType = "bugfix"
	TaskTypeRefactor TaskType = "refactor"
	TaskTypeDocs     TaskType = "docs"
	TaskTypeTest     TaskType = "test"
	TaskTypeChore    TaskType = "chore"
)

// TaskPriority represents task priority
type TaskPriority string

const (
	TaskPriorityCritical TaskPriority = "critical"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityMedium   TaskPriority = "medium"
	TaskPriorityLow      TaskPriority = "low"
)

// TaskStatus represents task status
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Dependency represents a task dependency
type Dependency struct {
	TaskID string `json:"task_id" yaml:"task_id"`
	Type   string `json:"type" yaml:"type"` // "blocks", "depends_on"
}

// Checkpoint represents a human approval checkpoint
type Checkpoint struct {
	ID                string           `json:"id" yaml:"id"`
	Description       string           `json:"description" yaml:"description"`
	Status            CheckpointStatus `json:"status" yaml:"status"`
	RequiredApprovers []string         `json:"required_approvers,omitempty" yaml:"required_approvers,omitempty"`
	ApprovedBy        []string         `json:"approved_by,omitempty" yaml:"approved_by,omitempty"`
	Comments          []Comment        `json:"comments,omitempty" yaml:"comments,omitempty"`
	BlocksTasks       []string         `json:"blocks_tasks,omitempty" yaml:"blocks_tasks,omitempty"`
	BlocksCompletion  bool             `json:"blocks_completion" yaml:"blocks_completion"`
	CreatedAt         time.Time        `json:"created_at" yaml:"created_at"`
	ResolvedAt        *time.Time       `json:"resolved_at,omitempty" yaml:"resolved_at,omitempty"`
}

// CheckpointStatus represents checkpoint status
type CheckpointStatus string

const (
	CheckpointStatusPending   CheckpointStatus = "pending"
	CheckpointStatusApproved  CheckpointStatus = "approved"
	CheckpointStatusDenied    CheckpointStatus = "denied"
	CheckpointStatusCancelled CheckpointStatus = "cancelled"
)

// Comment represents a comment on a checkpoint
type Comment struct {
	Author    string    `json:"author" yaml:"author"`
	Content   string    `json:"content" yaml:"content"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// TaskContext represents shared context for a task
type TaskContext struct {
	RelatedFiles []string               `json:"related_files,omitempty" yaml:"related_files,omitempty"`
	SharedVars   map[string]interface{} `json:"shared_vars,omitempty" yaml:"shared_vars,omitempty"`
}

// Message represents a message between agents
type Message struct {
	ID        string          `json:"id" yaml:"id"`
	Type      MessageType     `json:"type" yaml:"type"`
	From      string          `json:"from" yaml:"from"`
	To        string          `json:"to,omitempty" yaml:"to,omitempty"`
	Content   string          `json:"content" yaml:"content"`
	Channel   string          `json:"channel" yaml:"channel"`
	ThreadID  string          `json:"thread_id,omitempty" yaml:"thread_id,omitempty"`
	Metadata  MessageMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at" yaml:"created_at"`
}

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeDirect     MessageType = "direct"
	MessageTypeBroadcast  MessageType = "broadcast"
	MessageTypeChannel    MessageType = "channel"
	MessageTypeSystem     MessageType = "system"
	MessageTypeCheckpoint MessageType = "checkpoint"
)

// MessageMetadata represents message metadata
type MessageMetadata struct {
	Attachments []Attachment `json:"attachments,omitempty" yaml:"attachments,omitempty"`
	ReplyTo     string       `json:"reply_to,omitempty" yaml:"reply_to,omitempty"`
	Priority    string       `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	Name    string `json:"name" yaml:"name"`
	Type    string `json:"type" yaml:"type"`
	Path    string `json:"path" yaml:"path"`
	Content string `json:"content,omitempty" yaml:"content,omitempty"`
}

// Workspace represents the shared workspace
type Workspace struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	BasePath    string                 `json:"base_path" yaml:"base_path"`
	Config      WorkspaceConfig        `json:"config" yaml:"config"`
	Files       []WorkspaceFile        `json:"files" yaml:"files"`
	Variables   map[string]interface{} `json:"variables,omitempty" yaml:"variables,omitempty"`
	CreatedAt   time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" yaml:"updated_at"`
}

// WorkspaceConfig holds workspace configuration
type WorkspaceConfig struct {
	AutoSync          bool          `json:"auto_sync" yaml:"auto_sync"`
	SyncInterval      time.Duration `json:"sync_interval" yaml:"sync_interval"`
	MaxFiles          int           `json:"max_files" yaml:"max_files"`
	AllowedExtensions []string      `json:"allowed_extensions,omitempty" yaml:"allowed_extensions,omitempty"`
}

// WorkspaceFile represents a file in the workspace
type WorkspaceFile struct {
	Path       string    `json:"path" yaml:"path"`
	Type       string    `json:"type" yaml:"type"`
	Size       int64     `json:"size" yaml:"size"`
	ModifiedBy string    `json:"modified_by" yaml:"modified_by"`
	ModifiedAt time.Time `json:"modified_at" yaml:"modified_at"`
	Hash       string    `json:"hash" yaml:"hash"`
}
