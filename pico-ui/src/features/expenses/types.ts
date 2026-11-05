export type ExpenseType = "debit" | "credit"

export type Expense = {
  id: string
  orgId: string
  userId: string
  projectId: string
  type: ExpenseType
  amount: number
  currency: string
  description: string
  tags: string[]
  spentAt: string
  createdAt: string
  updatedAt: string
}

export type CreateExpenseInput = {
  projectId: string
  type: ExpenseType
  amount: number
  description: string
  tags: string[]
  spentAt: string
}

export type ExpensePage = {
  items: Expense[]
  totalCount: number
  totalDebit: number
  totalCredit: number
  netTotal: number
}
