import type { ExpenseSummaryItem } from "@/features/expenses/types"

export type CurrencyTotals = {
  currency: string
  projectCount: number
  totalCount: number
  totalDebit: number
  totalCredit: number
  netTotal: number
}

/**
 * Groups expense summary rows by currency. Amounts from different currencies
 * are never added together because minor units are not comparable across them.
 */
export function summarizeByCurrency(
  items: ExpenseSummaryItem[]
): CurrencyTotals[] {
  const summaries = new Map<
    string,
    CurrencyTotals & { projectIDs: Set<string> }
  >()

  for (const item of items) {
    let summary = summaries.get(item.currency)
    if (!summary) {
      summary = {
        currency: item.currency,
        projectCount: 0,
        totalCount: 0,
        totalDebit: 0,
        totalCredit: 0,
        netTotal: 0,
        projectIDs: new Set(),
      }
      summaries.set(item.currency, summary)
    }

    summary.projectIDs.add(item.projectId)
    summary.totalCount += item.totalCount
    summary.totalDebit += item.totalDebit
    summary.totalCredit += item.totalCredit
  }

  return [...summaries.values()]
    .map(({ projectIDs, ...summary }) => ({
      ...summary,
      projectCount: projectIDs.size,
      netTotal: summary.totalCredit - summary.totalDebit,
    }))
    .sort((left, right) => left.currency.localeCompare(right.currency))
}
