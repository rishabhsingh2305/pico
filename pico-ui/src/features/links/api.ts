import { requestJSON } from "@/lib/api-client"
import type {
  CreateLinkInput,
  LinkFilters,
  LinkPage,
  ShortLink,
  UpdateLinkInput,
} from "@/features/links/types"

export const LINK_PAGE_SIZE = 20

function linkQuery(orgId: string, filters: LinkFilters): URLSearchParams {
  const query = new URLSearchParams({ orgId })
  if (filters.q.trim() !== "") {
    query.set("q", filters.q.trim())
  }
  if (filters.tags.length > 0) {
    query.set("tags", filters.tags.join(","))
  }
  return query
}

export async function listLinks(
  orgId: string,
  filters: LinkFilters,
  offset = 0,
  limit = LINK_PAGE_SIZE
): Promise<LinkPage> {
  const query = linkQuery(orgId, filters)
  query.set("limit", String(limit))
  query.set("offset", String(offset))
  return requestJSON<LinkPage>(
    `/orgs/${encodeURIComponent(orgId)}/links?${query}`,
    { method: "GET" }
  )
}

export function createLink(
  orgId: string,
  input: CreateLinkInput
): Promise<ShortLink> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<ShortLink>(
    `/orgs/${encodeURIComponent(orgId)}/links?${query}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }
  )
}

export function updateLink(
  orgId: string,
  linkId: string,
  input: UpdateLinkInput
): Promise<ShortLink> {
  const query = new URLSearchParams({ orgId })
  return requestJSON<ShortLink>(
    `/orgs/${encodeURIComponent(orgId)}/links/${encodeURIComponent(linkId)}?${query}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }
  )
}

export async function deleteLink(orgId: string, linkId: string): Promise<void> {
  const query = new URLSearchParams({ orgId })
  await requestJSON<void>(
    `/orgs/${encodeURIComponent(orgId)}/links/${encodeURIComponent(linkId)}?${query}`,
    { method: "DELETE" }
  )
}
