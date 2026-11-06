import { describe, expect, it } from "vitest"

import { isSessionInvalidError, sessionFromUser } from "@/features/auth/session"
import { ApiError } from "@/lib/api-client"

describe("session", () => {
  it("maps a user response to a session", () => {
    expect(
      sessionFromUser({
        id: "user-1",
        name: "Ada Lovelace",
        email: "ada@example.com",
        emailVerified: true,
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
      })
    ).toEqual({
      userId: "user-1",
      name: "Ada Lovelace",
      email: "ada@example.com",
    })
  })

  it("identifies expired sessions", () => {
    expect(
      isSessionInvalidError(new ApiError(401, "authentication required"))
    ).toBe(true)
    expect(isSessionInvalidError(new ApiError(404, "user not found"))).toBe(
      true
    )
    expect(isSessionInvalidError(new ApiError(404, "org not found"))).toBe(
      false
    )
    expect(isSessionInvalidError(new ApiError(400, "bad request"))).toBe(false)
  })
})
