import { useQuery } from "@tanstack/react-query"

import { queryKeys } from "@/app/query-keys"
import { getAnalyticsBulkStats } from "@/features/analytics/api"

export function useAnalyticsBulkStats(
  userId: string,
  orgId: string,
  sourceIds: string[],
  from: string,
  to: string,
  enabled: boolean
) {
  const idsKey = [...sourceIds].sort().join(",")
  return useQuery({
    queryKey: queryKeys.analyticsBulkStats(
      userId,
      orgId,
      idsKey === "" ? [] : idsKey.split(","),
      from,
      to
    ),
    queryFn: () => getAnalyticsBulkStats(orgId, sourceIds, from, to),
    enabled: enabled && sourceIds.length > 0,
    staleTime: 60_000,
  })
}
