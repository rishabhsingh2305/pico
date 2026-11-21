import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"

import { queryKeys } from "@/app/query-keys"
import { getLinkStats } from "@/features/links/api"

export function useLinkStats(
  userId: string,
  orgId: string,
  linkIds: string[],
  enabled: boolean
) {
  const idsKey = [...new Set(linkIds)].sort().join(",")
  const ids = useMemo(() => (idsKey === "" ? [] : idsKey.split(",")), [idsKey])

  const statsQuery = useQuery({
    queryKey: queryKeys.linkStats(userId, orgId, ids),
    queryFn: () => getLinkStats(orgId, ids),
    enabled: enabled && ids.length > 0,
    staleTime: 60_000,
  })

  const clicksByLinkId = useMemo(() => {
    const map = new Map<string, number>()
    for (const item of statsQuery.data?.items ?? []) {
      map.set(item.linkId, item.clicks)
    }
    return map
  }, [statsQuery.data])

  return {
    clicksByLinkId,
    isLoading: statsQuery.isLoading,
    isError: statsQuery.isError,
  }
}
