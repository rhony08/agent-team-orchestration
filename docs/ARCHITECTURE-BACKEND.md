# Backend Architecture: Agent Team Orchestration

## Overview

The Agent Team Orchestration system enables multiple OpenCode instances to coordinate across different repositories (microservices) through a central hub with shared workspace.

## Core Components

### 1. Orchestrator Hub (`cmd/orchestrator/`)

**Purpose**: Central coordination point for all agents

```go
// Core orchestrator structure
type Orchestrator struct {
    ID          string                    // Unique orchestrator ID
    Workspace   *Workspace               // Shared workspace reference
    Agents      map[string]*Agent        // Registered agents
    MessageBus  *MessageBus              // Communication bus
    State       *OrchestratorState       // Persistent state
    Config      *OrchestratorConfig      // Configuration
}

// Main responsibilities:
// - Agent lifecycle management (register, unregister, health checks)
// - Message routing between agents
// - Task decomposition and assignment
// - Human checkpoint coordination
// - State synchronization
```

### 2. Agent Client (`cmd/agent/`)

**Purpose**: Wrapper around OpenCode that enables team coordination

```go
type Agent struct {
    ID            string              // Unique agent ID
    Role          AgentRole           // Tech Lead, Backend Dev, etc.
    Repo          Repository          // Assigned repository
    Orchestrator  string              // Connected orchestrator address
    State         AgentState          // Current state
    Workspace     *WorkspaceClient    // Shared workspace client
    MessageBus    *MessageBusClient   // Communication client
}

// Responsibilities:
// - Connect to orchestrator on startup
// - Report status and progress
// - Receive and execute tasks
// - Communicate with other agents via hub
// - Sync workspace state
```

### 3. Shared Workspace (`pkg/workspace/`)

**Purpose**: File-based shared memory for agent coordination

```
~/.crush/orchestrator/
├── workspaces/
│   └── {workspace-id}/
│       ├── meta.json              # Workspace metadata
│       ├── agents/                # Agent state files
│       │   ├── {agent-id}.json
│       │   └── {agent-id}.json
│       ├── messages/              # Message history
│       │   ├── 2024-01-15/
│       │   │   ├── 001-welcome.json
│       │   │   └── 002-task-assignment.json
│       │   └── 2024-01-16/
│       ├── context/               # Shared context
│       │   ├── project-brief.md
│       │   ├── architecture.md
│       │   └── dependencies.json
│       ├── tasks/                 # Task definitions
│       │   ├── active/
│       │   └── completed/
│       └── checkpoints/           # Human approval checkpoints
│           ├── pending/
│           └── approved/
```

### 4. Message Bus (`pkg/bus/`)

**Purpose**: Communication layer between agents

```go
// Message types
type MessageType string
const (
    MessageTypeTask        MessageType = "task"
    MessageTypeStatus      MessageType = "status"
    MessageTypeQuery       MessageType = "query"
    MessageTypeResponse    MessageType = "response"
    MessageTypeBroadcast   MessageType = "broadcast"
    MessageTypeCheckpoint  MessageType = "checkpoint"
)

type Message struct {
    ID        string      `json:"id"`
    Type      MessageType `json:"type"`
    From      string      `json:"from"`
    To        string      `json:"to"`           // "" = broadcast
    Timestamp time.Time   `json:"timestamp"`
    Payload   interface{} `json:"payload"`
    Metadata  Metadata    `json:"metadata"`
}

// Transport options:
// - Unix sockets (local)
// - TCP (remote)
// - WebSocket (browser/extension)
// - Files (fallback, polling-based)
```

### 5. State Manager (`pkg/state/`)

**Purpose**: File-based state persistence

```go
// State files are JSON/YAML for human readability
type StateManager struct {
    BasePath string
}

// Writes state to files, watches for changes
// Conflict resolution: last-write-wins with timestamps
// Locking: file-based locks or in-memory with sync
```

## Communication Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Agent A   │     │ Orchestrator│     │   Agent B   │
│  (Backend)  │     │    Hub      │     │  (Frontend) │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ 1. Register       │                   │
       │──────────────────>│                   │
       │                   │                   │
       │                   │ 2. Register       │
       │                   │<──────────────────│
       │                   │                   │
       │ 3. Task Request   │                   │
       │──────────────────>│                   │
       │                   │                   │
       │                   │ 4. Query Context  │
       │                   │<──────────────────│
       │                   │                   │
       │ 5. Provide Context│                   │
       │<──────────────────│                   │
       │                   │                   │
       │                   │ 6. Task Complete  │
       │                   │──────────────────>│
       │                   │                   │
```

## API Design

### Orchestrator API (HTTP/REST or gRPC)

```go
// HTTP REST API
type OrchestratorAPI interface {
    // Agent management
    POST   /api/v1/agents/register          // Register new agent
    POST   /api/v1/agents/{id}/heartbeat    // Health check
    DELETE /api/v1/agents/{id}               // Unregister
    GET    /api/v1/agents                    // List agents

    // Messaging
    POST   /api/v1/messages                  // Send message
    GET    /api/v1/messages                  // Get messages (polling)
    GET    /api/v1/messages/stream           // WebSocket stream

    // Workspace
    GET    /api/v1/workspace                 // Get workspace state
    POST   /api/v1/workspace/sync            // Sync changes
    GET    /api/v1/workspace/files/{path}    // Get file
    PUT    /api/v1/workspace/files/{path}    // Update file

    // Tasks
    POST   /api/v1/tasks                     // Create task
    GET    /api/v1/tasks/{id}                // Get task
    POST   /api/v1/tasks/{id}/assign         // Assign to agent
    POST   /api/v1/tasks/{id}/complete       // Mark complete

    // Checkpoints
    GET    /api/v1/checkpoints               // List pending checkpoints
    POST   /api/v1/checkpoints/{id}/approve  // Approve
    POST   /api/v1/checkpoints/{id}/deny     // Deny
}
```

### Agent-to-Agent Protocol

```go
// High-level operations
type AgentProtocol interface {
    // Direct messaging
    SendMessage(to string, msg Message) error
    Broadcast(msg Message) error

    // Task coordination
    RequestTask() (*Task, error)
    SubmitTaskResult(taskID string, result TaskResult) error
    QueryAgent(agentID string, query Query) (*Response, error)

    // Workspace sync
    SyncWorkspace() error
    UpdateSharedContext(key string, value interface{}) error
    GetSharedContext(key string) (interface{}, error)

    // Checkpoint handling
    RequestCheckpoint(checkpoint Checkpoint) error
    WaitForCheckpoint(checkpointID string) (bool, error)
}
```

## Data Models

### Agent Definition

```yaml
# ~/.crush/orchestrator/workspaces/{id}/agents/tech-lead.json
id: "agent-tech-lead-001"
name: "Tech Lead"
role: "tech-lead"
model: "claude-3.7-sonnet"
repo:
  url: "git@github.com:company/api-gateway.git"
  branch: "main"
status: "active"
capabilities:
  - "architecture-design"
  - "code-review"
  - "task-delegation"
templates:
  system_prompt: "You are a Tech Lead responsible for..."
  custom_commands:
    - "design-api"
    - "review-pr"
    - "delegate-task"
created_at: "2024-01-15T10:00:00Z"
last_seen: "2024-01-15T14:30:00Z"
```

### Task Definition

```yaml
# ~/.crush/orchestrator/workspaces/{id}/tasks/active/TASK-001.yaml
id: "TASK-001"
title: "Implement user authentication"
description: |
  Create JWT-based authentication for the API gateway.
  Must coordinate with user-service (Agent B).

type: "feature"
priority: "high"
status: "in_progress"

assignee: "agent-backend-dev-001"
creator: "agent-tech-lead-001"

dependencies:
  - task: "TASK-000"
    type: "blocks"
  - agent: "agent-backend-dev-002"
    type: "coordination"
    reason: "Shared database schema"

context:
  related_files:
    - "docs/auth-requirements.md"
    - "user-service/openapi.yaml"
  shared_vars:
    jwt_secret: "${VAULT:jwt_secret}"
    token_expiry: "24h"

checkpoints:
  - id: "CHK-001"
    description: "Approve database schema changes"
    status: "pending"
  - id: "CHK-002"
    description: "Approve API contract"
    status: "pending"

timeline:
  created: "2024-01-15T10:00:00Z"
  started: "2024-01-15T10:30:00Z"
  due: "2024-01-16T18:00:00Z"
```

### Message Format

```json
{
  "id": "msg-uuid-123",
  "type": "query",
  "from": "agent-backend-dev-001",
  "to": "agent-backend-dev-002",
  "timestamp": "2024-01-15T14:30:00Z",
  "payload": {
    "question": "What fields does the User model have?",
    "context": "I'm implementing authentication"
  },
  "metadata": {
    "priority": "normal",
    "reply_to": "msg-uuid-122",
    "workspace_id": "ws-microservice-v2"
  }
}
```

## Integration with OpenCode

### Approach 1: Extension/Plugin

```go
// OpenCode extension that adds orchestration capabilities
package orchestration

// Hook into OpenCode's agent tool system
type OrchestrationExtension struct {
    Orchestrator *Orchestrator
}

// Register new tools:
// - @orchestrator.register - Join team
// - @orchestrator.message - Send message to agent
// - @orchestrator.query - Query shared workspace
// - @orchestrator.checkpoint - Request human approval
// - @orchestrator.sync - Sync with team
```

### Approach 2: Wrapper Command

```bash
# New CLI that wraps OpenCode
crush-orchestrator init-workspace my-project
crush-orchestrator add-agent --role=backend-dev --repo=./service-a
crush-orchestrator start

# Or as subcommand
crush orchestrate --workspace=my-project
```

### Approach 3: MCP Server

```go
// Implement as MCP (Model Context Protocol) server
// OpenCode connects to orchestrator via MCP
type OrchestratorMCPServer struct {
    // Implements MCP protocol
    // Provides tools:
    // - team_message
    // - workspace_query
    // - request_checkpoint
}
```

## Recommended Architecture

**Hybrid Approach**:
1. **Orchestrator**: Standalone Go service ( Approach 2 )
2. **Agent**: OpenCode + extension ( Approach 1 + 3 )
3. **Communication**: File-based with optional TCP/WebSocket for real-time

## File Structure

```
agent-team-orchestration/
├── cmd/
│   ├── orchestrator/          # Main orchestrator CLI
│   │   └── main.go
│   └── agent/                 # Agent wrapper CLI
│       └── main.go
├── pkg/
│   ├── workspace/             # Shared workspace management
│   │   ├── workspace.go
│   │   ├── file_watcher.go
│   │   └── sync.go
│   ├── bus/                   # Message bus
│   │   ├── bus.go
│   │   ├── transport/
│   │   │   ├── file.go
│   │   │   ├── tcp.go
│   │   │   └── websocket.go
│   │   └── message.go
│   ├── state/                 # State management
│   │   ├── manager.go
│   │   └── lock.go
│   ├── api/                   # HTTP/gRPC API
│   │   ├── server.go
│   │   ├── handlers.go
│   │   └── middleware.go
│   └── protocol/              # Agent protocol
│       ├── client.go
│       └── types.go
├── internal/
│   ├── config/                # Configuration
│   ├── templates/             # Agent templates
│   └── auth/                  # Authentication
├── templates/                 # Built-in agent templates
│   ├── tech-lead.yaml
│   ├── backend-dev.yaml
│   └── frontend-dev.yaml
├── web/                       # Optional web dashboard
│   └── ...
├── go.mod
├── go.sum
└── README.md
```

## Security Considerations

1. **Authentication**: API keys, mTLS, or GitHub tokens
2. **Authorization**: Role-based access control
3. **Secrets**: Vault integration for sensitive data
4. **Sandbox**: Optional sandbox for agent execution
5. **Audit**: Log all agent actions for compliance

## Next Steps

1. Define MVP scope (2-3 agents, single workspace)
2. Implement file-based message bus
3. Create basic orchestrator CLI
4. Build OpenCode extension
5. Test with sample microservices
