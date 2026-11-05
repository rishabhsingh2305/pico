import { describe, expect, it } from "vitest"

import { byteLength } from "@/lib/text"
import { validateSignUp } from "@/features/auth/validation"

describe("validateSignUp", () => {
  const validInput = {
    name: "Ada Lovelace",
    email: "ada@example.com",
    password: "correct horse",
  }

  it("accepts valid input", () => {
    expect(validateSignUp(validInput)).toEqual({})
  })

  it("requires a name", () => {
    expect(validateSignUp({ ...validInput, name: "   " })).toMatchObject({
      name: "Enter your name.",
    })
  })

  it("limits the name to 120 bytes", () => {
    expect(
      validateSignUp({ ...validInput, name: "a".repeat(121) })
    ).toMatchObject({
      name: "Name must be at most 120 characters.",
    })
  })

  it("requires an email", () => {
    expect(validateSignUp({ ...validInput, email: "" })).toMatchObject({
      email: "Enter your email address.",
    })
  })

  it("rejects malformed email addresses", () => {
    expect(
      validateSignUp({ ...validInput, email: "not-an-email" })
    ).toMatchObject({
      email: "Enter a valid email address.",
    })
  })

  it("limits the email to 254 bytes", () => {
    expect(
      validateSignUp({ ...validInput, email: `${"a".repeat(250)}@x.co` })
    ).toMatchObject({
      email: "Email must be at most 254 characters.",
    })
  })

  it("requires at least 8 password bytes", () => {
    expect(validateSignUp({ ...validInput, password: "short" })).toMatchObject({
      password: "Password must be at least 8 characters.",
    })
  })

  it("limits the password to 72 bytes", () => {
    expect(
      validateSignUp({ ...validInput, password: "a".repeat(73) })
    ).toMatchObject({
      password: "Password must be at most 72 bytes.",
    })
  })

  it("counts password bytes rather than characters", () => {
    expect(byteLength("é")).toBe(2)
    expect(
      validateSignUp({ ...validInput, password: "é".repeat(37) })
    ).toMatchObject({
      password: "Password must be at most 72 bytes.",
    })
  })
})
