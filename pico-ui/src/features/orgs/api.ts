import { refreshTokens, requestJSON } from "@/lib/api-client"
import type { CreateOrgInput, Org } from "@/features/orgs/types"

const ORG_LIST_LIMIT = 100

export async function getOrg(orgId: string): Promise<Org> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<Org>(`/orgs/${encodeURIComponent(orgId)}?${query}`, {
    method: "GET",
  })
}

export async function listOrgs(tags: string[] = []): Promise<Org[]> {
  const query = new URLSearchParams({
    limit: String(ORG_LIST_LIMIT),
  })
  if (tags.length > 0) {
    query.set("tags", tags.join(","))
  }
  const response = await requestJSON<{ items: Org[] }>(`/orgs?${query}`, {
    method: "GET",
  })
  return response.items
}

export async function createOrg(input: CreateOrgInput): Promise<Org> {
  const org = await requestJSON<Org>("/orgs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
  // The new organization is not in the current access token yet. Refresh so
  // the owner wildcard is available before the UI navigates to it.
  await refreshTokens()
  return org
}

export function updateOrgTags(orgId: string, tags: string[]): Promise<Org> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<Org>(`/orgs/${encodeURIComponent(orgId)}?${query}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tags }),
  })
}
