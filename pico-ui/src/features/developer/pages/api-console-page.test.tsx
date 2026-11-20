import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ApiConsolePage } from "@/features/developer/pages/api-console-page"
import { AuthTestProvider } from "@/test/auth"

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function renderConsole() {
  return render(
    <AuthTestProvider session={{ userId: "user-1", email: "ada@example.com" }}>
      <MemoryRouter>
        <ApiConsolePage />
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("ApiConsolePage", () => {
  it("shows the base URL and healthy services", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input) => {
        const url = new URL(String(input), "http://localhost")
        if (url.pathname === "/api/ping") {
          return jsonResponse({ message: "pong" })
        }
        if (url.pathname === "/api/ready") {
          return jsonResponse({ status: "ok" })
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )

    renderConsole()

    expect(
      await screen.findByRole("heading", { name: "API console" })
    ).toBeInTheDocument()
    expect(screen.getByText("/api")).toBeInTheDocument()
    expect(await screen.findAllByText("OK")).toHaveLength(2)
    expect(screen.getByText(/curl \/api\/ping/)).toBeInTheDocument()
  })

  it("shows degraded when MongoDB is unavailable", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input) => {
        const url = new URL(String(input), "http://localhost")
        if (url.pathname === "/api/ping") {
          return jsonResponse({ message: "pong" })
        }
        if (url.pathname === "/api/ready") {
          return jsonResponse({ error: "database unavailable" }, 503)
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )

    renderConsole()

    expect(await screen.findByText("Degraded")).toBeInTheDocument()
    expect(screen.getByText("OK")).toBeInTheDocument()
  })

  it("shows offline when the API cannot be reached", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input) => {
        const url = new URL(String(input), "http://localhost")
        if (url.pathname === "/api/ping") {
          throw new TypeError("Failed to fetch")
        }
        if (url.pathname === "/api/ready") {
          return jsonResponse({ status: "ok" })
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )

    renderConsole()

    expect(await screen.findByText("Offline")).toBeInTheDocument()
    expect(screen.getByText(/The API could not be reached/)).toBeInTheDocument()
  })
})
