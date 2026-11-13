export type MonthRange = {
  from: string
  to: string
  label: string
}

export function monthRange(date = new Date()): MonthRange {
  const start = new Date(date.getFullYear(), date.getMonth(), 1, 0, 0, 0, 0)
  const end = new Date(
    date.getFullYear(),
    date.getMonth() + 1,
    0,
    23,
    59,
    59,
    999
  )

  return {
    from: start.toISOString(),
    to: end.toISOString(),
    label: new Intl.DateTimeFormat(undefined, {
      month: "long",
      year: "numeric",
    }).format(start),
  }
}

export function previousMonthRange(date = new Date()): MonthRange {
  return monthRange(new Date(date.getFullYear(), date.getMonth() - 1, 1))
}

export function lastThreeMonthsRange(date = new Date()): MonthRange {
  const start = new Date(date.getFullYear(), date.getMonth() - 2, 1, 0, 0, 0, 0)
  const end = new Date(
    date.getFullYear(),
    date.getMonth() + 1,
    0,
    23,
    59,
    59,
    999
  )
  return {
    from: start.toISOString(),
    to: end.toISOString(),
    label: "Last 3 months",
  }
}

export function yearRange(date = new Date()): MonthRange {
  const start = new Date(date.getFullYear(), 0, 1, 0, 0, 0, 0)
  const end = new Date(date.getFullYear(), 11, 31, 23, 59, 59, 999)
  return {
    from: start.toISOString(),
    to: end.toISOString(),
    label: "This year",
  }
}
