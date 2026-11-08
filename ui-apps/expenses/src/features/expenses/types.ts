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
  totalCount: number
  totalDebit: number
  totalCredit: number
  netTotal: number
}

export type CreateExpenseInput = {
  projectId: string
  type: ExpenseType
  amount: number
  description: string
  tags: string[]
  spentAt: string
}
