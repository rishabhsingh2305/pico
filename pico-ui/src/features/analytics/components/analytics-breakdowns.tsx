import { Card, CardContent } from "@/components/ui/card"
import type {
  AnalyticsDimensionItem,
  AnalyticsStats,
} from "@/features/analytics/types"

type AnalyticsBreakdownsProps = {
  stats: AnalyticsStats
}

export function AnalyticsBreakdowns({ stats }: AnalyticsBreakdownsProps) {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <BreakdownCard
        title="Countries"
        items={stats.breakdowns.countries}
        valueLabel={(value) => value}
      />
      <BreakdownCard
        title="Devices"
        items={stats.breakdowns.devices}
        valueLabel={(value) => value}
      />
      <BreakdownCard
        title="Traffic"
        items={stats.breakdowns.bots}
        valueLabel={(value) => (value === "true" ? "Bots" : "Humans")}
      />
      <BreakdownCard
        title="Referrers"
        items={stats.breakdowns.referrers}
        valueLabel={(value) => (value === "direct" ? "Direct" : value)}
      />
    </div>
  )
}

function BreakdownCard({
  title,
  items,
  valueLabel,
}: {
  title: string
  items: AnalyticsDimensionItem[]
  valueLabel: (value: string) => string
}) {
  const top = items.slice(0, 6)
  const max = top[0]?.count ?? 0
  return (
    <Card>
      <CardContent className="flex flex-col gap-3 py-4">
        <h3 className="text-label text-muted-foreground uppercase">{title}</h3>
        {top.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No captures in this range.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {top.map((item) => (
              <li key={item.value} className="flex flex-col gap-1">
                <div className="flex items-center justify-between gap-3 text-sm">
                  <span className="truncate">{valueLabel(item.value)}</span>
                  <span className="text-muted-foreground tabular-nums">
                    {new Intl.NumberFormat().format(item.count)}
                  </span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full bg-chart-1"
                    style={{
                      width: `${max > 0 ? Math.max(4, (item.count / max) * 100) : 0}%`,
                    }}
                  />
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
