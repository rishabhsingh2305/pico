import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { AppRouter } from "@/app/router"
import { storeSession } from "@/features/auth/session"

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

describe("AppRouter", () => {
  it("redirects the root path to signup", async () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Create your account" })
    ).toBeInTheDocument()
  })

  it("redirects the root path to organizations with a session", async () => {
    storeSession({ userId: "user-1", email: "ada@example.com" })
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValueOnce(jsonResponse({ items: [] }))
    )

    render(
      <MemoryRouter initialEntries={["/"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Organizations" })
    ).toBeInTheDocument()
    expect(await screen.findByText("No organizations yet")).toBeInTheDocument()
  })

  it("renders the organization dashboard route", async () => {
    storeSession({ userId: "user-1", email: "ada@example.com" })
    const org = {
      id: "org-1",
      name: "Acme",
      userId: "user-1",
      tags: [],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    }
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input, init) => {
        const url = new URL(String(input), "http://localhost")
        const method = init?.method ?? "GET"
        if (method === "GET" && url.pathname === "/api/orgs/org-1") {
          return jsonResponse(org)
        }
        if (method === "GET" && url.pathname === "/api/projects") {
          return jsonResponse({ items: [] })
        }
        if (method === "GET" && url.pathname === "/api/expenses") {
          return jsonResponse({
            items: [],
            totalCount: 0,
            totalDebit: 0,
            totalCredit: 0,
            netTotal: 0,
          })
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )

    render(
      <MemoryRouter initialEntries={["/orgs/org-1/dashboard"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Dashboard" })
    ).toBeInTheDocument()
    expect(await screen.findByText("Nothing to summarize")).toBeInTheDocument()
  })

  it("renders the public API docs route", async () => {
    render(
      <MemoryRouter initialEntries={["/api-docs"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Pico API" })
    ).toBeInTheDocument()
    expect(
      screen.getByRole("link", { name: "Create an account" })
    ).toBeInTheDocument()
  })

  it("renders the public design system route", async () => {
    render(
      <MemoryRouter initialEntries={["/design"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Design system" })
    ).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "Palette" })).toBeInTheDocument()
  })

  it("renders the signup route", async () => {
    render(
      <MemoryRouter initialEntries={["/signup"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Create your account" })
    ).toBeInTheDocument()
  })

  it("renders the verification route", async () => {
    render(
      <MemoryRouter initialEntries={["/verify-email"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", {
        name: "Verification link incomplete",
      })
    ).toBeInTheDocument()
  })

  it("renders the organization detail route", async () => {
    storeSession({ userId: "user-1", email: "ada@example.com" })
    const org = {
      id: "org-1",
      name: "Acme",
      userId: "user-1",
      tags: [],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    }
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input, init) => {
        const url = new URL(String(input), "http://localhost")
        const method = init?.method ?? "GET"
        if (method === "GET" && url.pathname === "/api/orgs/org-1") {
          return jsonResponse(org)
        }
        if (method === "GET" && url.pathname === "/api/projects") {
          return jsonResponse({ items: [] })
        }
        if (method === "GET" && url.pathname === "/api/expenses") {
          return jsonResponse({ items: [] })
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )

    render(
      <MemoryRouter initialEntries={["/orgs/org-1"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Acme" })
    ).toBeInTheDocument()
    expect(await screen.findByText("No projects yet")).toBeInTheDocument()
  })

  it("renders the project detail route", async () => {
    storeSession({ userId: "user-1", email: "ada@example.com" })
    const org = {
      id: "org-1",
      name: "Acme",
      userId: "user-1",
      tags: [],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    }
    const project = {
      id: "project-1",
      orgId: "org-1",
      userId: "user-1",
      name: "General",
      currency: "USD",
      tags: [],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    }
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(async (input, init) => {
        const url = new URL(String(input), "http://localhost")
        const method = init?.method ?? "GET"
        if (method === "GET" && url.pathname === "/api/orgs/org-1") {
          return jsonResponse(org)
        }
        if (method === "GET" && url.pathname === "/api/projects/project-1") {
          return jsonResponse(project)
        }
        if (method === "GET" && url.pathname === "/api/expenses") {
          return jsonResponse({ items: [] })
        }
        return jsonResponse({ error: "not found" }, 404)
      })
    )

    render(
      <MemoryRouter initialEntries={["/orgs/org-1/projects/project-1"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "General" })
    ).toBeInTheDocument()
    expect(await screen.findByText("No expenses yet")).toBeInTheDocument()
  })

  it("redirects unknown paths to signup", async () => {
    render(
      <MemoryRouter initialEntries={["/missing"]}>
        <AppRouter />
      </MemoryRouter>
    )

    expect(
      await screen.findByRole("heading", { name: "Create your account" })
    ).toBeInTheDocument()
  })
})
