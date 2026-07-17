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
  description: "Request human approval before proceeding with a critical operation",
  args: {
    type: tool.schema.enum(["pre_commit", "pre_push", "schema_change", "breaking_change", "destructive"]).describe("Checkpoint type"),
    description: tool.schema.string().describe("What needs approval"),
    affected_repos: tool.schema.array(tool.schema.string()).optional().describe("Affected repositories"),
  },
  async execute(args, context) {
    try {
      const checkpoint = await apiCall("/checkpoints", "POST", {
        type: args.type,
        description: args.description,
        requester: context.agent || "unknown",
        affected_repos: args.affected_repos || [],
      })

      // Poll for resolution
      const cpId = checkpoint.id
      let resolved = false
      let result = ""

      for (let i = 0; i < 60; i++) { // 5 minute timeout
        await new Promise(r => setTimeout(r, 5000))

        const cp = await apiCall(`/checkpoints/${cpId}`) as any
        if (cp.status !== "pending") {
          resolved = true
          if (cp.status === "approved") {
            result = "Checkpoint approved. Proceeding."
          } else {
            result = `Checkpoint denied. Reason: ${cp.reason || "No reason provided"}`
          }
          break
        }
      }

      if (!resolved) {
        result = "Checkpoint timed out after 5 minutes"
      }

      return result
    } catch (e: any) {
      return `Failed to create checkpoint: ${e.message}`
    }
  },
})
