import { normalizeTags } from "@/lib/tags"

export type ExpenseFilters = {
  tags: string[]
  from?: string
  to?: string
}

export type ExpenseFilterInput = {
  tags: string[]
  from: string
  to: string
}

export function parseExpenseFilters(input: ExpenseFilterInput): {
  filters?: ExpenseFilters
  error?: string
} {
  const tags = normalizeTags(input.tags).sort()

  let from: string | undefined
  if (input.from.trim() !== "") {
    const parsed = new Date(`${input.from}T00:00:00`)
    if (Number.isNaN(parsed.getTime())) {
      return { error: "Choose a valid from date." }
    }
    from = parsed.toISOString()
  }

  let to: string | undefined
  if (input.to.trim() !== "") {
    const parsed = new Date(`${input.to}T23:59:59.999`)
    if (Number.isNaN(parsed.getTime())) {
      return { error: "Choose a valid to date." }
    }
    to = parsed.toISOString()
  }

  if (from && to && from > to) {
    return { error: "From date must be before the to date." }
  }

  return { filters: { tags, from, to } }
}
