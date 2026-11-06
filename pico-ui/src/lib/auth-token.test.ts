import { describe, expect, it } from "vitest"

import { clearAuthToken, getAuthToken, setAuthToken } from "@/lib/auth-token"

describe("auth token storage", () => {
  it("stores, reads, and clears the token", () => {
    setAuthToken("token-1")
    expect(getAuthToken()).toBe("token-1")

    clearAuthToken()
    expect(getAuthToken()).toBeNull()
  })
})
