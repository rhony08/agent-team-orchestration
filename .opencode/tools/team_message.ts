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
  description: "Send a message to another agent in the team",
  args: {
    to: tool.schema.string().describe("Recipient agent ID or 'all' for broadcast"),
    type: tool.schema.enum(["status_update", "question", "dependency_alert", "task_assignment", "blocker"]).describe("Message type"),
    content: tool.schema.string().describe("Message content"),
  },
  async execute(args, context) {
    try {
      const message = await apiCall("/messages", "POST", {
        from: context.agent || "unknown",
        to: args.to,
        type: args.type,
        content: args.content,
      })
      return `Message sent to ${args.to} (ID: ${message.id})`
    } catch (e: any) {
      return `Failed to send message: ${e.message}`
    }
  },
})
