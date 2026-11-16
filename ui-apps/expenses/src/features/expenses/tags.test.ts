import { describe, expect, it } from "vitest"

import { unusedTags } from "@/features/expenses/tags"

describe("unusedTags", () => {
  it("removes tags that are already selected", () => {
    expect(unusedTags(["food", "travel", "rent"], ["travel"])).toEqual([
      "food",
      "rent",
    ])
  })

  it("compares tags case-insensitively and ignores empty values", () => {
    expect(unusedTags(["Food", "", "Rent"], ["food", " "])).toEqual(["Rent"])
  })
})
