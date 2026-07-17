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
  description: "Query orchestration state - get tasks, messages, checkpoints, or status",
  args: {
    action: tool.schema.enum(["status", "get_tasks", "get_messages", "get_checkpoints"]).describe("What to query"),
    status: tool.schema.enum(["pending", "assigned", "in_progress", "blocked", "completed", "cancelled"]).optional().describe("Filter tasks by status"),
    assignee: tool.schema.string().optional().describe("Filter tasks by assignee"),
  },
  async execute(args, context) {
    try {
      switch (args.action) {
        case "status": {
          const status = await apiCall("/status") as any
          return JSON.stringify({
            running: status.running,
            agents: status.agents?.length || 0,
            active_tasks: status.stats?.active_tasks || 0,
            completed_tasks: status.stats?.completed_tasks || 0,
            pending_checkpoints: status.stats?.pending_checkpoints || 0,
          }, null, 2)
        }

        case "get_tasks": {
          let path = "/tasks"
          const params = []
          if (args.status) params.push(`status=${args.status}`)
          if (args.assignee) params.push(`assignee=${args.assignee}`)
          if (params.length) path += "?" + params.join("&")

          const tasks = await apiCall(path) as any[]
          return JSON.stringify(tasks.map(t => ({
            id: t.id,
            title: t.title,
            status: t.status,
            assignee: t.assignee,
            repo: t.repo,
          })), null, 2)
        }

        case "get_messages": {
          const agentId = context.agent || "unknown"
          const messages = await apiCall(`/messages/${agentId}?limit=20`) as any[]
          return JSON.stringify(messages.map(m => ({
            from: m.from,
            type: m.type,
            content: m.content,
            time: m.created_at,
          })), null, 2)
        }

        case "get_checkpoints": {
          const checkpoints = await apiCall("/checkpoints") as any[]
          return JSON.stringify(checkpoints.map(cp => ({
            id: cp.id,
            type: cp.type,
            description: cp.description,
            status: cp.status,
          })), null, 2)
        }

        default:
          return "Unknown action"
      }
    } catch (e: any) {
      return `Failed to query state: ${e.message}`
    }
  },
})
