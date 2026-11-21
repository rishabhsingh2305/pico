const SHARE_TOKEN_KEY = "expenses.shareToken"

export type ShareLink = {
  token: string
  orgId: string
  resourceType: "org" | "project" | "expense"
  resourceId: string
  projectId?: string
}

export function parseShareLink(): ShareLink | null {
  const hash = new URLSearchParams(window.location.hash.replace(/^#/, ""))
  const token = hash.get("share") ?? getShareToken()
  if (!token) {
    return null
  }
  const query = new URLSearchParams(window.location.search)
  const resourceType = query.get("resourceType")
  if (
    resourceType !== "org" &&
    resourceType !== "project" &&
    resourceType !== "expense"
  ) {
    return null
  }
  return {
    token,
    orgId: query.get("orgId") ?? "",
    resourceType,
    resourceId: query.get("resourceId") ?? "",
    projectId: query.get("projectId") ?? undefined,
  }
}

export function acquireShareLink(): ShareLink | null {
  const link = parseShareLink()
  if (link) {
    setShareToken(link.token)
  }
  return link
}

export function getShareToken(): string | null {
  try {
    return sessionStorage.getItem(SHARE_TOKEN_KEY)
  } catch {
    return null
  }
}

export function setShareToken(token: string): void {
  try {
    sessionStorage.setItem(SHARE_TOKEN_KEY, token)
  } catch {
    // Storage can be unavailable in private contexts.
  }
}

export function clearShareToken(): void {
  try {
    sessionStorage.removeItem(SHARE_TOKEN_KEY)
  } catch {
    // Nothing to clear.
  }
}
