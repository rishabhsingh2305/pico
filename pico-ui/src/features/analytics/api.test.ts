import { afterEach, describe, expect, it, vi } from "vitest"

import {
  getAnalyticsStats,
  listAnalyticsSources,
} from "@/features/analytics/api"
import { defaultAnalyticsFilters } from "@/features/analytics/filters"

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("listAnalyticsSources", () => {
  it("passes search, pagination, and org id", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [], nextOffset: null }))
    vi.stubGlobal("fetch", fetchMock)

    await listAnalyticsSources("org-1", " home ", 20, 10)

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.pathname).toBe("/api/orgs/org-1/analytics/sources")
    expect(parsed.searchParams.get("q")).toBe("home")
    expect(parsed.searchParams.get("limit")).toBe("10")
    expect(parsed.searchParams.get("offset")).toBe("20")
  })
})

describe("getAnalyticsStats", () => {
  it("sends range and only populated filters", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        jsonResponse({ total: 0, series: [], events: [], breakdowns: {} })
      )
    vi.stubGlobal("fetch", fetchMock)

    await getAnalyticsStats("org-1", "source-1", {
      ...defaultAnalyticsFilters(new Date(2026, 10, 22)),
      event: "click",
      bot: "false",
    })

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.pathname).toBe(
      "/api/orgs/org-1/analytics/sources/source-1/stats"
    )
    expect(parsed.searchParams.get("event")).toBe("click")
    expect(parsed.searchParams.get("bot")).toBe("false")
    expect(parsed.searchParams.get("device")).toBeNull()
    expect(parsed.searchParams.get("country")).toBeNull()
    expect(parsed.searchParams.get("referrer")).toBeNull()
    expect(parsed.searchParams.get("from")).not.toBeNull()
    expect(parsed.searchParams.get("to")).not.toBeNull()
  })
})
