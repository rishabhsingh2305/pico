import { requestJSON } from "@/lib/api-client"
import type {
  CreateExpenseInput,
  Expense,
  ExpenseListResponse,
  ExpenseSummary,
  Org,
} from "@/features/expenses/types"

const EXPENSES_TAG = "expenses"
const ORG_LIST_LIMIT = 100
const EXPENSE_LIST_LIMIT = 100

export async function findExpensesOrg(userId: string): Promise<Org | null> {
  const query = new URLSearchParams({
    tags: EXPENSES_TAG,
    limit: String(ORG_LIST_LIMIT),
  })
  const response = await requestJSON<{ items: Org[] }>(`/orgs?${query}`, {
    method: "GET",
  })

  return (
    response.items.find((org) => org.userId === userId) ??
    response.items[0] ??
    null
  )
}

export function createExpensesOrg(): Promise<Org> {
  return requestJSON<Org>("/orgs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: "Expenses", tags: [EXPENSES_TAG] }),
  })
}

export type ListExpensesOptions = {
  from?: string
  to?: string
  tags?: string[]
  limit?: number
  projectIds?: string[]
}

export type SummarizeExpensesOptions = {
  from?: string
  to?: string
  tags?: string[]
  projectIds?: string[]
}

function expenseQuery(
  orgId: string,
  projectId: string | undefined,
  options: SummarizeExpensesOptions
): URLSearchParams {
  const query = new URLSearchParams({ orgId })
  if (options.projectIds && options.projectIds.length > 0) {
    query.set("projectIds", options.projectIds.join(","))
  } else if (projectId) {
    query.set("projectId", projectId)
  }
  if (options.from) {
    query.set("from", options.from)
  }
  if (options.to) {
    query.set("to", options.to)
  }
  if (options.tags && options.tags.length > 0) {
    query.set("tags", options.tags.join(","))
  }
  return query
}

export function listExpenses(
  orgId: string,
  projectId: string,
  options: ListExpensesOptions = {}
): Promise<ExpenseListResponse> {
  const query = expenseQuery(orgId, projectId, options)
  query.set("limit", String(options.limit ?? EXPENSE_LIST_LIMIT))
  return requestJSON<ExpenseListResponse>(`/expenses?${query}`, {
    method: "GET",
  })
}

export function summarizeExpenses(
  orgId: string,
  projectId?: string,
  options: SummarizeExpensesOptions = {}
): Promise<ExpenseSummary> {
  const query = expenseQuery(orgId, projectId, options)
  return requestJSON<ExpenseSummary>(`/expenses/summary?${query}`, {
    method: "GET",
  })
}

export function createExpense(
  orgId: string,
  input: CreateExpenseInput
): Promise<Expense> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<Expense>(`/expenses?${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}
