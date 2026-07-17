import { tool } from "@opencode-ai/plugin"

const API_URL = process.env.ORCHESTRATOR_API || "http://localhost:9800"
const SECRET = process.env.ORCHESTRATOR_SECRET || ""

async function apiCall(path: string, method: string = "GET", body?: any) {
  const url = API_URL + path
  const res = await fetch(url, {
    method,
    headers: {
      "Authorization": "Bearer " + SECRET,
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const error = await res.text()
    throw new Error(`API error ${res.status}: ${error}`)
  }
  return res.json()
}

export default tool({
  description: "Create a new task for the team",
  args: {
    title: tool.schema.string().describe("Task title"),
    description: tool.schema.string().describe("Task description"),
    type: tool.schema.enum(["feature", "bugfix", "refactor", "docs", "test", "chore"]).optional().describe("Task type"),
    priority: tool.schema.enum(["critical", "high", "medium", "low"]).optional().describe("Task priority"),
    assignee: tool.schema.string().optional().describe("Agent ID to assign"),
    dependencies: tool.schema.array(tool.schema.string()).optional().describe("Task IDs this depends on"),
    repo: tool.schema.string().optional().describe("Repository name"),
  },
  async execute(args, context) {
    try {
      const task = await apiCall("/tasks", "POST", {
        title: args.title,
        description: args.description,
        type: args.type || "feature",
        priority: args.priority || "medium",
        assignee: args.assignee,
        creator: context.agent || "unknown",
        dependencies: (args.dependencies || []).map(id => ({ task_id: id, type: "depends_on" })),
        repo: args.repo,
      })
      return `Task created: ${task.id} — ${task.title}`
    } catch (e: any) {
      return `Failed to create task: ${e.message}`
    }
  },
})
