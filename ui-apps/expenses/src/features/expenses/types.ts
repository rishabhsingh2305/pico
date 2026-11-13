export type Org = {
  id: string
  name: string
  userId: string
  tags: string[]
  createdAt: string
  updatedAt: string
}

export type ExpenseType = "debit" | "credit"

export type Expense = {
  id: string
  projectId: string
  type: ExpenseType
  amount: number
  currency: string
  description: string
  tags: string[]
  spentAt: string
}

export type ExpenseListResponse = {
  items: Expense[]
  nextOffset: number | null
}

export type ExpenseSummaryItem = {
  projectId: string
  currency: string
  totalCount: number
  totalDebit: number
  totalCredit: number
  netTotal: number
}

export type ExpenseSummary = {
  items: ExpenseSummaryItem[]
}

export type CreateExpenseInput = {
  projectId: string
  type: ExpenseType
  amount: number
  description: string
  tags: string[]
  spentAt: string
}
