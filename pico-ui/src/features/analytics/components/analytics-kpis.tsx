import { Card, CardContent } from "@/components/ui/card"
import type { AnalyticsStats } from "@/features/analytics/types"

type AnalyticsKpisProps = {
  stats: AnalyticsStats
}

export function AnalyticsKpis({ stats }: AnalyticsKpisProps) {
  const botCount =
    stats.breakdowns.bots.find((item) => item.value === "true")?.count ?? 0
  const humanCount = Math.max(0, stats.total - botCount)
  const topEvent = stats.events[0]
  const topCountry = stats.breakdowns.countries[0]
  const topDevice = stats.breakdowns.devices[0]

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <Kpi label="Total captures" value={formatNumber(stats.total)} />
      <Kpi label="Human captures" value={formatNumber(humanCount)} />
      <Kpi label="Bot captures" value={formatNumber(botCount)} />
      <Kpi label="Top event" value={topEvent?.event ?? "—"} />
      <Kpi label="Top country" value={topCountry?.value ?? "—"} />
      <Kpi label="Top device" value={topDevice?.value ?? "—"} />
    </div>
  )
}

function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardContent className="flex flex-col gap-1 py-4">
        <p className="text-label text-muted-foreground uppercase">{label}</p>
        <p className="truncate font-display text-2xl font-semibold tracking-tight tabular-nums">
          {value}
        </p>
      </CardContent>
    </Card>
  )
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat().format(value)
}
