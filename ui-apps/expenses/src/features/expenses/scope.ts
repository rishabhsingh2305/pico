import { createExpensesOrg, findExpensesOrg } from "@/features/expenses/api"
import type { Org } from "@/features/expenses/types"

let pending: Promise<Org> | null = null
let pendingUserId: string | null = null
let isCreating = false
const creatingListeners = new Set<() => void>()

/**
 * Resolves the internal org tagged "expenses", creating it on first use.
 *
 * The promise is shared per user so React StrictMode or a double mount cannot
 * create two orgs. Retries always re-check first, so a lost create response
 * does not produce a duplicate.
 */
export function ensureExpensesOrg(
  userId: string,
  onCreating?: () => void
): Promise<Org> {
  if (onCreating) {
    creatingListeners.add(onCreating)
    if (isCreating) {
      onCreating()
    }
  }

  if (pending && pendingUserId === userId) {
    return pending
  }

  pendingUserId = userId
  pending = resolve(userId).finally(() => {
    if (pendingUserId === userId) {
      pending = null
      pendingUserId = null
    }
    isCreating = false
    creatingListeners.clear()
  })
  return pending
}

async function resolve(userId: string): Promise<Org> {
  const existing = await findExpensesOrg(userId)
  if (existing) {
    return existing
  }

  isCreating = true
  for (const listener of creatingListeners) {
    listener()
  }
  return createExpensesOrg()
}
