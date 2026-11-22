import { useCallback, useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router"
import { useQuery } from "@tanstack/react-query"
import { ArrowLeftIcon, CircleAlertIcon } from "lucide-react"

import { CopyButton } from "@/components/copy-button"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { queryKeys } from "@/app/query-keys"
import { getAnalyticsSource } from "@/features/analytics/api"
import { AnalyticsBreakdowns } from "@/features/analytics/components/analytics-breakdowns"
import { AnalyticsChart } from "@/features/analytics/components/analytics-chart"
import { AnalyticsFilters } from "@/features/analytics/components/analytics-filters"
import { AnalyticsKpis } from "@/features/analytics/components/analytics-kpis"
import { analyticsErrorMessage } from "@/features/analytics/error-message"
import { defaultAnalyticsFilters } from "@/features/analytics/filters"
import { useAnalyticsStats } from "@/features/analytics/hooks/use-analytics-stats"
import type {
  AnalyticsStats,
  AnalyticsStatsFilters,
} from "@/features/analytics/types"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import { useSession } from "@/features/auth/auth-provider"
import { useOrgWorkspace } from "@/features/orgs/layouts/org-workspace-layout"
import { usePageTitle } from "@/hooks/use-page-title"

const EMPTY_STATS: AnalyticsStats = {
  total: 0,
  series: [],
  events: [],
  breakdowns: {
    countries: [],
    devices: [],
    bots: [],
    referrers: [],
  },
}

export function AnalyticsDetailPage() {
  const session = useSession()
  return <AnalyticsDetailView session={session} />
}

function AnalyticsDetailView({ session }: { session: Session }) {
  const { orgId = "", sourceId = "" } = useParams<{
    orgId: string
    sourceId: string
  }>()
  const navigate = useNavigate()
  const { canViewAnalytics, sessionInvalid } = useOrgWorkspace()
  const [filters, setFilters] = useState<AnalyticsStatsFilters>(() =>
    defaultAnalyticsFilters()
  )

  const sourceQuery = useQuery({
    queryKey: queryKeys.analyticsSource(session.userId, orgId, sourceId),
    queryFn: () => getAnalyticsSource(orgId, sourceId),
    enabled: canViewAnalytics && sourceId !== "",
  })
  const source = sourceQuery.data
  const statsQuery = useAnalyticsStats(
    session.userId,
    orgId,
    sourceId,
    filters,
    canViewAnalytics && source !== undefined
  )

  usePageTitle(
    source ? `${source.name} · Analytics · Pico` : "Analytics · Pico"
  )

  useEffect(() => {
    if (sourceQuery.error && isSessionInvalidError(sourceQuery.error)) {
      sessionInvalid()
    }
  }, [sessionInvalid, sourceQuery.error])

  const handleFilters = useCallback((next: AnalyticsStatsFilters) => {
    setFilters(next)
  }, [])

  if (!canViewAnalytics) {
    return (
      <Card>
        <CardContent className="flex flex-col gap-4 py-8">
          <Alert>
            <CircleAlertIcon aria-hidden="true" />
            <AlertDescription>
              You do not have access to analytics in this organization.
            </AlertDescription>
          </Alert>
          <div>
            <Button
              type="button"
              variant="outline"
              onClick={() => navigate(`/orgs/${orgId}`)}
            >
              Back to organization
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (sourceQuery.isPending) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
          <Spinner />
          Loading analytics...
        </CardContent>
      </Card>
    )
  }

  if (sourceQuery.isError || !source) {
    return (
      <Card>
        <CardContent className="flex flex-col gap-4 py-6">
          <Alert variant="destructive">
            <CircleAlertIcon aria-hidden="true" />
            <AlertDescription>
              {sourceQuery.isError
                ? analyticsErrorMessage(sourceQuery.error)
                : "Analytics source not found."}
            </AlertDescription>
          </Alert>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => void sourceQuery.refetch()}
            >
              Try again
            </Button>
            <Link
              to={`/orgs/${orgId}/analytics`}
              className={buttonVariants({ variant: "ghost" })}
            >
              Back to analytics
            </Link>
          </div>
        </CardContent>
      </Card>
    )
  }

  const stats = statsQuery.data ?? EMPTY_STATS
  const statsPending = statsQuery.isPending
  const statsError = statsQuery.isError

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link
          to={`/orgs/${orgId}/analytics`}
          className="inline-flex items-center gap-1 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
        >
          <ArrowLeftIcon className="size-3.5" aria-hidden="true" />
          Analytics
        </Link>
      </div>

      <PageHeader
        eyebrow="Analytics source"
        title={source.name}
        description={
          source.description ||
          "Capture counts by event, time, and traffic dimensions."
        }
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={source.disabled ? "secondary" : "default"}>
              {source.disabled ? "Disabled" : "Active"}
            </Badge>
            <CopyButton
              value={source.pixelUrl}
              label={`Copy pixel URL for ${source.name}`}
            />
          </div>
        }
      />

      {source.disabled ? (
        <Alert>
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>
            This source is disabled. Existing captures remain, but new captures
            are not counted.
          </AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardContent className="flex flex-col gap-3 py-4">
          <p className="text-label text-muted-foreground uppercase">
            Pixel URL
          </p>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <code className="overflow-x-auto rounded-md bg-muted px-3 py-2 font-mono text-xs">
              {source.pixelUrl}
            </code>
            <CopyButton
              value={source.pixelUrl}
              label={`Copy pixel URL for ${source.name}`}
            />
          </div>
          <p className="text-sm text-muted-foreground">
            Embed{" "}
            <code className="font-mono">{source.pixelPath}?event=view</code> or
            fire the URL from JavaScript for other actions.
          </p>
          <div className="flex flex-wrap gap-1">
            {source.events.map((event) => (
              <Badge key={event} variant="outline">
                {event}
              </Badge>
            ))}
          </div>
        </CardContent>
      </Card>

      {statsError ? (
        <Alert variant="destructive">
          <CircleAlertIcon aria-hidden="true" />
          <AlertDescription>
            {analyticsErrorMessage(statsQuery.error)}
          </AlertDescription>
        </Alert>
      ) : null}

      <AnalyticsKpis stats={stats} />

      <AnalyticsFilters
        events={source.events}
        value={filters}
        onChange={handleFilters}
      />

      <Card>
        <CardContent className="flex flex-col gap-4 py-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="font-display text-xl font-semibold tracking-tight">
              Captures over time
            </h2>
            <span className="text-xs text-muted-foreground">
              {filters.interval === "hour" ? "Hourly" : "Daily"}
            </span>
          </div>
          {statsPending ? (
            <div className="flex min-h-56 items-center justify-center gap-2 text-sm text-muted-foreground">
              <Spinner />
              Loading captures...
            </div>
          ) : (
            <AnalyticsChart series={stats.series} interval={filters.interval} />
          )}
        </CardContent>
      </Card>

      <AnalyticsBreakdowns stats={stats} />
    </div>
  )
}
