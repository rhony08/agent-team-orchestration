// Orchestration Plugin for OpenCode
// Enables multi-agent coordination across repositories
import type { Plugin } from "@opencode-ai/plugin"

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

export const OrchestrationPlugin: Plugin = async ({ project, client, $, directory }) => {
  let connected = false

  return {
    // Initialize on session start
    "session.created": async () => {
      try {
        await apiCall("/health")
        connected = true
        await client.app.log({
          body: {
            service: "orchestration",
            level: "info",
            message: "Connected to orchestration server",
          },
        })
      } catch (e) {
        connected = false
        console.warn("Orchestrator not available, running in standalone mode")
      }
    },

    // Check for messages on idle
    "session.idle": async () => {
      if (!connected) return
      // Messages are fetched via sync_workspace tool
    },

    // Log tool usage for audit
    "tool.execute.after": async (input, output) => {
      if (!connected) return
      // Could log to audit trail here
    },
  }
}
