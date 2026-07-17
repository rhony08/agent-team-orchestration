# Agent Team Orchestration - Example

This example demonstrates how to use the orchestration system to coordinate
multiple AI agents across different repositories.

## Prerequisites

- Go 1.21+
- OpenCode installed (`curl -fsSL https://opencode.ai/install | bash`)
- Git repositories to orchestrate

## Setup

### 1. Build the orchestrator

```bash
go build -o crush-orchestrator ./cmd/orchestrator/
```

### 2. Create sample repositories (for testing)

```bash
mkdir -p /tmp/example-repos/{api-gateway,user-service,notification-service}

# Initialize git repos
for repo in api-gateway user-service notification-service; do
  cd /tmp/example-repos/$repo
  git init
  echo "# $repo" > README.md
  git add . && git commit -m "Initial commit"
  cd -
done
```

### 3. Initialize orchestration workspace

```bash
./crush-orchestrator init my-project \
  --repos /tmp/example-repos/api-gateway,/tmp/example-repos/user-service,/tmp/example-repos/notification-service
```

This creates:
- `.orchestrator/` directory with state files
- `.opencode/` directories in each repo with plugin, tools, and agent configs

### 4. Start orchestration

```bash
./crush-orchestrator start
```

This will:
- Start the HTTP API server on port 9800
- Spawn OpenCode instances in each repository
- Display status of all agents

## Usage

Once running, you can interact with the agents through OpenCode:

### Tech Lead Agent (Primary)

The Tech Lead agent coordinates work across repositories:

```
# In any of the repo directories, OpenCode is running
# Switch to tech-lead agent (Tab key)

> Implement JWT authentication across api-gateway, user-service, and notification-service

# Tech Lead will:
# 1. Create tasks with task_create
# 2. Identify dependencies
# 3. Assign to backend-dev agents
# 4. Monitor progress
```

### Backend Developer Agent (Subagent)

Backend agents work on specific tasks:

```
# Switch to backend-dev or use @backend-dev

> Implement the JWT middleware in api-gateway

# Backend Dev will:
# 1. Check assigned tasks via sync_workspace
# 2. Implement the changes
# 3. Request checkpoint before committing
# 4. Report progress via team_message
```

### Custom Tools

The following tools are available to all agents:

| Tool | Description |
|------|-------------|
| `team_message` | Send messages between agents |
| `task_create` | Create new tasks |
| `request_checkpoint` | Request human approval |
| `sync_workspace` | Query orchestration state |

## Checkpoint Flow

When an agent needs approval:

1. Agent calls `request_checkpoint` with description
2. Checkpoint appears in terminal where `crush-orchestrator` is running
3. User approves (y) or denies (n + reason)
4. Agent receives decision and continues

## Architecture

```
┌─────────────────────────────────────────────┐
│           Go Wrapper (CLI)                   │
│  crush-orchestrator init/start/stop/status   │
│                                              │
│  ┌──────────────┐  ┌──────────────┐         │
│  │ HTTP API     │  │ Process      │         │
│  │ (port 9800)  │  │ Manager      │         │
│  └──────────────┘  └──────────────┘         │
└─────────────────┬───────────────────────────┘
                  │ HTTP + Bearer Token
                  │
┌─────────────────┴───────────────────────────┐
│         OpenCode Instances                   │
│                                              │
│  ┌──────────────────────────────────────┐   │
│  │  Orchestration Plugin (TS)           │   │
│  │  - Custom tools                      │   │
│  │  - Agent configs                     │   │
│  └──────────────────────────────────────┘   │
│                                              │
│  api-gateway (9801)  user-service (9802)    │
│  notification-service (9803)                 │
└──────────────────────────────────────────────┘
```

## State Structure

```
.orchestrator/
├── config.json           # Project configuration
├── auth.key              # Shared secret (gitignored)
├── state.json            # Lightweight summary
├── tasks/
│   ├── active/           # Active task files
│   └── completed/        # Completed task files
├── messages/
│   ├── inbox/            # Per-agent inboxes
│   └── archive/          # Old messages
├── checkpoints/
│   ├── pending/          # Awaiting approval
│   └── resolved/         # Approved/denied
└── agents/               # Registered agents
```

## Stopping

Press `Ctrl+C` to stop all agents and the API server.

Or from another terminal:

```bash
./crush-orchestrator stop
```

## Troubleshooting

### "opencode not found in PATH"

Install OpenCode:
```bash
curl -fsSL https://opencode.ai/install | bash
```

### "Orchestrator not running"

Make sure you've started the orchestrator:
```bash
./crush-orchestrator start
```

### Port conflicts

If port 9800 is in use, specify a different port:
```bash
./crush-orchestrator start --port 9900
```
