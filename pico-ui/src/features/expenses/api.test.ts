import { afterEach, describe, expect, it, vi } from "vitest"

import { listExpenses, summarizeExpenses } from "@/features/expenses/api"

const emptyPage = {
  items: [],
  nextOffset: null,
}

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

describe("listExpenses", () => {
  it("omits projectId for organization-wide requests", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(emptyPage))
    vi.stubGlobal("fetch", fetchMock)

    await listExpenses({
      userId: "user-1",
      orgId: "org-1",
      tags: [],
      limit: 6,
      offset: 0,
    })

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.searchParams.has("projectId")).toBe(false)
    expect(parsed.searchParams.get("orgId")).toBe("org-1")
    expect(parsed.searchParams.get("limit")).toBe("6")
  })

  it("includes projectId when one is provided", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(emptyPage))
    vi.stubGlobal("fetch", fetchMock)

    await listExpenses({
      userId: "user-1",
      orgId: "org-1",
      projectId: "project-1",
      tags: [],
      limit: 1,
      offset: 0,
    })

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.searchParams.get("projectId")).toBe("project-1")
  })
})

describe("summarizeExpenses", () => {
  it("requests grouped totals with the same filters", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
    vi.stubGlobal("fetch", fetchMock)

    await summarizeExpenses({
      userId: "user-1",
      orgId: "org-1",
      projectId: "project-1",
      tags: ["food", "travel"],
      from: "2026-01-01T00:00:00Z",
      to: "2026-01-31T23:59:59Z",
    })

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.pathname).toBe("/api/expenses/summary")
    expect(parsed.searchParams.get("projectId")).toBe("project-1")
    expect(parsed.searchParams.get("tags")).toBe("food,travel")
    expect(parsed.searchParams.get("from")).toBe("2026-01-01T00:00:00Z")
    expect(parsed.searchParams.get("to")).toBe("2026-01-31T23:59:59Z")
    expect(parsed.searchParams.has("limit")).toBe(false)
  })

  it("sends multiple project ids instead of a single project", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
    vi.stubGlobal("fetch", fetchMock)

    await summarizeExpenses({
      userId: "user-1",
      orgId: "org-1",
      projectIds: ["project-1", "project-2"],
      tags: [],
    })

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.searchParams.get("projectIds")).toBe("project-1,project-2")
    expect(parsed.searchParams.has("projectId")).toBe(false)
  })
})
