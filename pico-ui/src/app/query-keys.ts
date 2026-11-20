import type { ExpenseFilters } from "@/features/expenses/filters"
import type { LinkFilters } from "@/features/links/types"

const projectsRoot = (userId: string, orgId: string) =>
  ["projects", userId, orgId] as const

const expenseRoot = (userId: string, orgId: string, projectId: string) =>
  ["expenses", userId, orgId, projectId] as const

const linksRoot = (userId: string, orgId: string) =>
  ["links", userId, orgId] as const

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
  linksRoot,
  links: (
    userId: string,
    orgId: string,
    filters: LinkFilters,
    offset: number
  ) =>
    [
      ...linksRoot(userId, orgId),
      filters.q,
      [...filters.tags].sort(),
      offset,
    ] as const,
  shareStatus: (
    orgId: string,
    resourceType: "org" | "project" | "expense",
    ids: string[]
  ) => ["share-status", orgId, resourceType, [...ids].sort()] as const,
  shares: (
    orgId: string,
    resourceType: "org" | "project" | "expense",
    resourceId: string
  ) => ["shares", orgId, resourceType, resourceId] as const,
}
