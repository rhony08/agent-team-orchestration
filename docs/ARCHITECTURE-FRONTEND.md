# Frontend Architecture: Agent Team Orchestration TUI

## Overview

The frontend extends OpenCode's Bubble Tea TUI to provide multi-agent orchestration capabilities with real-time visualization of agent teams.

## Design Principles

1. **Familiar Interface**: Extend OpenCode's existing UI patterns
2. **At-a-Glance Status**: Quick understanding of all agent states
3. **Context Preservation**: Human always knows what's happening
4. **Easy Intervention**: Simple checkpoint approval and direct agent access

## Screen Layouts

### 1. Orchestrator Dashboard (Main Screen)

```
┌─────────────────────────────────────────────────────────────────┐
│  ⌬ OpenCode Orchestrator  │  Workspace: microservice-v2         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ AGENT STATUS                                            │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ [●] Tech Lead      │ main  │ task:designing-api    │ 14m │   │
│  │ [●] Backend Dev A  │ main  │ task:implementing-auth│  8m │   │
│  │ [●] Backend Dev B  │ main  │ idle:waiting-input    │  2m │   │
│  │ [○] Frontend Dev   │ main  │ status:offline        │  -- │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ SHARED WORKSPACE                                        │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ 📁 docs/architecture.md      [Modified 5m ago]          │   │
│  │ 📁 tasks/TASK-001.yaml       [Active]                   │   │
│  │ 📁 context/api-contract.json [Tech Lead updated]        │   │
│  │ 📁 messages/2024-01-15/      [23 new]                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ RECENT ACTIVITY                                         │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ [14:32] Backend Dev A → Backend Dev B: "What's the..." │   │
│  │ [14:28] Tech Lead updated: architecture.md              │   │
│  │ [14:25] ⚠️ Checkpoint pending: Approve database schema  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  [Ctrl+N] New Task  [Ctrl+M] Messages  [Ctrl+A] Agents          │
│  [Ctrl+K] Commands  [Ctrl+P] Checkpoint  [Ctrl+C] Quit          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Key Elements:**
- **Header**: Current workspace name, connection status
- **Agent Status Panel**: All agents with role, branch, current task, duration
- **Shared Workspace Panel**: Key files and their status
- **Activity Feed**: Recent messages and events
- **Footer**: Global shortcuts

### 2. Agent Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│  ⌬ Backend Dev A (ID: agent-backend-001)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Status:    [●] Active                                          │
│  Role:      Backend Developer                                   │
│  Repo:      git@github.com:company/api-gateway.git              │
│  Branch:    feature/auth-jwt                                    │
│  Task:      TASK-001 - Implement user authentication            │
│  Duration:  8m 32s                                              │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  CURRENT SESSION                                                │
├─────────────────────────────────────────────────────────────────┤
│  > Implement JWT middleware                                     │
│  > Working on token generation...                               │
│  > Need to check user-service for User model fields             │
│  > Sent query to Backend Dev B                                  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  RECENT FILES                                                   │
├─────────────────────────────────────────────────────────────────┤
│  M  auth/middleware.go                                          │
│  A  auth/token.go                                               │
│  M  go.mod                                                      │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  [i] Intervene  [Enter] Open Session  [Esc] Back                │
└─────────────────────────────────────────────────────────────────┘
```

### 3. Message/Communication View

```
┌─────────────────────────────────────────────────────────────────┐
│  💬 Team Messages                                     [23/156]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ [#general]                                              │   │
│  │                                                         │   │
│  │ [Tech Lead]                                             │   │
│  │ I've updated the API contract. Please review.          │
│  │ 📎 api-contract-v2.yaml                                 │   │
│  │                                                         │   │
│  │ [Backend Dev A]                                         │   │
│  │ Will do. Working on JWT implementation.                │   │
│  │                                                         │   │
│  │ [Backend Dev B]                                         │   │
│  │ @Backend Dev A User model has: id, email, password_hash│   │
│  │ See: user-service/models/user.go                      │   │
│  │                                                         │   │
│  │ [Backend Dev A]                                         │   │
│  │ Thanks! Using that now.                                │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Type message...  [Ctrl+Enter to broadcast to all]      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  [Tab] Switch Channel  [Ctrl+M] Mention Agent  [Esc] Back       │
└─────────────────────────────────────────────────────────────────┘
```

**Features:**
- Channel list (#general, per-agent, per-task)
- Message threading
- File attachments
- Agent mentions (@AgentName)
- Broadcast to all agents

### 4. Shared Workspace Browser

```
┌─────────────────────────────────────────────────────────────────┐
│  📁 Shared Workspace: microservice-v2                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  📂 context/                                                    │
│  ├─ 📄 project-brief.md           [Tech Lead]        [2h ago] │
│  ├─ 📄 architecture.md            [Tech Lead]        [5m ago] │
│  └─ 📄 dependencies.json          [Auto-generated]   [1h ago] │
│                                                                 │
│  📂 tasks/                                                      │
│  ├─ 📄 TASK-001.yaml              [Backend Dev A]    [● Active]│
│  ├─ 📄 TASK-002.yaml              [Unassigned]       [○ Open]  │
│  └─ 📄 TASK-003.yaml              [Frontend Dev]     [✓ Done] │
│                                                                 │
│  📂 messages/                                                   │
│  └─ 📂 2024-01-15/              [23 files]                     │
│                                                                 │
│  📂 agents/                                                     │
│  ├─ 📄 tech-lead.json                                        │
│  ├─ 📄 backend-dev-a.json                                    │
│  └─ 📄 backend-dev-b.json                                    │
│                                                                 │
│  [Enter] Open File  [e] Edit  [d] Diff  [Esc] Back              │
└─────────────────────────────────────────────────────────────────┘
```

### 5. Checkpoint Approval Dialog

```
┌─────────────────────────────────────────────────────────────────┐
│  ⚠️  Checkpoint Pending                                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Task: TASK-001 - Implement user authentication                 │
│  Agent: Backend Dev A                                           │
│                                                                 │
│  Request: Approve database schema changes                       │
│  ─────────────────────────────────────────────────────────────  │
│  Changes to `users` table:                                      │
│    + password_hash VARCHAR(255)                                 │
│    + last_login TIMESTAMP                                       │
│    + email_verified BOOLEAN                                     │
│  ─────────────────────────────────────────────────────────────  │
│  [View Full Diff]                                               │
│                                                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │  Allow   │ │  Allow   │ │   Deny   │ │  Ignore  │           │
│  │   (a)    │ │ Session  │ │   (d)    │ │   (i)    │           │
│  │          │ │   (A)    │ │          │ │          │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
│                                                                 │
│  [Add comment...]                                               │
└─────────────────────────────────────────────────────────────────┘
```

### 6. New Task Creation

```
┌─────────────────────────────────────────────────────────────────┐
│  ➕ Create New Task                                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Title: [Implement rate limiting middleware                    ]│
│                                                                 │
│  Description:                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Add Redis-based rate limiting to API gateway.           │   │
│  │ Limit: 100 req/min per IP.                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Assign to: [Backend Dev A ▼]                                  │
│  Priority:  [High ▼]                                           │
│                                                                 │
│  Dependencies:                                                  │
│  [x] TASK-001 - Depends on auth being complete                 │
│  [ ] TASK-003 - Frontend ready                                 │
│                                                                 │
│  Requires Checkpoint: [✓] Before merging                        │
│                                                                 │
│  ┌─────────────────┐ ┌─────────────────┐                       │
│  │  Create Task    │ │   Cancel        │                       │
│  └─────────────────┘ └─────────────────┘                       │
└─────────────────────────────────────────────────────────────────┘
```

### 7. Command Palette

```
┌─────────────────────────────────────────────────────────────────┐
│  ⌘ Commands                                                     │
├─────────────────────────────────────────────────────────────────┤
│  > create task                                                   │
│                                                                 │
│    create task - Create a new task for an agent                 │
│    create workspace - Initialize new orchestrator workspace     │
│    add agent - Add a new agent to the team                      │
│    broadcast - Send message to all agents                       │
│    sync workspace - Force workspace synchronization             │
│    view checkpoint - Review pending checkpoints                 │
│    export logs - Export session logs                            │
│    settings - Configure orchestrator settings                   │
│                                                                 │
│  [↑↓] Navigate  [Enter] Select  [Esc] Cancel                    │
└─────────────────────────────────────────────────────────────────┘
```

## Key UI Components

### Component Hierarchy

```
App (bubbletea.Model)
├── OrchestratorDashboard
│   ├── Header
│   │   └── StatusBar
│   ├── AgentStatusPanel
│   │   └── AgentCard (xN)
│   ├── WorkspacePanel
│   │   └── FileTree
│   ├── ActivityFeed
│   │   └── ActivityItem (xN)
│   └── Footer
│       └── HelpBar
├── AgentDetailView
│   ├── AgentHeader
│   ├── SessionPreview
│   └── FileChanges
├── MessageView
│   ├── ChannelList
│   ├── MessageList
│   │   └── MessageBubble
│   └── MessageInput
├── WorkspaceBrowser
│   └── FileTree
├── CheckpointDialog
│   ├── DiffViewer
│   └── ActionButtons
├── TaskCreator
│   ├── FormFields
│   └── DependencySelector
└── CommandPalette
    └── SearchResults
```

### Custom Bubble Tea Components

```go
// Agent status card
package components

type AgentCard struct {
    Agent      *types.Agent
    IsSelected bool
    IsOnline   bool
}

func (a AgentCard) View() string {
    status := "●"
    if !a.IsOnline {
        status = "○"
    }

    return lipgloss.JoinVertical(
        lipgloss.Left,
        fmt.Sprintf("%s %s", status, a.Agent.Name),
        fmt.Sprintf("   %s | %s | %s",
            a.Agent.CurrentTask,
            a.Agent.Branch,
            formatDuration(a.Agent.TaskDuration)),
    )
}

// Activity feed item
type ActivityItem struct {
    Timestamp time.Time
    Actor     string
    Action    string
    Details   string
}

func (a ActivityItem) View() string {
    return fmt.Sprintf("[%s] %s %s %s",
        formatTime(a.Timestamp),
        style.Highlight.Render(a.Actor),
        a.Action,
        a.Details)
}

// Message bubble
type MessageBubble struct {
    From      string
    Content   string
    Timestamp time.Time
    IsSelf    bool
}

func (m MessageBubble) View() string {
    bubbleStyle := style.MessageBubble
    if m.IsSelf {
        bubbleStyle = style.MessageBubbleSelf
    }

    return lipgloss.JoinVertical(
        lipgloss.Left,
        fmt.Sprintf("[%s] %s", m.From, formatTime(m.Timestamp)),
        bubbleStyle.Render(m.Content),
    )
}
```

## Navigation Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        DASHBOARD                                │
│                    (Default view)                               │
│                                                                 │
│  [Ctrl+A] Agents List ◄──────────────────────────────┐         │
│  [Ctrl+M] Messages ──────┐                           │         │
│  [Ctrl+W] Workspace ─────┼───┐                       │         │
│  [Ctrl+P] Checkpoints ───┼───┼───┐                   │         │
│  [Ctrl+N] New Task ──────┼───┼───┼───┐               │         │
│  [Ctrl+K] Commands ──────┼───┼───┼───┼───┐           │         │
│                          │   │   │   │   │           │         │
│                          ▼   ▼   ▼   ▼   ▼           ▼         │
│                    ┌─────────────────────────────────────────┐ │
│                    │           MODAL DIALOGS                 │ │
│                    │    (Overlay on dashboard)               │ │
│                    └─────────────────────────────────────────┘ │
│                                                                 │
│  [Enter] on Agent Card ──────────────────────┐                 │
│                                              ▼                 │
│                    ┌─────────────────────────────────────────┐ │
│                    │         AGENT DETAIL VIEW               │ │
│                    │     (Full screen, can intervene)        │ │
│                    └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Keyboard Shortcuts

### Global (Always Available)

| Shortcut | Action |
|----------|--------|
| `Ctrl+C` | Quit |
| `Ctrl+?` | Show help |
| `Ctrl+A` | Show agents list |
| `Ctrl+M` | Show messages |
| `Ctrl+W` | Show workspace browser |
| `Ctrl+P` | Show pending checkpoints |
| `Ctrl+N` | Create new task |
| `Ctrl+K` | Open command palette |
| `Ctrl+S` | Sync workspace |
| `Esc` | Close dialog / Go back |

### Dashboard Navigation

| Shortcut | Action |
|----------|--------|
| `↑/↓` or `j/k` | Navigate panels |
| `Tab` | Next panel |
| `Shift+Tab` | Previous panel |
| `Enter` | Open selected item |
| `r` | Refresh data |

### Agent Detail View

| Shortcut | Action |
|----------|--------|
| `i` | Intervene in session |
| `Enter` | Open agent's OpenCode session |
| `m` | Message this agent |
| `Esc` | Back to dashboard |

### Message View

| Shortcut | Action |
|----------|--------|
| `Tab` | Switch channel |
| `↑/↓` | Scroll messages |
| `Ctrl+M` | Mention agent |
| `Ctrl+B` | Broadcast to all |
| `Ctrl+Enter` | Send message |
| `Esc` | Exit to dashboard |

### Workspace Browser

| Shortcut | Action |
|----------|--------|
| `↑/↓` | Navigate files |
| `Enter` | Open file (view) |
| `e` | Edit file |
| `d` | Show diff |
| `r` | Refresh |
| `/` | Search files |

## Styling Guide

### Color Palette

```go
// Styles matching OpenCode aesthetic
package style

var (
    // Primary colors
    Primary   = lipgloss.Color("#7C3AED") // Violet
    Secondary = lipgloss.Color("#06B6D4") // Cyan
    Success   = lipgloss.Color("#10B981") // Emerald
    Warning   = lipgloss.Color("#F59E0B") // Amber
    Danger    = lipgloss.Color("#EF4444") // Red

    // Status colors
    Online    = lipgloss.Color("#10B981")
    Offline   = lipgloss.Color("#9CA3AF")
    Busy      = lipgloss.Color("#F59E0B")
    Error     = lipgloss.Color("#EF4444")

    // UI colors
    Background = lipgloss.Color("#1F2937")
    Surface    = lipgloss.Color("#374151")
    Border     = lipgloss.Color("#4B5563")
    Text       = lipgloss.Color("#F3F4F6")
    Muted      = lipgloss.Color("#9CA3AF")
)

// Component styles
var (
    Header = lipgloss.NewStyle().
        Bold(true).
        Background(Primary).
        Foreground(lipgloss.Color("#FFFFFF")).
        Padding(0, 1)

    AgentCard = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Border).
        Padding(1)

    AgentCardSelected = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Primary).
        Padding(1)

    MessageBubble = lipgloss.NewStyle().
        Background(Surface).
        Foreground(Text).
        Padding(1, 2).
        Border(lipgloss.RoundedBorder(), false, false, false, true).
        BorderForeground(Secondary)

    CheckpointAlert = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Warning).
        Background(lipgloss.Color("#451A03")).
        Padding(1)
)
```

## State Management

```go
// TUI state model
type Model struct {
    // Current view
    View ViewType

    // Data
    Orchestrator   *types.Orchestrator
    Agents         []types.Agent
    Messages       []types.Message
    Workspace      *types.Workspace
    Checkpoints    []types.Checkpoint
    CurrentTask    *types.Task

    // UI state
    SelectedAgent    int
    SelectedChannel  int
    SelectedFile     int
    MessageInput     textinput.Model
    CommandInput     textinput.Model
    Viewport         viewport.Model

    // Async
    Loading      bool
    Error        error
    Spinner      spinner.Model
}

type ViewType int
const (
    ViewDashboard ViewType = iota
    ViewAgentDetail
    ViewMessages
    ViewWorkspace
    ViewCheckpoint
    ViewTaskCreator
    ViewCommandPalette
)

// Update loop
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    case AgentUpdateMsg:
        return m.handleAgentUpdate(msg)
    case MessageReceivedMsg:
        return m.handleNewMessage(msg)
    case CheckpointPendingMsg:
        return m.showCheckpointDialog(msg)
    }
    return m, nil
}
```

## Integration with OpenCode

### Option 1: Extension (Recommended)

Create an OpenCode extension that adds orchestrator views:

```go
// internal/extensions/orchestrator.go
package orchestrator

func Register(app *app.App) {
    // Add new page to OpenCode's page system
    app.Pages.Add("orchestrator", NewOrchestratorPage())

    // Register keyboard shortcut
    app.KeyMap.Set("orchestrator", "Ctrl+O")

    // Hook into agent tool system
    app.Tools.Register("orchestrator_message", orchestratorMessageTool)
    app.Tools.Register("orchestrator_query", orchestratorQueryTool)
}
```

### Option 2: Standalone TUI

Create a separate TUI that can communicate with OpenCode:

```go
// cmd/orchestrator-tui/main.go
func main() {
    // Connect to orchestrator API
    client := api.NewClient(cfg.OrchestratorURL)

    // Run Bubble Tea app
    m := tui.NewModel(client)
    p := tea.NewProgram(m, tea.WithAltScreen())

    // Launch OpenCode agent sessions on demand
    m.OnAgentOpen = func(agentID string) {
        exec.Command("crush", "-c", agentRepo).Start()
    }

    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Real-time Updates

```go
// File watcher for workspace changes
func watchWorkspace(path string) chan WorkspaceEvent {
    events := make(chan WorkspaceEvent)

    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(path)

    go func() {
        for {
            select {
            case event := <-watcher.Events:
                events <- WorkspaceEvent{
                    Type:    event.Op.String(),
                    Path:    event.Name,
                    Time:    time.Now(),
                }
            }
        }
    }()

    return events
}

// Poll for new messages
func pollMessages(orchestratorURL string) tea.Cmd {
    return func() tea.Msg {
        messages := fetchMessages(orchestratorURL)
        return MessagesUpdateMsg{Messages: messages}
    }
}
```

## File Structure

```
agent-team-orchestration/
├── cmd/
│   └── orchestrator-tui/      # Standalone TUI (optional)
│       └── main.go
├── internal/
│   └── tui/
│       ├── app.go             # Main app model
│       ├── views/
│       │   ├── dashboard.go
│       │   ├── agent_detail.go
│       │   ├── messages.go
│       │   ├── workspace.go
│       │   ├── checkpoint.go
│       │   └── task_creator.go
│       ├── components/
│       │   ├── agent_card.go
│       │   ├── message_bubble.go
│       │   ├── file_tree.go
│       │   ├── activity_feed.go
│       │   └── modal.go
│       ├── style/
│       │   └── theme.go
│       └── api/
│           └── client.go
├── pkg/
│   └── types/
│       └── types.go
└── go.mod
```

## Recommended Implementation Path

1. **Phase 1**: Dashboard with agent status panel
2. **Phase 2**: Add message view
3. **Phase 3**: Add workspace browser
4. **Phase 4**: Add checkpoint dialogs
5. **Phase 5**: Full task creation flow
6. **Phase 6**: Integration with OpenCode as extension
