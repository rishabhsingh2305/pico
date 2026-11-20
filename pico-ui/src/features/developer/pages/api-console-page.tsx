import { Link } from "react-router"
import { useQuery } from "@tanstack/react-query"
import {
  ArrowUpRightIcon,
  BookOpenIcon,
  CircleAlertIcon,
  RefreshCwIcon,
} from "lucide-react"

import { PageHeader } from "@/components/page-header"
import { CodeBlock } from "@/components/code-snippet"
import { CopyButton } from "@/components/copy-button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { queryKeys } from "@/app/query-keys"
import { checkPing, checkReady } from "@/features/developer/api"
import { usePageTitle } from "@/hooks/use-page-title"
import { API_BASE_URL } from "@/lib/env"

type HealthState = "checking" | "ok" | "unavailable" | "offline"

export function ApiConsolePage() {
  usePageTitle("API console · Pico")

  const pingQuery = useQuery({
    queryKey: queryKeys.healthPing,
    queryFn: checkPing,
    refetchInterval: 30_000,
  })
  const readyQuery = useQuery({
    queryKey: queryKeys.healthReady,
    queryFn: checkReady,
    refetchInterval: 30_000,
  })

  const pingState: HealthState = pingQuery.isPending
    ? "checking"
    : pingQuery.isError
      ? "offline"
      : "ok"
  const readyState: HealthState = readyQuery.isPending
    ? "checking"
    : readyQuery.isError
      ? "offline"
      : readyQuery.data?.status === "ok"
        ? "ok"
        : "unavailable"

  const isRefreshing = pingQuery.isFetching || readyQuery.isFetching

  const quickStart = `# Liveness
curl ${API_BASE_URL}/ping

# Sign in and copy the access token
curl -X POST ${API_BASE_URL}/auth/login \\
  -H 'Content-Type: application/json' \\
  -d '{"email":"ada@example.com","password":"correct horse"}'`

  return (
    <div className="flex flex-col gap-8">
      <PageHeader
        eyebrow="Developer"
        title="API console"
        description="Check the API, copy the base URL, and start making requests. Key management and usage analytics are not available yet."
        actions={
          <Link
            to="/api-docs"
            target="_blank"
            rel="noreferrer"
            className={buttonVariants({
              variant: "outline",
              className:
                "h-10 px-4 text-xs font-semibold tracking-[0.14em] uppercase",
            })}
          >
            <BookOpenIcon data-icon="inline-start" />
            API docs
            <ArrowUpRightIcon data-icon="inline-end" aria-hidden="true" />
          </Link>
        }
      />

      <div className="grid gap-5 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <p className="text-label text-muted-foreground uppercase">
              Connection
            </p>
            <CardTitle className="font-display text-2xl tracking-tight">
              Base URL
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-2 rounded-md border bg-muted/30 px-3 py-2">
              <code className="min-w-0 flex-1 truncate font-mono text-xs">
                {API_BASE_URL}
              </code>
              <CopyButton value={API_BASE_URL} label="Copy API base URL" />
            </div>
            <p className="text-sm text-muted-foreground">
              The browser client calls this origin. Public endpoints such as
              <code className="mx-1 font-mono text-xs">/ping</code>
              do not require a token.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex-row items-start justify-between gap-3">
            <div>
              <p className="text-label text-muted-foreground uppercase">
                Status
              </p>
              <CardTitle className="mt-1 font-display text-2xl tracking-tight">
                Health
              </CardTitle>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={isRefreshing}
              onClick={() => {
                void pingQuery.refetch()
                void readyQuery.refetch()
              }}
            >
              <RefreshCwIcon data-icon="inline-start" />
              Check again
            </Button>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <HealthRow label="API" endpoint="/ping" state={pingState} />
            <HealthRow label="MongoDB" endpoint="/ready" state={readyState} />
            {pingQuery.isError || readyQuery.isError ? (
              <Alert variant="destructive">
                <CircleAlertIcon aria-hidden="true" />
                <AlertDescription>
                  The API could not be reached. Check the origin and network
                  connection.
                </AlertDescription>
              </Alert>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <p className="text-label text-muted-foreground uppercase">
            Quick start
          </p>
          <CardTitle className="font-display text-2xl tracking-tight">
            First requests
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <CodeBlock code={quickStart} label="Shell" />
          <p className="text-sm text-muted-foreground">
            The API docs contain every endpoint, request body, response, and
            permission rule.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

function HealthRow({
  label,
  endpoint,
  state,
}: {
  label: string
  endpoint: string
  state: HealthState
}) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border pb-3 last:border-0 last:pb-0">
      <div className="min-w-0">
        <p className="font-medium">{label}</p>
        <code className="font-mono text-xs text-muted-foreground">
          {endpoint}
        </code>
      </div>
      <HealthBadge state={state} />
    </div>
  )
}

function HealthBadge({ state }: { state: HealthState }) {
  const labels: Record<HealthState, string> = {
    checking: "Checking",
    ok: "OK",
    unavailable: "Degraded",
    offline: "Offline",
  }

  return (
    <Badge
      variant={
        state === "ok"
          ? "default"
          : state === "offline"
            ? "destructive"
            : "secondary"
      }
    >
      {labels[state]}
    </Badge>
  )
}
