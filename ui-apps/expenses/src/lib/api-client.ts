import {
  clearTokens,
  getAuthToken,
  getRefreshToken,
  setAuthToken,
  setRefreshToken,
} from "@/lib/auth-token"
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
  let response = await send(path, init)

  if (response.status === 401 && canRefresh(path)) {
    if (await refreshTokens()) {
      response = await send(path, init)
    }
  }

  return parseResponse<T>(response)
}

function send(path: string, init: RequestInit): Promise<Response> {
  const headers = new Headers(init.headers)
  headers.set("Accept", "application/json")

  const token = getAuthToken()
  if (token) {
    headers.set("Authorization", `Bearer ${token}`)
  }

  return fetch(`${API_BASE_URL}${path}`, {
    credentials: "omit",
    ...init,
    headers,
  })
}

function canRefresh(path: string): boolean {
  return (
    !path.startsWith("/auth/login") &&
    !path.startsWith("/auth/refresh") &&
    !path.startsWith("/auth/logout")
  )
}

let refreshInFlight: Promise<boolean> | null = null

export function refreshTokens(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight
  }

  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    return Promise.resolve(false)
  }

  refreshInFlight = fetch(`${API_BASE_URL}/auth/refresh`, {
    method: "POST",
    credentials: "omit",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
  })
    .then(async (response) => {
      if (!response.ok) {
        clearTokens()
        return false
      }
      const body = parseJSON(await response.text())
      if (!isRecord(body) || typeof body.accessToken !== "string") {
        clearTokens()
        return false
      }
      setAuthToken(body.accessToken)
      if (typeof body.refreshToken === "string" && body.refreshToken !== "") {
        setRefreshToken(body.refreshToken)
      }
      return true
    })
    .catch(() => false)
    .finally(() => {
      refreshInFlight = null
    })

  return refreshInFlight
}

async function parseResponse<T>(response: Response): Promise<T> {
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
