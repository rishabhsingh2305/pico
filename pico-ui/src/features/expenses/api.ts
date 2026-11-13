import { requestJSON } from "@/lib/api-client"
import type {
  CreateExpenseInput,
  Expense,
  ExpensePage,
  ExpenseSummary,
} from "@/features/expenses/types"

export type ListExpensesParams = {
  userId: string
  orgId: string
  projectId?: string
  projectIds?: string[]
  tags: string[]
  from?: string
  to?: string
  limit: number
  offset: number
}

export type SummarizeExpensesParams = {
  userId: string
  orgId: string
  projectId?: string
  projectIds?: string[]
  tags: string[]
  from?: string
  to?: string
}

function expenseQuery(params: {
  userId: string
  orgId: string
  projectId?: string
  projectIds?: string[]
  tags: string[]
  from?: string
  to?: string
}): URLSearchParams {
  const query = new URLSearchParams({
    orgId: params.orgId,
    userId: params.userId,
  })
  if (params.projectIds && params.projectIds.length > 0) {
    query.set("projectIds", params.projectIds.join(","))
  } else if (params.projectId) {
    query.set("projectId", params.projectId)
  }
  if (params.tags.length > 0) {
    query.set("tags", params.tags.join(","))
  }
  if (params.from) {
    query.set("from", params.from)
  }
  if (params.to) {
    query.set("to", params.to)
  }
  return query
}

export async function listExpenses(
  params: ListExpensesParams
): Promise<ExpensePage> {
  const query = expenseQuery(params)
  query.set("limit", String(params.limit))
  query.set("offset", String(params.offset))

  return requestJSON<ExpensePage>(`/expenses?${query}`, { method: "GET" })
}

export function summarizeExpenses(
  params: SummarizeExpensesParams
): Promise<ExpenseSummary> {
  const query = expenseQuery(params)
  return requestJSON<ExpenseSummary>(`/expenses/summary?${query}`, {
    method: "GET",
  })
}

export function createExpense(
  userId: string,
  orgId: string,
  input: CreateExpenseInput
): Promise<Expense> {
  const query = new URLSearchParams({ orgId, userId })
  return requestJSON<Expense>(`/expenses?${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}

export async function deleteExpense(
  userId: string,
  orgId: string,
  expenseId: string
): Promise<void> {
  const query = new URLSearchParams({ orgId, userId })
  await requestJSON<void>(
    `/expenses/${encodeURIComponent(expenseId)}?${query}`,
    { method: "DELETE" }
  )
}
