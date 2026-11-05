import { describe, expect, it } from "vitest"

import {
  validateProjectName,
  validateProjectTags,
} from "@/features/projects/validation"

describe("validateProjectName", () => {
  it("accepts a valid name", () => {
    expect(validateProjectName("General")).toBeUndefined()
  })

  it("requires a name", () => {
    expect(validateProjectName("  ")).toBe("Enter a project name.")
  })

  it("limits the name to 120 bytes", () => {
    expect(validateProjectName("a".repeat(121))).toBe(
      "Name must be at most 120 characters."
    )
  })
})

describe("validateProjectTags", () => {
  it("accepts valid tags", () => {
    expect(validateProjectTags(["travel", "food"])).toBeUndefined()
  })

  it("rejects more than 20 tags", () => {
    expect(
      validateProjectTags(
        Array.from({ length: 21 }, (_, index) => `tag-${index}`)
      )
    ).toBe("Use at most 20 tags.")
  })

  it("rejects oversized tags", () => {
    expect(validateProjectTags(["a".repeat(33)])).toBe(
      "Each tag must be at most 32 characters."
    )
  })
})
