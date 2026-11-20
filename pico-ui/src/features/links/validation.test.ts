import { describe, expect, it } from "vitest"

import {
  validateLinkForm,
  validateLinkSlug,
  validateLinkTitle,
  validateLinkURL,
} from "@/features/links/validation"

describe("link validation", () => {
  it("requires an http or https destination", () => {
    expect(validateLinkURL("")).toBe("Enter a destination URL.")
    expect(validateLinkURL("ftp://example.com")).toBe(
      "Only http and https URLs are supported."
    )
    expect(validateLinkURL("https://user:pass@example.com")).toBe(
      "URLs with credentials are not supported."
    )
    expect(validateLinkURL("https://example.com/page")).toBeUndefined()
  })

  it("accepts an empty alias and validates custom aliases", () => {
    expect(validateLinkSlug("")).toBeUndefined()
    expect(validateLinkSlug("ab")).toBe("Alias must be 3 to 32 characters.")
    expect(validateLinkSlug("bad slug")).toBe(
      "Use lowercase letters, numbers, and hyphens; start and end with a letter or number."
    )
    expect(validateLinkSlug("Launch-2026")).toBeUndefined()
  })

  it("caps the title at 120 bytes", () => {
    expect(validateLinkTitle("Launch")).toBeUndefined()
    expect(validateLinkTitle("a".repeat(121))).toBe(
      "Title must be at most 120 characters."
    )
  })

  it("collects form errors", () => {
    expect(
      validateLinkForm({
        url: "",
        slug: "x",
        title: "a".repeat(121),
        tags: Array.from({ length: 21 }, (_, index) => `tag-${index}`),
      })
    ).toEqual({
      url: "Enter a destination URL.",
      slug: "Alias must be 3 to 32 characters.",
      title: "Title must be at most 120 characters.",
      tags: "Use at most 20 tags.",
    })
  })
})
