import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api-client"
import { orgErrorMessage } from "@/features/orgs/error-message"

describe("orgErrorMessage", () => {
  it("shows server validation messages", () => {
    expect(orgErrorMessage(new ApiError(400, "name is required"))).toBe(
      "name is required"
    )
  })

  it("maps forbidden errors", () => {
    expect(orgErrorMessage(new ApiError(403, "forbidden"))).toBe(
      "You do not have permission to do that."
    )
  })

  it("maps stale sessions", () => {
    expect(orgErrorMessage(new ApiError(404, "user not found"))).toBe(
      "Your session is no longer valid. Sign in again."
    )
  })

  it("maps network errors", () => {
    expect(orgErrorMessage(new TypeError("Failed to fetch"))).toBe(
      "Could not reach the server. Check your connection and try again."
    )
  })

  it("falls back for unexpected errors", () => {
    expect(orgErrorMessage(new ApiError(500, "database down"))).toBe(
      "Something went wrong. Please try again."
    )
  })
})
