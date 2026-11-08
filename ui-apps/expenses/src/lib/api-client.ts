import { getAuthToken } from "@/lib/auth-token"
import { API_BASE_URL } from "@/lib/env"

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
  }
}

export async function requestJSON<T>(
  path: string,
  init: RequestInit = {}
): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set("Accept", "application/json")

  const token = getAuthToken()
  if (token) {
    headers.set("Authorization", `Bearer ${token}`)
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "omit",
    ...init,
    headers,
  })
  const body = parseJSON(await response.text())

  if (!response.ok) {
    throw new ApiError(response.status, errorMessage(body))
  }

  return body as T
}

function parseJSON(raw: string): unknown {
  if (!raw) {
    return undefined
  }

  try {
    return JSON.parse(raw)
  } catch {
    return undefined
  }
}

function errorMessage(body: unknown): string {
  if (
    isRecord(body) &&
    typeof body.error === "string" &&
    body.error.trim() !== ""
  ) {
    return body.error
  }
  return "Request failed"
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}
