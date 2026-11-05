import { parseAmountToMinorUnits } from "@/features/expenses/money"
import type { ExpenseType } from "@/features/expenses/types"
import { normalizeTags, tagValidationError } from "@/lib/tags"

const DESCRIPTION_MAX_LENGTH = 1000

export type ExpenseFormValues = {
  projectId: string
  type: ExpenseType
  amount: string
  description: string
  tags: string[]
  spentAt: string
}

export type ExpenseField = keyof ExpenseFormValues
export type ExpenseErrors = Partial<Record<ExpenseField, string>>

export type ExpenseValidation = {
  errors: ExpenseErrors
  type?: ExpenseType
  amountMinorUnits?: number
  tags?: string[]
  spentAtIso?: string
}

export function validateExpense(
  values: ExpenseFormValues,
  currency: string
): ExpenseValidation {
  const errors: ExpenseErrors = {}

  if (values.projectId.trim() === "") {
    errors.projectId = "Choose a project."
  }

  let type: ExpenseType | undefined
  if (values.type === "debit" || values.type === "credit") {
    type = values.type
  } else {
    errors.type = "Choose debit or credit."
  }

  const amountMinorUnits = parseAmountToMinorUnits(values.amount, currency)
  if (values.amount.trim() === "") {
    errors.amount = "Enter an amount."
  } else if (amountMinorUnits === null) {
    errors.amount = "Enter a valid amount."
  } else if (amountMinorUnits <= 0) {
    errors.amount = "Amount must be greater than zero."
  }

  if (values.description.trim().length > DESCRIPTION_MAX_LENGTH) {
    errors.description = "Description must be at most 1000 characters."
  }

  const tags = normalizeTags(values.tags)
  const tagError = tagValidationError(tags)
  if (tagError) {
    errors.tags = tagError
  }

  let spentAtIso: string | undefined
  if (values.spentAt.trim() === "") {
    errors.spentAt = "Choose a date."
  } else {
    const parsed = new Date(values.spentAt)
    if (Number.isNaN(parsed.getTime())) {
      errors.spentAt = "Choose a valid date and time."
    } else {
      spentAtIso = parsed.toISOString()
    }
  }

  return {
    errors,
    type,
    amountMinorUnits: amountMinorUnits ?? undefined,
    tags,
    spentAtIso,
  }
}

export function nowLocalDateTime(): string {
  const now = new Date()
  const local = new Date(now.getTime() - now.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}
