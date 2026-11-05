import { afterEach, describe, expect, it, vi } from "vitest"

import { listExpenses } from "@/features/expenses/api"

const emptyPage = {
  items: [],
  totalCount: 0,
  totalDebit: 0,
  totalCredit: 0,
  netTotal: 0,
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
