import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { getSession } from "@/features/auth/session"
import { VerifyEmailPage } from "@/features/auth/pages/verify-email-page"

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function renderPage(hash = "") {
  return render(
    <MemoryRouter initialEntries={[`/verify-email${hash}`]}>
      <VerifyEmailPage />
    </MemoryRouter>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("VerifyEmailPage", () => {
  it("verifies the token from the URL fragment", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(
      jsonResponse({
        status: "verified",
        userId: "user-1",
        email: "ada@example.com",
      })
    )
    vi.stubGlobal("fetch", fetchMock)

    renderPage("#token=token-1")

    expect(
      await screen.findByRole("heading", { name: "Email verified" })
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Continue to organizations" })
    ).toBeInTheDocument()
    expect(getSession()).toEqual({
      userId: "user-1",
      email: "ada@example.com",
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe("/api/auth/verify-email")
    expect(init?.method).toBe("POST")
    expect(new Headers(init?.headers).get("Content-Type")).toBe(
      "application/json"
    )
    expect(JSON.parse(String(init?.body))).toEqual({ token: "token-1" })
  })

  it("shows a missing-token state without calling the API", () => {
    const fetchMock = vi.fn<typeof fetch>()
    vi.stubGlobal("fetch", fetchMock)

    renderPage()

    expect(
      screen.getByRole("heading", { name: "Verification link incomplete" })
    ).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("shows an expired-link state for a 400 response", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(jsonResponse({ error: "invalid token" }, 400))
    )

    renderPage("#token=expired")

    expect(
      await screen.findByRole("heading", { name: "Invalid verification link" })
    ).toBeInTheDocument()
  })

  it("shows a network error and can retry", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValueOnce(
        jsonResponse({
          status: "verified",
          userId: "user-1",
          email: "ada@example.com",
        })
      )
    vi.stubGlobal("fetch", fetchMock)
    const userEventInstance = userEvent.setup()

    renderPage("#token=token-1")

    expect(
      await screen.findByRole("heading", {
        name: "Could not verify your email",
      })
    ).toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "Try again" })
    )

    expect(
      await screen.findByRole("heading", { name: "Email verified" })
    ).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })
})
