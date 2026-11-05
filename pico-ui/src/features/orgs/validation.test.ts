import { describe, expect, it } from "vitest"

import { validateOrgName, validateOrgTags } from "@/features/orgs/validation"

describe("validateOrgName", () => {
  it("accepts a valid name", () => {
    expect(validateOrgName("Acme Inc.")).toBeUndefined()
  })

  it("requires a name", () => {
    expect(validateOrgName("   ")).toBe("Enter an organization name.")
  })

  it("limits the name to 120 bytes", () => {
    expect(validateOrgName("a".repeat(121))).toBe(
      "Name must be at most 120 characters."
    )
  })
})

describe("validateOrgTags", () => {
  it("accepts valid tags", () => {
    expect(validateOrgTags(["team", "finance"])).toBeUndefined()
  })

  it("rejects more than 20 tags", () => {
    expect(
      validateOrgTags(Array.from({ length: 21 }, (_, index) => `tag-${index}`))
    ).toBe("Use at most 20 tags.")
  })

  it("rejects oversized tags", () => {
    expect(validateOrgTags(["a".repeat(33)])).toBe(
      "Each tag must be at most 32 characters."
    )
  })
})
