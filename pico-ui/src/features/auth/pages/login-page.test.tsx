import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { LoginPage } from "@/features/auth/pages/login-page"
import { getAuthToken } from "@/lib/auth-token"
import { AuthTestProvider } from "@/test/auth"

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function renderPage() {
  return render(
    <AuthTestProvider session={null}>
      <MemoryRouter initialEntries={["/login"]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/orgs" element={<div>Organizations destination</div>} />
        </Routes>
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("LoginPage", () => {
  it("signs in and continues to organizations", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValueOnce(
      jsonResponse({
        accessToken: "token-1",
        tokenType: "Bearer",
        user: {
          id: "user-1",
          name: "Ada Lovelace",
          email: "ada@example.com",
          emailVerified: true,
          createdAt: "2026-01-01T00:00:00Z",
          updatedAt: "2026-01-01T00:00:00Z",
        },
        expiresAt: "2026-01-08T00:00:00Z",
      })
    )
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()

    renderPage()

    await user.type(screen.getByLabelText("Email"), "ada@example.com")
    await user.type(screen.getByLabelText("Password"), "correct horse")
    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(
      await screen.findByText("Organizations destination")
    ).toBeInTheDocument()

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe("/api/auth/login")
    expect(init?.method).toBe("POST")
    expect(init?.credentials).toBe("omit")
    expect(getAuthToken()).toBe("token-1")
    expect(JSON.parse(String(init?.body))).toEqual({
      email: "ada@example.com",
      password: "correct horse",
    })
  })

  it("validates required fields", async () => {
    const user = userEvent.setup()
    renderPage()

    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(screen.getByText("Enter your email address.")).toBeInTheDocument()
    expect(screen.getByText("Enter your password.")).toBeInTheDocument()
  })

  it("maps invalid credentials", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(
          jsonResponse({ error: "invalid email or password" }, 401)
        )
    )
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText("Email"), "ada@example.com")
    await user.type(screen.getByLabelText("Password"), "wrong")
    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(
      await screen.findByText("Email or password is incorrect.")
    ).toBeInTheDocument()
  })

  it("asks unverified users to verify their email", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(
          jsonResponse({ error: "email not verified" }, 403)
        )
    )
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText("Email"), "ada@example.com")
    await user.type(screen.getByLabelText("Password"), "correct horse")
    await user.click(screen.getByRole("button", { name: "Sign in" }))

    expect(
      await screen.findByText("Verify your email before signing in.")
    ).toBeInTheDocument()
  })
})
