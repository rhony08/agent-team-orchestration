package state

import "time"

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

// Task represents a unit of work
type Task struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Type         TaskType     `json:"type,omitempty"`
	Priority     TaskPriority `json:"priority,omitempty"`
	Status       TaskStatus   `json:"status"`
	Assignee     string       `json:"assignee,omitempty"`
	Creator      string       `json:"creator"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Repo         string       `json:"repo,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
}

// Dependency represents a task dependency
type Dependency struct {
	TaskID string `json:"task_id"`
	Type   string `json:"type"` // "blocks", "depends_on"
}

// TaskUpdate represents partial updates to a task
type TaskUpdate struct {
	Status   *TaskStatus   `json:"status,omitempty"`
	Assignee *string       `json:"assignee,omitempty"`
	Priority *TaskPriority `json:"priority,omitempty"`
}

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeStatusUpdate     MessageType = "status_update"
	MessageTypeQuestion         MessageType = "question"
	MessageTypeDependencyAlert  MessageType = "dependency_alert"
	MessageTypeTaskAssignment   MessageType = "task_assignment"
	MessageTypeBlocker          MessageType = "blocker"
)

// Message represents a message between agents
type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Content   string      `json:"content"`
	CreatedAt time.Time   `json:"created_at"`
}

// CheckpointStatus represents checkpoint status
type CheckpointStatus string

const (
	CheckpointStatusPending  CheckpointStatus = "pending"
	CheckpointStatusApproved CheckpointStatus = "approved"
	CheckpointStatusDenied   CheckpointStatus = "denied"
	CheckpointStatusTimedOut CheckpointStatus = "timed_out"
)

// CheckpointType represents the type of checkpoint
type CheckpointType string

const (
	CheckpointTypePreCommit     CheckpointType = "pre_commit"
	CheckpointTypePrePush       CheckpointType = "pre_push"
	CheckpointTypeSchemaChange  CheckpointType = "schema_change"
	CheckpointTypeBreakingChange CheckpointType = "breaking_change"
	CheckpointTypeDestructive   CheckpointType = "destructive"
)

// Checkpoint represents a human approval checkpoint
type Checkpoint struct {
	ID            string           `json:"id"`
	Type          CheckpointType   `json:"type"`
	Description   string           `json:"description"`
	Status        CheckpointStatus `json:"status"`
	Requester     string           `json:"requester"`
	AffectedRepos []string         `json:"affected_repos,omitempty"`
	Reason        string           `json:"reason,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	ResolvedAt    *time.Time       `json:"resolved_at,omitempty"`
}

// AgentStatus represents agent status
type AgentStatus string

const (
	AgentStatusActive AgentStatus = "active"
	AgentStatusIdle   AgentStatus = "idle"
	AgentStatusBusy   AgentStatus = "busy"
	AgentStatusFailed AgentStatus = "failed"
)

// Agent represents a registered agent
type Agent struct {
	ID     string      `json:"id"`
	Type   string      `json:"type"`
	Repo   string      `json:"repo"`
	Status AgentStatus `json:"status"`
}

// Summary is a lightweight state summary
type Summary struct {
	Version   string      `json:"version"`
	UpdatedAt time.Time   `json:"updated_at"`
	Stats     Stats       `json:"stats"`
}

// Stats holds aggregate statistics
type Stats struct {
	ActiveTasks        int `json:"active_tasks"`
	CompletedTasks     int `json:"completed_tasks"`
	PendingCheckpoints int `json:"pending_checkpoints"`
	TotalAgents        int `json:"total_agents"`
}
