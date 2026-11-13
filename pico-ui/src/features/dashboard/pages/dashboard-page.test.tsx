import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { Session } from "@/features/auth/session"
import { RequireSession } from "@/features/auth/auth-provider"
import { OrgDashboardPage } from "@/features/dashboard/pages/dashboard-page"
import { formatMoney } from "@/features/expenses/money"
import type { Expense } from "@/features/expenses/types"
import type { Org } from "@/features/orgs/types"
import type { Project } from "@/features/projects/types"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

const acme: Org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: [],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const general: Project = {
  id: "project-1",
  orgId: "org-1",
  userId: "user-1",
  name: "General",
  currency: "USD",
  tags: ["travel"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const berlin: Project = {
  id: "project-2",
  orgId: "org-1",
  userId: "user-1",
  name: "Berlin",
  currency: "EUR",
  tags: [],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const clientPayment: Expense = {
  id: "expense-1",
  orgId: "org-1",
  userId: "user-1",
  projectId: general.id,
  type: "credit",
  amount: 500000,
  currency: "USD",
  description: "Client payment",
  tags: ["income"],
  spentAt: "2026-11-04T09:00:00Z",
  createdAt: "2026-11-04T09:00:00Z",
  updatedAt: "2026-11-04T09:00:00Z",
}

type Totals = {
  totalCount: number
  totalDebit: number
  totalCredit: number
  netTotal: number
}

const usdTotals: Totals = {
  totalCount: 2,
  totalDebit: 5000,
  totalCredit: 15000,
  netTotal: 10000,
}

const eurTotals: Totals = {
  totalCount: 1,
  totalDebit: 3000,
  totalCredit: 8000,
  netTotal: 5000,
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
  } as Response
}

type ApiOptions = {
  orgResponse?: Response
  projects?: Project[]
  totals?: Record<string, Totals>
  recent?: Expense[]
}

function apiHandler(options: ApiOptions = {}) {
  return async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost")
    const method = init?.method ?? "GET"

    if (method !== "GET") {
      return jsonResponse({ error: "not found" }, 404)
    }

    if (url.pathname === "/api/orgs/org-1") {
      return options.orgResponse ?? jsonResponse(acme)
    }

    if (url.pathname === "/api/projects") {
      return jsonResponse({ items: options.projects ?? [] })
    }

    if (url.pathname === "/api/expenses/summary") {
      return jsonResponse({
        items: Object.entries(options.totals ?? {}).map(
          ([projectId, totals]) => ({
            projectId,
            currency:
              projectId === berlin.id ? berlin.currency : general.currency,
            ...totals,
          })
        ),
      })
    }

    if (url.pathname === "/api/expenses") {
      return jsonResponse({
        items: options.recent ?? [],
        nextOffset: null,
      })
    }

    return jsonResponse({ error: "not found" }, 404)
  }
}

function stubApi(options: ApiOptions = {}) {
  const fetchMock = vi.fn<typeof fetch>(apiHandler(options))
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function expectedSummary(
  totals: Totals,
  projectCount: number,
  currency: string
): string {
  const entryLabel = totals.totalCount === 1 ? "entry" : "entries"
  const projectLabel = projectCount === 1 ? "project" : "projects"
  return `${totals.totalCount} ${entryLabel} across ${projectCount} ${projectLabel}. Net ${formatMoney(
    totals.netTotal,
    currency
  )}. Credit ${formatMoney(
    totals.totalCredit,
    currency
  )}. Debit ${formatMoney(totals.totalDebit, currency)}.`
}

function renderDashboard(initialSession: Session | null = session) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={["/orgs/org-1/dashboard"]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route
              path="/orgs/:orgId/dashboard"
              element={<OrgDashboardPage />}
            />
          </Route>
          <Route path="/login" element={<div>Login destination</div>} />
        </Routes>
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("OrgDashboardPage", () => {
  it("redirects to login without a session", () => {
    renderDashboard(null)

    expect(screen.getByText("Login destination")).toBeInTheDocument()
  })

  it("loads totals, project flow, and recent activity", async () => {
    const fetchMock = stubApi({
      projects: [general],
      totals: { [general.id]: usdTotals },
      recent: [clientPayment],
    })

    renderDashboard()

    expect(
      await screen.findByRole("heading", { name: "Dashboard" })
    ).toBeInTheDocument()
    expect(
      await screen.findByText(expectedSummary(usdTotals, 1, "USD"), {
        selector: "p.sr-only",
      })
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Net flow and recent activity for Acme/)
    ).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /General/ })).toHaveAttribute(
      "href",
      "/orgs/org-1/projects/project-1"
    )
    expect(screen.getByText("Client payment")).toBeInTheDocument()
    expect(screen.getByText(/5,000\.00/)).toBeInTheDocument()

    const orgCall = fetchMock.mock.calls.find(([url]) =>
      String(url).startsWith("/api/orgs/org-1?")
    )
    expect(orgCall).toBeDefined()

    const summaryRequest = fetchMock.mock.calls
      .map(([url]) => new URL(String(url), "http://localhost"))
      .find((url) => url.pathname === "/api/expenses/summary")
    expect(summaryRequest).toBeDefined()

    const recentRequest = fetchMock.mock.calls
      .map(([url]) => new URL(String(url), "http://localhost"))
      .find((url) => url.pathname === "/api/expenses")
    expect(recentRequest?.searchParams.get("limit")).toBe("6")

    expect(
      fetchMock.mock.calls.some(([url]) =>
        String(url).startsWith("/api/expenses?projectId=")
      )
    ).toBe(false)
  })

  it("groups mixed currencies and switches the active currency", async () => {
    stubApi({
      projects: [general, berlin],
      totals: {
        [general.id]: usdTotals,
        [berlin.id]: eurTotals,
      },
    })

    renderDashboard()

    expect(
      await screen.findByText(expectedSummary(usdTotals, 1, "USD"), {
        selector: "p.sr-only",
      })
    ).toBeInTheDocument()
    expect(screen.queryByText("Berlin")).not.toBeInTheDocument()

    const eurButton = screen.getByRole("button", { name: "EUR" })
    expect(eurButton).toHaveAttribute("aria-pressed", "false")

    await userEvent.setup().click(eurButton)

    expect(
      await screen.findByText(expectedSummary(eurTotals, 1, "EUR"), {
        selector: "p.sr-only",
      })
    ).toBeInTheDocument()
    expect(screen.getByText("Berlin")).toBeInTheDocument()
    expect(eurButton).toHaveAttribute("aria-pressed", "true")
  })

  it("keeps the large net styling when the total is negative", async () => {
    stubApi({
      projects: [general],
      totals: {
        [general.id]: {
          totalCount: 1,
          totalDebit: 10000,
          totalCredit: 0,
          netTotal: -10000,
        },
      },
    })

    renderDashboard()

    expect(
      await screen.findByText(
        expectedSummary(
          {
            totalCount: 1,
            totalDebit: 10000,
            totalCredit: 0,
            netTotal: -10000,
          },
          1,
          "USD"
        ),
        { selector: "p.sr-only" }
      )
    ).toBeInTheDocument()
    const metric = document.querySelector(".text-metric")
    expect(metric).not.toBeNull()
    expect(metric?.parentElement).toHaveClass("text-primary")
  })

  it("shows an empty project state", async () => {
    stubApi({ projects: [] })

    renderDashboard()

    expect(await screen.findByText("Nothing to summarize")).toBeInTheDocument()
    expect(
      screen.getByRole("link", { name: "Create a project" })
    ).toHaveAttribute("href", "/orgs/org-1")
  })

  it("shows an organization-not-found state", async () => {
    stubApi({
      orgResponse: jsonResponse({ error: "org not found" }, 404),
    })

    renderDashboard()

    expect(
      await screen.findByText("Organization not found.")
    ).toBeInTheDocument()
  })

  it("shows a load error and retries", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockImplementation(apiHandler({ projects: [] }))
    vi.stubGlobal("fetch", fetchMock)

    renderDashboard()

    expect(
      await screen.findByText(
        "Could not reach the server. Check your connection and try again."
      )
    ).toBeInTheDocument()

    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Try again" }))

    expect(await screen.findByText("Nothing to summarize")).toBeInTheDocument()
  })

  it("clears a stale session and redirects to login", async () => {
    stubApi({
      orgResponse: jsonResponse({ error: "user not found" }, 404),
    })

    renderDashboard()

    expect(await screen.findByText("Login destination")).toBeInTheDocument()
  })
})
