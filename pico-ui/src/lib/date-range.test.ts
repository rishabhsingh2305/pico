import { describe, expect, it } from "vitest"

import {
  DATE_RANGE_PRESETS,
  dateRangeFromValues,
  findDateRangePreset,
  formatDateRangeLabel,
  toDateRangeValue,
} from "@/lib/date-range"

const today = new Date(2026, 10, 5, 15, 30)

function presetRange(id: string) {
  const preset = DATE_RANGE_PRESETS.find((entry) => entry.id === id)
  if (!preset) {
    throw new Error(`missing preset ${id}`)
  }
  return preset.resolve(today)
}

describe("date range presets", () => {
  it("resolves today to a single day", () => {
    expect(presetRange("today")).toEqual({
      from: "2026-11-05",
      to: "2026-11-05",
    })
  })

  it("includes today in the last 7 and last 30 days", () => {
    expect(presetRange("last-7-days")).toEqual({
      from: "2026-10-30",
      to: "2026-11-05",
    })
    expect(presetRange("last-30-days")).toEqual({
      from: "2026-10-07",
      to: "2026-11-05",
    })
  })

  it("starts this month at the first day and ends today", () => {
    expect(presetRange("this-month")).toEqual({
      from: "2026-11-01",
      to: "2026-11-05",
    })
  })

  it("covers the whole previous month", () => {
    expect(presetRange("last-month")).toEqual({
      from: "2026-10-01",
      to: "2026-10-31",
    })
  })

  it("matches a preset by its exact range", () => {
    expect(findDateRangePreset("2026-10-30", "2026-11-05", today)?.id).toBe(
      "last-7-days"
    )
    expect(findDateRangePreset("2026-10-30", "2026-11-04", today)).toBe(
      undefined
    )
    expect(findDateRangePreset("", "", today)).toBe(undefined)
  })
})

describe("date range values", () => {
  it("parses and formats day strings without timezone drift", () => {
    const dates = dateRangeFromValues("2026-11-01", "2026-11-05")

    expect(dates.from?.getFullYear()).toBe(2026)
    expect(dates.from?.getMonth()).toBe(10)
    expect(dates.from?.getDate()).toBe(1)
    expect(toDateRangeValue(dates)).toEqual({
      from: "2026-11-01",
      to: "2026-11-05",
    })
  })

  it("round-trips partial values", () => {
    expect(toDateRangeValue(dateRangeFromValues("2026-11-01", ""))).toEqual({
      from: "2026-11-01",
      to: "",
    })
    expect(toDateRangeValue({})).toEqual({ from: "", to: "" })
  })
})

describe("formatDateRangeLabel", () => {
  it("describes an empty range", () => {
    expect(formatDateRangeLabel("", "")).toBe("Any date")
  })

  it("describes partial ranges", () => {
    expect(formatDateRangeLabel("2026-11-01", "")).toBe("From Nov 1, 2026")
    expect(formatDateRangeLabel("", "2026-11-01")).toBe("Until Nov 1, 2026")
  })

  it("omits the year within one year", () => {
    expect(formatDateRangeLabel("2026-11-01", "2026-11-05")).toBe(
      "Nov 1 – Nov 5"
    )
  })

  it("shows years across a year boundary", () => {
    expect(formatDateRangeLabel("2025-12-30", "2026-01-02")).toBe(
      "Dec 30, 2025 – Jan 2, 2026"
    )
  })

  it("shows one date for a single day", () => {
    expect(formatDateRangeLabel("2026-11-05", "2026-11-05")).toBe("Nov 5, 2026")
  })
})
