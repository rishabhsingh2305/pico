export type AnalyticsSource = {
  id: string
  orgId: string
  name: string
  description: string
  key: string
  events: string[]
  pixelPath: string
  pixelUrl: string
  disabled: boolean
  createdAt: string
  updatedAt: string
}

export type AnalyticsSourcePage = {
  items: AnalyticsSource[]
  nextOffset: number | null
}

export type CreateAnalyticsSourceInput = {
  name: string
  description?: string
  events: string[]
}

export type AnalyticsStatsFilters = {
  from: string
  to: string
  interval: "hour" | "day"
  event: string
  country: string
  device: string
  bot: string
  referrer: string
}

export type AnalyticsSeriesItem = {
  bucket: string
  count: number
}

export type AnalyticsEventItem = {
  event: string
  count: number
}

export type AnalyticsDimensionItem = {
  value: string
  count: number
}

export type AnalyticsStats = {
  total: number
  series: AnalyticsSeriesItem[]
  events: AnalyticsEventItem[]
  breakdowns: {
    countries: AnalyticsDimensionItem[]
    devices: AnalyticsDimensionItem[]
    bots: AnalyticsDimensionItem[]
    referrers: AnalyticsDimensionItem[]
  }
}

export type AnalyticsBulkStatsItem = {
  sourceId: string
  captures: number
}

export type AnalyticsBulkStatsResponse = {
  items: AnalyticsBulkStatsItem[]
}
