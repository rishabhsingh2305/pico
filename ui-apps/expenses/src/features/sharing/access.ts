export function canManageShares(isOwner: boolean, roles: string[]): boolean {
  return isOwner || roles.includes("*") || roles.includes("share.admin")
}

export function canViewShares(isOwner: boolean, roles: string[]): boolean {
  return canManageShares(isOwner, roles) || roles.includes("share.viewer")
}
