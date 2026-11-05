import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api-client"
import { signUpErrorMessage } from "@/features/auth/error-message"

describe("signUpErrorMessage", () => {
  it("shows server validation messages", () => {
    expect(signUpErrorMessage(new ApiError(400, "email is required"))).toBe(
      "email is required"
    )
  })

  it("maps duplicate emails", () => {
    expect(signUpErrorMessage(new ApiError(409, "email already exists"))).toBe(
      "An account with this email already exists."
    )
  })

  it("maps email delivery failures", () => {
    expect(signUpErrorMessage(new ApiError(502, "delivery failed"))).toBe(
      "We couldn't send the verification email. Please try again."
    )
  })

  it("hides unexpected API errors", () => {
    expect(signUpErrorMessage(new ApiError(500, "database down"))).toBe(
      "Something went wrong. Please try again."
    )
  })

  it("maps network failures", () => {
    expect(signUpErrorMessage(new TypeError("Failed to fetch"))).toBe(
      "Could not reach the server. Check your connection and try again."
    )
  })

  it("falls back for unknown errors", () => {
    expect(signUpErrorMessage("boom")).toBe(
      "Something went wrong. Please try again."
    )
  })
})
