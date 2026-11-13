import type { ExpenseSummaryItem } from "@/features/expenses/types"
import type { Project } from "@/features/projects/types"

export type ProjectTotals = {
  project: Project
  totals: ExpenseSummaryItem
}

export type CurrencySummary = {
  currency: string
  projects: ProjectTotals[]
  totalCount: number
  totalDebit: number
  totalCredit: number
  netTotal: number
}

/**
 * Groups project totals by currency. Amounts from different currencies are
 * never added together because minor units are not comparable across them.
 */
export function summarizeCurrencies(
  entries: ProjectTotals[]
): CurrencySummary[] {
  const summaries = new Map<string, CurrencySummary>()

  for (const entry of entries) {
    const currency = entry.project.currency
    let summary = summaries.get(currency)
    if (!summary) {
      summary = {
        currency,
        projects: [],
        totalCount: 0,
        totalDebit: 0,
        totalCredit: 0,
        netTotal: 0,
      }
      summaries.set(currency, summary)
    }

    summary.projects.push(entry)
    summary.totalCount += entry.totals.totalCount
    summary.totalDebit += entry.totals.totalDebit
    summary.totalCredit += entry.totals.totalCredit
  }

  return [...summaries.values()].map((summary) => ({
    ...summary,
    netTotal: summary.totalCredit - summary.totalDebit,
    projects: [...summary.projects].sort(
      (left, right) =>
        right.totals.netTotal - left.totals.netTotal ||
        left.project.name.localeCompare(right.project.name)
    ),
  }))
}

export function maxFlowValue(projects: ProjectTotals[]): number {
  return projects.reduce(
    (max, { totals }) => Math.max(max, totals.totalCredit, totals.totalDebit),
    0
  )
}

export function flowWidth(value: number, max: number): number {
  if (value <= 0 || max <= 0) {
    return 0
  }
  return Math.max(2, Math.min(100, (value / max) * 100))
}
