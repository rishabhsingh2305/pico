import { requestJSON } from "@/lib/api-client"
import type { CreateOrgInput, Org } from "@/features/orgs/types"

const ORG_LIST_LIMIT = 100

export async function getOrg(userId: string, orgId: string): Promise<Org> {
  const query = new URLSearchParams({ orgId, userId })
  return requestJSON<Org>(`/orgs/${encodeURIComponent(orgId)}?${query}`, {
    method: "GET",
  })
}

export async function listOrgs(
  userId: string,
  tags: string[] = []
): Promise<Org[]> {
  const query = new URLSearchParams({
    userId,
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

export function createOrg(userId: string, input: CreateOrgInput): Promise<Org> {
  const query = new URLSearchParams({ userId })
  return requestJSON<Org>(`/orgs?${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}

export function updateOrgTags(
  userId: string,
  orgId: string,
  tags: string[]
): Promise<Org> {
  const query = new URLSearchParams({ orgId, userId })
  return requestJSON<Org>(`/orgs/${encodeURIComponent(orgId)}?${query}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tags }),
  })
}
