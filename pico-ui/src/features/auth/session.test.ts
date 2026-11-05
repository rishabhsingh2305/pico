import { beforeEach, describe, expect, it } from "vitest"

import {
  clearSession,
  getSession,
  isUserNotFoundError,
  storeSession,
} from "@/features/auth/session"
import { ApiError } from "@/lib/api-client"

beforeEach(() => {
  localStorage.clear()
})

describe("temporary session", () => {
  it("stores and reads a session", () => {
    storeSession({ userId: "user-1", email: "ada@example.com" })

    expect(getSession()).toEqual({
      userId: "user-1",
      email: "ada@example.com",
    })
  })

  it("clears a session", () => {
    storeSession({ userId: "user-1", email: "ada@example.com" })
    clearSession()

    expect(getSession()).toBeNull()
  })

  it("ignores malformed stored data", () => {
    localStorage.setItem("pico.session", "not-json")

    expect(getSession()).toBeNull()
  })

  it("ignores sessions without a user id", () => {
    localStorage.setItem(
      "pico.session",
      JSON.stringify({ userId: "", email: "ada@example.com" })
    )

    expect(getSession()).toBeNull()
  })

  it("identifies a user-not-found error", () => {
    expect(isUserNotFoundError(new ApiError(404, "user not found"))).toBe(true)
    expect(isUserNotFoundError(new ApiError(404, "org not found"))).toBe(false)
    expect(isUserNotFoundError(new ApiError(400, "bad request"))).toBe(false)
  })
})
