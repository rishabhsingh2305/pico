import { describe, expect, it } from "vitest"

import { parseExpenseFilters } from "@/features/expenses/filters"

describe("parseExpenseFilters", () => {
  it("normalizes and sorts tags", () => {
    const result = parseExpenseFilters({
      tags: [" travel", "food", "food", ""],
      from: "",
      to: "",
    })

    expect(result.error).toBeUndefined()
    expect(result.filters).toEqual({ tags: ["food", "travel"] })
  })

  it("converts date inputs to ISO day bounds", () => {
    const result = parseExpenseFilters({
      tags: [],
      from: "2026-01-02",
      to: "2026-01-03",
    })

    expect(result.error).toBeUndefined()
    expect(result.filters?.from).toMatch(/^2026-01-0[12]T/)
    expect(result.filters?.to).toMatch(/^2026-01-0[34]T/)
    expect(result.filters?.from).toMatch(/Z$/)
    expect(result.filters?.to).toMatch(/Z$/)
  })

  it("rejects an inverted range", () => {
    const result = parseExpenseFilters({
      tags: [],
      from: "2026-01-05",
      to: "2026-01-02",
    })

    expect(result.error).toBe("From date must be before the to date.")
  })

  it("rejects invalid dates", () => {
    expect(
      parseExpenseFilters({ tags: [], from: "not-a-date", to: "" }).error
    ).toBe("Choose a valid from date.")
    expect(
      parseExpenseFilters({ tags: [], from: "", to: "not-a-date" }).error
    ).toBe("Choose a valid to date.")
  })
})
