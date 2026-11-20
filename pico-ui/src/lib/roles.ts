const OWNER_ROLE = "*"

export function hasFeatureRole(roles: string[], feature: string): boolean {
  return (
    roles.includes(OWNER_ROLE) ||
    roles.some((role) => role.startsWith(`${feature}.`))
  )
}
