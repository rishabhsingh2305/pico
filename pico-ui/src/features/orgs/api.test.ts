import { afterEach, describe, expect, it, vi } from "vitest"

import { listOrgs, updateOrgTags } from "@/features/orgs/api"
import type { Org } from "@/features/orgs/types"

const org: Org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: ["team"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
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

describe("listOrgs", () => {
  it("omits the tags filter when none are selected", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [org] }))
    vi.stubGlobal("fetch", fetchMock)

    await listOrgs("user-1")

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.pathname).toBe("/api/orgs")
    expect(parsed.searchParams.get("userId")).toBe("user-1")
    expect(parsed.searchParams.has("tags")).toBe(false)
  })

  it("sends comma-separated filter tags", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [org] }))
    vi.stubGlobal("fetch", fetchMock)

    await listOrgs("user-1", ["team", "finance"])

    const [url] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.searchParams.get("tags")).toBe("team,finance")
  })
})

describe("updateOrgTags", () => {
  it("patches only the organization tags", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(org))
    vi.stubGlobal("fetch", fetchMock)

    await updateOrgTags("user-1", "org-1", ["team", "ops"])

    const [url, init] = fetchMock.mock.calls[0]
    const parsed = new URL(String(url), "http://localhost")
    expect(parsed.pathname).toBe("/api/orgs/org-1")
    expect(parsed.searchParams.get("orgId")).toBe("org-1")
    expect(parsed.searchParams.get("userId")).toBe("user-1")
    expect(init?.method).toBe("PATCH")
    expect(new Headers(init?.headers).get("Content-Type")).toBe(
      "application/json"
    )
    expect(JSON.parse(String(init?.body))).toEqual({ tags: ["team", "ops"] })
  })
})
