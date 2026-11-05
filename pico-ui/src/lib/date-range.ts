import {
  endOfMonth,
  format,
  parse,
  startOfMonth,
  subDays,
  subMonths,
} from "date-fns"

export const DATE_RANGE_DATE_FORMAT = "yyyy-MM-dd"

export type DateRangeValue = {
  from: string
  to: string
}

export type DateRangeDates = {
  from: Date | undefined
  to?: Date | undefined
}

export type DateRangePresetID =
  "today" | "last-7-days" | "last-30-days" | "this-month" | "last-month"

export type DateRangePreset = {
  id: DateRangePresetID
  label: string
  resolve: (today: Date) => DateRangeValue
}

function toValue(from: Date, to: Date): DateRangeValue {
  return {
    from: format(from, DATE_RANGE_DATE_FORMAT),
    to: format(to, DATE_RANGE_DATE_FORMAT),
  }
}

export const DATE_RANGE_PRESETS: readonly DateRangePreset[] = [
  {
    id: "today",
    label: "Today",
    resolve: (today) => toValue(today, today),
  },
  {
    id: "last-7-days",
    label: "Last 7 days",
    resolve: (today) => toValue(subDays(today, 6), today),
  },
  {
    id: "last-30-days",
    label: "Last 30 days",
    resolve: (today) => toValue(subDays(today, 29), today),
  },
  {
    id: "this-month",
    label: "This month",
    resolve: (today) => toValue(startOfMonth(today), today),
  },
  {
    id: "last-month",
    label: "Last month",
    resolve: (today) => {
      const lastMonth = subMonths(today, 1)
      return toValue(startOfMonth(lastMonth), endOfMonth(lastMonth))
    },
  },
]

function parseDate(value: string): Date | undefined {
  if (value.trim() === "") {
    return undefined
  }
  const parsed = parse(value, DATE_RANGE_DATE_FORMAT, new Date())
  return Number.isNaN(parsed.getTime()) ? undefined : parsed
}

export function dateRangeFromValues(from: string, to: string): DateRangeDates {
  return { from: parseDate(from), to: parseDate(to) }
}

export function toDateRangeValue(range: {
  from?: Date
  to?: Date
}): DateRangeValue {
  return {
    from: range.from ? format(range.from, DATE_RANGE_DATE_FORMAT) : "",
    to: range.to ? format(range.to, DATE_RANGE_DATE_FORMAT) : "",
  }
}

export function findDateRangePreset(
  from: string,
  to: string,
  today: Date = new Date()
): DateRangePreset | undefined {
  return DATE_RANGE_PRESETS.find((preset) => {
    const value = preset.resolve(today)
    return value.from === from && value.to === to
  })
}

function shortDate(value: string): string {
  const parsed = parseDate(value)
  return parsed ? format(parsed, "MMM d, yyyy") : value
}

export function formatDateRangeLabel(from: string, to: string): string {
  if (from === "" && to === "") {
    return "Any date"
  }
  if (from !== "" && to === "") {
    return `From ${shortDate(from)}`
  }
  if (from === "" && to !== "") {
    return `Until ${shortDate(to)}`
  }

  const fromDate = parseDate(from)
  const toDate = parseDate(to)
  if (!fromDate || !toDate) {
    return `${from} – ${to}`
  }
  if (fromDate.getFullYear() !== toDate.getFullYear()) {
    return `${format(fromDate, "MMM d, yyyy")} – ${format(toDate, "MMM d, yyyy")}`
  }
  if (from === to) {
    return format(fromDate, "MMM d, yyyy")
  }
  return `${format(fromDate, "MMM d")} – ${format(toDate, "MMM d")}`
}
