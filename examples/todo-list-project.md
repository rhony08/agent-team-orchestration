# Sample Project: Todo List App

A multi-repository todo list application to test the orchestration system.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Web Frontend                        │
│                    (React + TypeScript)                   │
│                    Port: 3000                            │
└─────────────────────┬───────────────────────────────────┘
                      │ API calls
                      ▼
┌─────────────────────────────────────────────────────────┐
│                      API Server                          │
│                    (Node.js + Express)                   │
│                    Port: 4000                            │
└─────────────────────┬───────────────────────────────────┘
                      │ Database queries
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    Database Service                      │
│                    (PostgreSQL + Prisma)                 │
│                    Port: 5432                            │
└─────────────────────────────────────────────────────────┘
```

## Repositories

| Repo | Description | Tech Stack |
|------|-------------|------------|
| `todo-api` | REST API server | Node.js, Express, TypeScript |
| `todo-web` | Web frontend | React, TypeScript, Vite |
| `todo-db` | Database schema & migrations | Prisma, PostgreSQL |

## Setup Instructions

### 1. Create the repositories

```bash
mkdir -p ~/todo-project/{todo-api,todo-web,todo-db}
```

### 2. Initialize each repository

```bash
# API
cd ~/todo-project/todo-api
git init
npm init -y
npm install express typescript ts-node @types/express @types/node
npx tsc --init

# Web
cd ~/todo-project/todo-web
git init
npm create vite@latest . -- --template react-ts
npm install

# DB
cd ~/todo-project/todo-db
git init
npm init -y
npm install prisma @prisma/client
npx prisma init
```

### 3. Initialize orchestration

```bash
crush-orchestrator init todo-app --repos ~/todo-project/todo-api,~/todo-project/todo-web,~/todo-project/todo-db
```

### 4. Start orchestration

```bash
crush-orchestrator start
```

## File Structure

Each repository will have the following structure after implementation:

### todo-api/
```
src/
├── index.ts              # Entry point
├── routes/
│   └── todos.ts          # Todo routes
├── models/
│   └── todo.ts           # Todo model
└── middleware/
    └── validate.ts       # Validation middleware
package.json
tsconfig.json
```

### todo-web/
```
src/
├── App.tsx               # Main app component
├── components/
│   ├── TodoList.tsx      # Todo list component
│   ├── TodoItem.tsx      # Single todo item
│   └── AddTodo.tsx       # Add todo form
├── hooks/
│   └── useTodos.ts       # Todo hook
├── api/
│   └── todos.ts          # API client
└── types/
    └── todo.ts           # TypeScript types
package.json
tsconfig.json
```

### todo-db/
```
prisma/
├── schema.prisma         # Database schema
└── migrations/           # Database migrations
package.json
```

## API Specification

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/todos | List all todos |
| GET | /api/todos/:id | Get a todo |
| POST | /api/todos | Create a todo |
| PATCH | /api/todos/:id | Update a todo |
| DELETE | /api/todos/:id | Delete a todo |

### Data Model

```typescript
interface Todo {
  id: string;
  title: string;
  description?: string;
  completed: boolean;
  createdAt: Date;
  updatedAt: Date;
}
```

### Example Requests

**Create Todo:**
```json
POST /api/todos
{
  "title": "Buy groceries",
  "description": "Milk, eggs, bread"
}
```

**Update Todo:**
```json
PATCH /api/todos/123
{
  "completed": true
}
```

## Implementation Tasks

Use these tasks with the orchestration system:

### Task 1: Database Schema (todo-db)
```
Title: Create Prisma schema for todos
Description: Define the Todo model in Prisma schema with fields: id, title, description, completed, createdAt, updatedAt
Assignee: todo-db
Priority: high
```

### Task 2: API Endpoints (todo-api)
```
Title: Implement REST API for todos
Description: Create Express routes for CRUD operations on todos. Use Prisma client for database access.
Assignee: todo-api
Priority: high
Dependencies: Task 1
```

### Task 3: API Client (todo-web)
```
Title: Create API client for todo service
Description: Create TypeScript API client with functions for all todo endpoints
Assignee: todo-web
Priority: medium
Dependencies: Task 2
```

### Task 4: Todo Components (todo-web)
```
Title: Build todo UI components
Description: Create React components: TodoList, TodoItem, AddTodo form with proper TypeScript types
Assignee: todo-web
Priority: medium
Dependencies: Task 3
```

### Task 5: Integration (all)
```
Title: End-to-end integration test
Description: Test the full flow: create todo via UI, verify in database, mark as complete
Assignee: todo-api
Priority: low
Dependencies: Task 4
```

## Testing with Orchestration

### Step 1: Start orchestration
```bash
crush-orchestrator start
```

### Step 2: Create tasks
```bash
# Task 1: Database schema
crush-orchestrator task create \
  --title "Create Prisma schema" \
  --description "Define Todo model with id, title, description, completed, createdAt, updatedAt" \
  --assignee todo-db \
  --priority high

# Task 2: API endpoints
crush-orchestrator task create \
  --title "Implement REST API" \
  --description "Create Express routes for CRUD operations" \
  --assignee todo-api \
  --priority high
```

### Step 3: Send work to instances
```bash
# Tell todo-db to create the schema
crush-orchestrator send todo-db "Create a Prisma schema for a Todo model with fields: id (UUID), title (String), description (String, optional), completed (Boolean, default false), createdAt (DateTime), updatedAt (DateTime). Include a migration."

# After schema is done, tell todo-api to build the API
crush-orchestrator send todo-api "Create a REST API with Express for todos. Endpoints: GET /api/todos, GET /api/todos/:id, POST /api/todos, PATCH /api/todos/:id, DELETE /api/todos/:id. Use Prisma client for database access."
```

### Step 4: Handle checkpoints
```bash
# List pending checkpoints
crush-orchestrator checkpoint list

# Approve when ready
crush-orchestrator checkpoint approve abc123
```

### Step 5: Check status
```bash
# See what's happening
crush-orchestrator status

# See all tasks
crush-orchestrator task list
```

## Expected Workflow

1. **User** starts orchestration
2. **User** sends "create database schema" to todo-db
3. **todo-db** creates Prisma schema and migrations
4. **todo-db** requests checkpoint: "Ready to commit schema"
5. **User** approves checkpoint
6. **User** sends "build REST API" to todo-api
7. **todo-api** creates Express routes using Prisma
8. **todo-api** requests checkpoint: "API ready for testing"
9. **User** approves checkpoint
10. **User** sends "create API client" to todo-web
11. **todo-web** creates TypeScript API client
12. **User** sends "build UI components" to todo-web
13. **todo-web** creates React components
14. **todo-web** requests checkpoint: "UI ready for review"
15. **User** approves checkpoint
16. Done! Full stack todo app ready

## Common Issues

### Port conflicts
If you get port errors, the system will automatically find an available port.

### Instance not responding
Check if OpenCode is installed:
```bash
opencode --version
```

### Checkpoint timeout
Checkpoints timeout after 5 minutes. Approve them quickly or restart the task.
