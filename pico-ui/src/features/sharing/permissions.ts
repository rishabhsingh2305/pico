const SCOPE_LABELS: Record<string, string> = {
  "orgs.get": "Read organization",
  "projects.list": "List projects",
  "projects.get": "Read project",
  "expenses.list": "List expenses",
  "expenses.summary": "Expense totals",
  "expenses.get": "Read expenses",
}

export type ScopeGroup = {
  label: string
  scopes: string[]
}

export function scopeLabel(scope: string): string {
  return SCOPE_LABELS[scope] ?? scope
}

export function groupScopes(scopes: string[]): ScopeGroup[] {
  const groups = new Map<string, string[]>()

  for (const scope of scopes) {
    const label = scope.startsWith("orgs.")
      ? "Organization"
      : scope.startsWith("projects.")
        ? "Projects"
        : scope.startsWith("expenses.")
          ? "Expenses"
          : "Other"
    const existing = groups.get(label)
    if (existing) {
      existing.push(scope)
    } else {
      groups.set(label, [scope])
    }
  }

  return [...groups.entries()].map(([label, items]) => ({
    label,
    scopes: items,
  }))
}

export function accessSummary(
  resourceType: "org" | "project" | "expense"
): string {
  switch (resourceType) {
    case "org":
      return "Read-only organization"
    case "project":
      return "Read-only project"
    case "expense":
      return "Read-only expense"
  }
}
