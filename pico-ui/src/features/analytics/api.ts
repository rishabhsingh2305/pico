import { requestJSON } from "@/lib/api-client"
import type {
  AnalyticsBulkStatsResponse,
  AnalyticsSource,
  AnalyticsSourcePage,
  AnalyticsStats,
  AnalyticsStatsFilters,
  CreateAnalyticsSourceInput,
} from "@/features/analytics/types"
import { dateRangeToAPI } from "@/features/analytics/filters"

export const ANALYTICS_PAGE_SIZE = 20

export async function listAnalyticsSources(
  orgId: string,
  query: string,
  offset = 0,
  limit = ANALYTICS_PAGE_SIZE
): Promise<AnalyticsSourcePage> {
  const params = new URLSearchParams({ orgId })
  if (query.trim() !== "") {
    params.set("q", query.trim())
  }
  params.set("limit", String(limit))
  params.set("offset", String(offset))
  return requestJSON<AnalyticsSourcePage>(
    `/orgs/${encodeURIComponent(orgId)}/analytics/sources?${params}`,
    { method: "GET" }
  )
}

export function getAnalyticsSource(
  orgId: string,
  sourceId: string
): Promise<AnalyticsSource> {
  const params = new URLSearchParams({ orgId })
  return requestJSON<AnalyticsSource>(
    `/orgs/${encodeURIComponent(orgId)}/analytics/sources/${encodeURIComponent(sourceId)}?${params}`,
    { method: "GET" }
  )
}

export function createAnalyticsSource(
  orgId: string,
  input: CreateAnalyticsSourceInput
): Promise<AnalyticsSource> {
  const params = new URLSearchParams({ orgId })
  return requestJSON<AnalyticsSource>(
    `/orgs/${encodeURIComponent(orgId)}/analytics/sources?${params}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }
  )
}

export function getAnalyticsStats(
  orgId: string,
  sourceId: string,
  filters: AnalyticsStatsFilters
): Promise<AnalyticsStats> {
  const range = dateRangeToAPI(filters.from, filters.to)
  const params = new URLSearchParams({
    orgId,
    from: range.from,
    to: range.to,
    interval: filters.interval,
  })
  for (const key of [
    "event",
    "country",
    "device",
    "bot",
    "referrer",
  ] as const) {
    if (filters[key] !== "") {
      params.set(key, filters[key])
    }
  }
  return requestJSON<AnalyticsStats>(
    `/orgs/${encodeURIComponent(orgId)}/analytics/sources/${encodeURIComponent(sourceId)}/stats?${params}`,
    { method: "GET" }
  )
}

export function getAnalyticsBulkStats(
  orgId: string,
  sourceIds: string[],
  from: string,
  to: string
): Promise<AnalyticsBulkStatsResponse> {
  const range = dateRangeToAPI(from, to)
  const params = new URLSearchParams({
    orgId,
    ids: sourceIds.join(","),
    from: range.from,
    to: range.to,
  })
  return requestJSON<AnalyticsBulkStatsResponse>(
    `/orgs/${encodeURIComponent(orgId)}/analytics/stats?${params}`,
    { method: "GET" }
  )
}
