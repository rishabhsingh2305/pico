import { useQuery } from "@tanstack/react-query"

import { queryKeys } from "@/app/query-keys"
import { getAnalyticsStats } from "@/features/analytics/api"
import type { AnalyticsStatsFilters } from "@/features/analytics/types"

export function useAnalyticsStats(
  userId: string,
  orgId: string,
  sourceId: string,
  filters: AnalyticsStatsFilters,
  enabled: boolean
) {
  return useQuery({
    queryKey: queryKeys.analyticsStats(userId, orgId, sourceId, filters),
    queryFn: () => getAnalyticsStats(orgId, sourceId, filters),
    enabled: enabled && sourceId !== "",
    staleTime: 60_000,
  })
}
