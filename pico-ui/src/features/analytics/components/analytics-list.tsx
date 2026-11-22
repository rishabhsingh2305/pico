import { ExternalLinkIcon } from "lucide-react"
import { Link } from "react-router"

import { CopyButton } from "@/components/copy-button"
import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { AnalyticsSource } from "@/features/analytics/types"

type AnalyticsListProps = {
  orgId: string
  sources: AnalyticsSource[]
  capturesBySourceId: ReadonlyMap<string, number>
  statsLoading: boolean
  statsError: boolean
}

export function AnalyticsList({
  orgId,
  sources,
  capturesBySourceId,
  statsLoading,
  statsError,
}: AnalyticsListProps) {
  if (sources.length === 0) {
    return null
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Analytics</TableHead>
          <TableHead>Pixel</TableHead>
          <TableHead className="text-right">Captures (30d)</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sources.map((source) => (
          <TableRow key={source.id}>
            <TableCell className="max-w-[18rem]">
              <Link
                to={`/orgs/${orgId}/analytics/${source.id}`}
                className="truncate font-medium underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
              >
                {source.name}
              </Link>
              {source.description ? (
                <p className="truncate text-xs text-muted-foreground">
                  {source.description}
                </p>
              ) : null}
            </TableCell>
            <TableCell>
              <span className="font-mono text-xs text-muted-foreground">
                {source.pixelPath}
              </span>
            </TableCell>
            <TableCell className="text-right text-muted-foreground tabular-nums">
              <CaptureCount
                count={capturesBySourceId.get(source.id)}
                loading={statsLoading}
                error={statsError}
              />
            </TableCell>
            <TableCell>
              <Badge variant={source.disabled ? "secondary" : "default"}>
                {source.disabled ? "Disabled" : "Active"}
              </Badge>
            </TableCell>
            <TableCell className="text-muted-foreground">
              {formatCreatedAt(source.createdAt)}
            </TableCell>
            <TableCell>
              <div className="flex items-center justify-end gap-0.5">
                <CopyButton
                  value={source.pixelUrl}
                  label={`Copy pixel URL for ${source.name}`}
                />
                <Link
                  to={`/orgs/${orgId}/analytics/${source.id}`}
                  aria-label={`Open ${source.name}`}
                  title="Open dashboard"
                  className={buttonVariants({
                    variant: "ghost",
                    size: "icon-sm",
                  })}
                >
                  <ExternalLinkIcon aria-hidden="true" />
                </Link>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function CaptureCount({
  count,
  loading,
  error,
}: {
  count: number | undefined
  loading: boolean
  error: boolean
}) {
  if (loading) {
    return <span aria-label="Loading captures">—</span>
  }
  if (error) {
    return (
      <span title="Could not load captures" aria-label="Captures unavailable">
        —
      </span>
    )
  }
  return <>{new Intl.NumberFormat().format(count ?? 0)}</>
}

function formatCreatedAt(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return "recently"
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
  }).format(date)
}
