import { describe, expect, it } from "vitest"

import {
  dateRangeToAPI,
  defaultAnalyticsFilters,
  defaultAnalyticsRange,
} from "@/features/analytics/filters"

describe("analytics filters", () => {
  it("defaults to the last 30 days grouped by day", () => {
    const filters = defaultAnalyticsFilters(new Date(2026, 10, 22))
    expect(filters).toEqual({
      from: "2026-10-24",
      to: "2026-11-22",
      interval: "day",
      event: "",
      country: "",
      device: "",
      bot: "",
      referrer: "",
    })
  })

  it("builds an inclusive local date range in UTC", () => {
    const range = defaultAnalyticsRange(new Date(2026, 10, 22))
    expect(range).toEqual({ from: "2026-10-24", to: "2026-11-22" })
  })

  it("converts date inputs to ISO day bounds", () => {
    const range = dateRangeToAPI("2026-11-01", "2026-11-02")
    expect(range.from).toBe(new Date(2026, 10, 1, 0, 0, 0).toISOString())
    expect(range.to).toBe(new Date(2026, 10, 2, 23, 59, 59, 999).toISOString())
  })
})
