import { ApiError } from "@/lib/api-client"
import type { User } from "@/features/auth/types"

export type Session = {
  userId: string
  name?: string
  email: string
}

export function sessionFromUser(user: User): Session {
  return {
    userId: user.id,
    name: user.name,
    email: user.email,
  }
}

export function isSessionInvalidError(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (error.status === 401 ||
      (error.status === 404 && error.message === "user not found"))
  )
}
