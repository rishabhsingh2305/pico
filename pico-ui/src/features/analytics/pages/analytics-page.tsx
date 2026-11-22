import { useCallback, useEffect, useMemo, useState } from "react"
import { useNavigate, useParams } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { BarChart3Icon, CircleAlertIcon } from "lucide-react"

import { PageHeader } from "@/components/page-header"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { queryKeys } from "@/app/query-keys"
import { useAuth, useSession } from "@/features/auth/auth-provider"
import { isSessionInvalidError, type Session } from "@/features/auth/session"
import {
  ANALYTICS_PAGE_SIZE,
  listAnalyticsSources,
} from "@/features/analytics/api"
import { AnalyticsList } from "@/features/analytics/components/analytics-list"
import { CreateAnalyticsDialog } from "@/features/analytics/components/create-analytics-dialog"
import { analyticsErrorMessage } from "@/features/analytics/error-message"
import { defaultAnalyticsRange } from "@/features/analytics/filters"
import { useAnalyticsBulkStats } from "@/features/analytics/hooks/use-analytics-bulk-stats"
import type {
  AnalyticsSource,
  AnalyticsSourcePage,
} from "@/features/analytics/types"
import { useOrgWorkspace } from "@/features/orgs/layouts/org-workspace-layout"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import { usePageTitle } from "@/hooks/use-page-title"

type LoadStatus = "forbidden" | "loading" | "error" | "ready"

const EMPTY_PAGE: AnalyticsSourcePage = { items: [], nextOffset: null }

export function AnalyticsPage() {
  const session = useSession()
  return <AnalyticsView session={session} />
}

function AnalyticsView({ session }: { session: Session }) {
  const { orgId = "" } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { logout } = useAuth()
  const {
    org,
    canViewAnalytics,
    canManageAnalytics,
    sessionInvalid: workspaceSessionInvalid,
  } = useOrgWorkspace()

  const [query, setQuery] = useState("")
  const [offset, setOffset] = useState(0)
  const debouncedQuery = useDebouncedValue(query, 300)
  const range = useMemo(() => defaultAnalyticsRange(), [])

  const sourcesQuery = useQuery({
    queryKey: queryKeys.analyticsSources(
      session.userId,
      orgId,
      debouncedQuery,
      offset
    ),
    queryFn: () => listAnalyticsSources(orgId, debouncedQuery, offset),
    enabled: canViewAnalytics,
  })

  const page = sourcesQuery.data ?? EMPTY_PAGE
  const stats = useAnalyticsBulkStats(
    session.userId,
    orgId,
    page.items.map((source) => source.id),
    range.from,
    range.to,
    canViewAnalytics && page.items.length > 0
  )

  const status: LoadStatus = !canViewAnalytics
    ? "forbidden"
    : sourcesQuery.isPending
      ? "loading"
      : sourcesQuery.isError
        ? "error"
        : "ready"
  const error = sourcesQuery.isError
    ? analyticsErrorMessage(sourcesQuery.error)
    : ""

  usePageTitle("Analytics · Pico")

  const sessionInvalid = useCallback(() => {
    workspaceSessionInvalid()
  }, [workspaceSessionInvalid])

  useEffect(() => {
    if (sourcesQuery.error && isSessionInvalidError(sourcesQuery.error)) {
      void logout()
      navigate("/login", { replace: true })
    }
  }, [logout, navigate, sourcesQuery.error])

  const invalidate = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: queryKeys.analyticsRoot(session.userId, orgId),
    })
  }, [orgId, queryClient, session.userId])

  function handleCreated(source: AnalyticsSource) {
    invalidate()
    navigate(`/orgs/${orgId}/analytics/${source.id}`)
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Pixel analytics"
        title="Analytics"
        description={`Track captures from pixels across ${org.name}.`}
        actions={
          canManageAnalytics ? (
            <CreateAnalyticsDialog
              orgId={orgId}
              onCreated={handleCreated}
              onSessionInvalid={sessionInvalid}
            />
          ) : null
        }
      />

      {status === "forbidden" ? (
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
      ) : (
        <>
          <div className="border-y border-border py-4">
            <Field className="max-w-sm">
              <FieldLabel
                htmlFor="analytics-search"
                className="text-label text-muted-foreground uppercase"
              >
                Search analytics
              </FieldLabel>
              <Input
                id="analytics-search"
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value)
                  setOffset(0)
                }}
                placeholder="Name"
                autoComplete="off"
              />
            </Field>
          </div>

          {status === "loading" ? (
            <Card>
              <CardContent className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground">
                <Spinner />
                Loading analytics...
              </CardContent>
            </Card>
          ) : null}

          {status === "error" ? (
            <Card>
              <CardContent className="flex flex-col gap-4 py-6">
                <Alert variant="destructive">
                  <CircleAlertIcon aria-hidden="true" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void sourcesQuery.refetch()}
                  >
                    Try again
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => navigate(`/orgs/${orgId}`)}
                  >
                    Back to organization
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : null}

          {status === "ready" && page.items.length === 0 ? (
            <div className="flex flex-col items-center gap-3 border-y border-border py-14 text-center">
              <div className="flex size-12 items-center justify-center rounded-md border border-border bg-card text-muted-foreground">
                <BarChart3Icon className="size-5" aria-hidden="true" />
              </div>
              <p className="font-display text-2xl font-semibold tracking-tight">
                {debouncedQuery.trim() !== ""
                  ? "No matching analytics"
                  : "No analytics yet"}
              </p>
              <p className="max-w-sm text-sm text-muted-foreground">
                {debouncedQuery.trim() !== ""
                  ? "Try a different name."
                  : canManageAnalytics
                    ? "Create a pixel and start capturing events."
                    : "No analytics have been created in this organization."}
              </p>
            </div>
          ) : null}

          {status === "ready" && page.items.length > 0 ? (
            <>
              <AnalyticsList
                orgId={orgId}
                sources={page.items}
                capturesBySourceId={
                  new Map(
                    (stats.data?.items ?? []).map((item) => [
                      item.sourceId,
                      item.captures,
                    ])
                  )
                }
                statsLoading={stats.isLoading}
                statsError={stats.isError}
              />
              <nav
                aria-label="Analytics pagination"
                className="flex items-center justify-end gap-2"
              >
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={offset === 0}
                  onClick={() =>
                    setOffset(Math.max(0, offset - ANALYTICS_PAGE_SIZE))
                  }
                >
                  Previous
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={page.nextOffset === null}
                  onClick={() => {
                    if (page.nextOffset !== null) {
                      setOffset(page.nextOffset)
                    }
                  }}
                >
                  Next
                </Button>
              </nav>
            </>
          ) : null}
        </>
      )}
    </div>
  )
}
