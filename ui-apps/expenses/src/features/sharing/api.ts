import { requestJSON } from "@/lib/api-client"
import type {
  CreateShareInput,
  ShareCredential,
  ShareResourceType,
  ShareStatusItem,
} from "@/features/sharing/types"

export function createShare(
  orgId: string,
  input: CreateShareInput
): Promise<ShareCredential> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<ShareCredential>(`/orgs/${orgId}/shares?${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
}

export async function listShares(
  orgId: string,
  resourceType: ShareResourceType,
  resourceId: string
): Promise<ShareCredential[]> {
  const query = new URLSearchParams({
    orgId,
    kind: "share",
    resourceType,
    resourceIds: resourceId,
  })
  const response = await requestJSON<{ items: ShareCredential[] }>(
    `/orgs/${orgId}/credentials?${query}`,
    { method: "GET" }
  )
  return response.items
}

export async function getShareStatus(
  orgId: string,
  resourceType: ShareResourceType,
  resourceIds: string[]
): Promise<ShareStatusItem[]> {
  if (resourceIds.length === 0) {
    return []
  }
  const query = new URLSearchParams({
    orgId,
    resourceType,
    resourceIds: resourceIds.join(","),
  })
  const response = await requestJSON<{ items: ShareStatusItem[] }>(
    `/orgs/${orgId}/shares/status?${query}`,
    { method: "GET" }
  )
  return response.items
}

export function revokeShare(orgId: string, shareId: string): Promise<void> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<void>(
    `/orgs/${orgId}/shares/${encodeURIComponent(shareId)}?${query}`,
    { method: "DELETE" }
  )
}

export function buildShareUrl(
  token: string,
  orgId: string,
  resourceType: ShareResourceType,
  resourceId: string,
  projectId?: string
): string {
  const query = new URLSearchParams({
    orgId,
    resourceType,
    resourceId,
  })
  if (projectId) {
    query.set("projectId", projectId)
  }
  return `${window.location.origin}/shared?${query}#share=${encodeURIComponent(token)}`
}
