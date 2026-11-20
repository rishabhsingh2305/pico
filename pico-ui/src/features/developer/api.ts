import { ApiError, requestJSON } from "@/lib/api-client"

export type PingResult = {
  message: string
}

export type ReadyResult = {
  status: "ok" | "unavailable"
}

export async function checkPing(): Promise<PingResult> {
  return requestJSON<PingResult>("/ping", { method: "GET" })
}

export async function checkReady(): Promise<ReadyResult> {
  try {
    const response = await requestJSON<{ status: string }>("/ready", {
      method: "GET",
    })
    return {
      status: response.status === "ok" ? "ok" : "unavailable",
    }
  } catch (error) {
    if (error instanceof ApiError && error.status === 503) {
      return { status: "unavailable" }
    }
    throw error
  }
}
