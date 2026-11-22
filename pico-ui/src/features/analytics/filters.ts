import { subDays } from "date-fns"

import { DATE_RANGE_DATE_FORMAT } from "@/lib/date-range"
import type { AnalyticsStatsFilters } from "@/features/analytics/types"

export function defaultAnalyticsRange(reference = new Date()) {
  const end = reference
  const start = subDays(end, 29)
  return {
    from: formatDate(start),
    to: formatDate(end),
  }
}

export function defaultAnalyticsFilters(
  reference = new Date()
): AnalyticsStatsFilters {
  const range = defaultAnalyticsRange(reference)
  return {
    from: range.from,
    to: range.to,
    interval: "day",
    event: "",
    country: "",
    device: "",
    bot: "",
    referrer: "",
  }
}

export function dateRangeToAPI(from: string, to: string) {
  return {
    from: new Date(`${from}T00:00:00`).toISOString(),
    to: new Date(`${to}T23:59:59.999`).toISOString(),
  }
}

function formatDate(value: Date): string {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, "0")
  const day = String(value.getDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

export { DATE_RANGE_DATE_FORMAT }
