import { ApiError } from "@/lib/api-client"

export type TemporarySession = {
  userId: string
  email: string
}

const STORAGE_KEY = "pico.session"

export function getSession(): TemporarySession | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return null
    }

    const parsed: unknown = JSON.parse(raw)
    if (!isTemporarySession(parsed)) {
      localStorage.removeItem(STORAGE_KEY)
      return null
    }

    return {
      userId: parsed.userId,
      email: parsed.email,
    }
  } catch {
    return null
  }
}

function isTemporarySession(value: unknown): value is TemporarySession {
  if (typeof value !== "object" || value === null) {
    return false
  }

  const record = value as Record<string, unknown>
  return (
    typeof record.userId === "string" &&
    record.userId.trim() !== "" &&
    typeof record.email === "string"
  )
}

export function storeSession(session: TemporarySession): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
  } catch {
    // Storage can be unavailable in private or restricted contexts.
  }
}

export function clearSession(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // Nothing to clear when storage is unavailable.
  }
}

export function isUserNotFoundError(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    error.status === 404 &&
    error.message === "user not found"
  )
}
