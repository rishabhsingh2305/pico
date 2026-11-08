import type { Expense } from "@/features/expenses/types"

export type MonthRange = {
  from: string
  to: string
  label: string
  start: Date
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
    start,
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
    start,
  }
}

export function yearRange(date = new Date()): MonthRange {
  const start = new Date(date.getFullYear(), 0, 1, 0, 0, 0, 0)
  const end = new Date(date.getFullYear(), 11, 31, 23, 59, 59, 999)
  return {
    from: start.toISOString(),
    to: end.toISOString(),
    label: "This year",
    start,
  }
}

export function daysInMonth(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate()
}

export function dailyDebitTotals(items: Expense[], monthStart: Date): number[] {
  const totals = Array.from({ length: daysInMonth(monthStart) }, () => 0)

  for (const item of items) {
    if (item.type !== "debit") {
      continue
    }

    const spentAt = new Date(item.spentAt)
    if (
      spentAt.getFullYear() !== monthStart.getFullYear() ||
      spentAt.getMonth() !== monthStart.getMonth()
    ) {
      continue
    }

    totals[spentAt.getDate() - 1] += item.amount
  }

  return totals
}

export function percentChange(
  current: number,
  previous: number
): number | null {
  if (previous === 0) {
    return current === 0 ? 0 : null
  }
  return ((current - previous) / previous) * 100
}
