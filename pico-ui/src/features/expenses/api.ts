import { requestJSON } from "@/lib/api-client"
import type {
  CreateExpenseInput,
  Expense,
  ExpensePage,
} from "@/features/expenses/types"

export type ListExpensesParams = {
  userId: string
  orgId: string
  projectId?: string
  tags: string[]
  from?: string
  to?: string
  limit: number
  offset: number
}

export async function listExpenses(
  params: ListExpensesParams
): Promise<ExpensePage> {
  const query = new URLSearchParams({
    orgId: params.orgId,
    userId: params.userId,
    limit: String(params.limit),
    offset: String(params.offset),
  })
  if (params.projectId) {
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

  return requestJSON<ExpensePage>(`/expenses?${query}`, { method: "GET" })
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
