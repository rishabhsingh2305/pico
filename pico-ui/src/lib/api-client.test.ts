import { describe, expect, it, vi } from "vitest"

import {
  getAuthToken,
  getRefreshToken,
  setAuthToken,
  setRefreshToken,
} from "@/lib/auth-token"
import { ApiError, requestJSON } from "@/lib/api-client"

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
  } as Response
}

describe("requestJSON", () => {
  it("parses successful JSON responses", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))
    vi.stubGlobal("fetch", fetchMock)

    await expect(requestJSON("/test", {})).resolves.toEqual({ ok: true })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe("/api/test")
    expect(new Headers(init.headers).get("Accept")).toBe("application/json")
  })

  it("throws ApiError with the server message", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse({ error: "invalid input" }, 400))
    )

    await expect(requestJSON("/test", {})).rejects.toEqual(
      expect.objectContaining({
        name: "ApiError",
        status: 400,
        message: "invalid input",
      })
    )
  })

  it("uses a stable fallback for malformed error bodies", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: async () => "not json",
      } as Response)
    )

    await expect(requestJSON("/test", {})).rejects.toEqual(
      expect.objectContaining({
        name: "ApiError",
        status: 500,
        message: "Request failed",
      })
    )
  })

  it("propagates network failures", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch"))
    )

    await expect(requestJSON("/test", {})).rejects.toBeInstanceOf(TypeError)
  })

  it("returns undefined for empty successful responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(undefined)))

    await expect(requestJSON("/test", {})).resolves.toBeUndefined()
  })

  it("attaches a bearer token when present", async () => {
    setAuthToken("token-1")
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))
    vi.stubGlobal("fetch", fetchMock)

    await requestJSON("/test", {})

    const [, init] = fetchMock.mock.calls[0]
    expect(new Headers(init.headers).get("Authorization")).toBe(
      "Bearer token-1"
    )
  })

  it("exposes the status on ApiError", () => {
    const error = new ApiError(409, "duplicate")

    expect(error.status).toBe(409)
    expect(error.message).toBe("duplicate")
  })

  it("refreshes an expired token once and retries the request", async () => {
    setAuthToken("expired-token")
    setRefreshToken("refresh-1")

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ error: "authentication required" }, 401)
      )
      .mockResolvedValueOnce(
        jsonResponse({ accessToken: "access-2", refreshToken: "refresh-2" })
      )
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
    vi.stubGlobal("fetch", fetchMock)

    await expect(requestJSON("/test", {})).resolves.toEqual({ ok: true })

    expect(fetchMock).toHaveBeenCalledTimes(3)
    const [refreshURL, refreshInit] = fetchMock.mock.calls[1]
    expect(refreshURL).toBe("/api/auth/refresh")
    expect(JSON.parse(String(refreshInit.body))).toEqual({
      refreshToken: "refresh-1",
    })

    const [, retryInit] = fetchMock.mock.calls[2]
    expect(new Headers(retryInit.headers).get("Authorization")).toBe(
      "Bearer access-2"
    )
    expect(getAuthToken()).toBe("access-2")
    expect(getRefreshToken()).toBe("refresh-2")
  })

  it("does not refresh for authentication endpoints", async () => {
    setRefreshToken("refresh-1")

    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: "invalid credentials" }, 401))
    vi.stubGlobal("fetch", fetchMock)

    await expect(
      requestJSON("/auth/login", { method: "POST" })
    ).rejects.toBeInstanceOf(ApiError)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it("clears tokens when the refresh token is rejected", async () => {
    setAuthToken("expired-token")
    setRefreshToken("refresh-1")

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ error: "authentication required" }, 401)
      )
      .mockResolvedValueOnce(
        jsonResponse({ error: "authentication required" }, 401)
      )
    vi.stubGlobal("fetch", fetchMock)

    await expect(requestJSON("/test", {})).rejects.toBeInstanceOf(ApiError)
    expect(getAuthToken()).toBeNull()
    expect(getRefreshToken()).toBeNull()
  })
})
