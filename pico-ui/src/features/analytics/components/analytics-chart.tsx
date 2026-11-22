import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"

import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import type { AnalyticsSeriesItem } from "@/features/analytics/types"

type AnalyticsChartProps = {
  series: AnalyticsSeriesItem[]
  interval: "hour" | "day"
}

export function AnalyticsChart({ series, interval }: AnalyticsChartProps) {
  const data = series.map((item) => ({
    label: formatBucket(item.bucket, interval),
    count: item.count,
  }))

  if (data.length === 0) {
    return (
      <div className="flex min-h-56 items-center justify-center rounded-md border border-dashed border-border text-sm text-muted-foreground">
        No captures in this range.
      </div>
    )
  }

  return (
    <ChartContainer
      config={{
        count: {
          label: "Captures",
          color: "var(--chart-1)",
        },
      }}
      className="aspect-auto h-64 w-full sm:h-72"
      aria-label="Captures over time"
    >
      <AreaChart
        data={data}
        margin={{ left: 0, right: 12, top: 12, bottom: 0 }}
      >
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey="label"
          tickLine={false}
          axisLine={false}
          minTickGap={24}
        />
        <YAxis
          allowDecimals={false}
          tickLine={false}
          axisLine={false}
          width={36}
        />
        <ChartTooltip
          cursor={false}
          content={<ChartTooltipContent labelKey="label" />}
        />
        <Area
          type="monotone"
          dataKey="count"
          stroke="var(--color-count)"
          fill="var(--color-count)"
          fillOpacity={0.18}
          strokeWidth={2}
          isAnimationActive={false}
        />
      </AreaChart>
    </ChartContainer>
  )
}

function formatBucket(bucket: string, interval: "hour" | "day"): string {
  const date = new Date(bucket)
  if (Number.isNaN(date.getTime())) {
    return bucket
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    ...(interval === "hour" ? { hour: "2-digit", minute: "2-digit" } : {}),
  }).format(date)
}
