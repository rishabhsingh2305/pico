import type { ExpenseFilters } from "@/features/expenses/filters"

const projectsRoot = (userId: string, orgId: string) =>
  ["projects", userId, orgId] as const

const expenseRoot = (userId: string, orgId: string, projectId: string) =>
  ["expenses", userId, orgId, projectId] as const

function expenseFilterKey(filters: ExpenseFilters) {
  return [filters.tags, filters.from ?? null, filters.to ?? null] as const
}

export const queryKeys = {
  org: (userId: string, orgId: string) => ["org", userId, orgId] as const,
  projectsRoot,
  projects: (userId: string, orgId: string, tags: string[]) =>
    [...projectsRoot(userId, orgId), tags] as const,
  project: (userId: string, orgId: string, projectId: string) =>
    ["project", userId, orgId, projectId] as const,
  expenseRoot,
  expenses: (
    userId: string,
    orgId: string,
    projectId: string,
    filters: ExpenseFilters,
    offset: number
  ) =>
    [
      ...expenseRoot(userId, orgId, projectId),
      "page",
      ...expenseFilterKey(filters),
      offset,
    ] as const,
  expenseSummary: (
    userId: string,
    orgId: string,
    projectId: string,
    filters: ExpenseFilters
  ) =>
    [
      ...expenseRoot(userId, orgId, projectId),
      "summary",
      ...expenseFilterKey(filters),
    ] as const,
}
