import { afterEach, describe, expect, it, vi } from "vitest"

import {
  getAuthToken,
  getRefreshToken,
  setAuthToken,
  setRefreshToken,
} from "@/lib/auth-token"
import { clearShareToken, setShareToken } from "@/lib/share-token"
import { ApiError, requestJSON } from "@/lib/api-client"

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  clearShareToken()
  setAuthToken("")
  setRefreshToken("")
})

describe("api client", () => {
  it("uses the share credential instead of the bearer token", async () => {
    setAuthToken("user-token")
    setShareToken("share-token")
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))
    vi.stubGlobal("fetch", fetchMock)

    await requestJSON("/expenses?orgId=org-1", {})

    const [, init] = fetchMock.mock.calls[0]
    expect(new Headers(init.headers).get("Authorization")).toBe(
      "Share share-token"
    )
  })

  it("does not use the share credential on auth routes", async () => {
    setAuthToken("user-token")
    setShareToken("share-token")
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))
    vi.stubGlobal("fetch", fetchMock)

    await requestJSON("/auth/session", {})

    const [, init] = fetchMock.mock.calls[0]
    expect(new Headers(init.headers).get("Authorization")).toBe(
      "Bearer user-token"
    )
  })

  it("does not refresh a failed share request", async () => {
    setShareToken("share-token")
    setRefreshToken("refresh-token")
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: "expired" }, 401))
    vi.stubGlobal("fetch", fetchMock)

    await expect(
      requestJSON("/expenses?orgId=org-1", {})
    ).rejects.toBeInstanceOf(ApiError)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("exposes the token accessors", () => {
    setAuthToken("user-token")
    setRefreshToken("refresh-token")
    expect(getAuthToken()).toBe("user-token")
    expect(getRefreshToken()).toBe("refresh-token")
  })
})
