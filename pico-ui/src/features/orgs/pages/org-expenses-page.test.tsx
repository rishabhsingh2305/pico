import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import { OrgWorkspaceLayout } from "@/features/orgs/layouts/org-workspace-layout"
import { OrgExpensesPage } from "@/features/orgs/pages/org-expenses-page"
import { RequireSession } from "@/features/auth/auth-provider"
import type { Session } from "@/features/auth/session"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

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
  name: "Launch",
  currency: "INR",
  tags: [],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const expense = {
  id: "expense-1",
  orgId: "org-1",
  userId: "user-1",
  projectId: "project-1",
  type: "debit" as const,
  amount: 1299,
  currency: "INR",
  description: "Lunch",
  tags: ["food"],
  spentAt: "2026-11-20T12:30:00Z",
  createdAt: "2026-11-20T12:30:00Z",
  updatedAt: "2026-11-20T12:30:00Z",
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function stubApi() {
  const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
    const url = new URL(String(input), "http://localhost")
    const method = init?.method ?? "GET"

    if (method === "GET" && url.pathname === "/api/orgs/org-1") {
      return jsonResponse(org)
    }
    if (method === "GET" && url.pathname === "/api/projects") {
      return jsonResponse({ items: [project] })
    }
    if (method === "GET" && url.pathname === "/api/expenses/summary") {
      return jsonResponse({
        items: [
          {
            projectId: "project-1",
            currency: "INR",
            totalCount: 1,
            totalDebit: 1299,
            totalCredit: 0,
            netTotal: -1299,
          },
        ],
      })
    }
    if (method === "GET" && url.pathname === "/api/expenses") {
      return jsonResponse({ items: [expense], nextOffset: null })
    }
    return jsonResponse({ error: "not found" }, 404)
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function renderExpenses(initialSession: Session | null = session) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={["/orgs/org-1/expenses"]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route path="/orgs/:orgId" element={<OrgWorkspaceLayout />}>
              <Route path="expenses" element={<OrgExpensesPage />} />
            </Route>
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

describe("OrgExpensesPage", () => {
  it("shows the organization ledger and totals", async () => {
    stubApi()

    renderExpenses()

    expect(
      await screen.findByRole("heading", { name: "Expenses" })
    ).toBeInTheDocument()
    expect(await screen.findByText("Lunch")).toBeInTheDocument()
    expect(screen.getAllByText("Launch").length).toBeGreaterThan(0)
    expect(screen.getByText("food")).toBeInTheDocument()
    expect(
      screen.getByText("1 entry across 1 project", { exact: false })
    ).toBeInTheDocument()
  })

  it("filters the ledger by project", async () => {
    const fetchMock = stubApi()
    const user = userEvent.setup()

    renderExpenses()
    await screen.findByText("Lunch")

    await user.click(screen.getByLabelText("Project"))
    await user.click(await screen.findByRole("option", { name: "Launch" }))
    await user.click(screen.getByRole("button", { name: "Apply" }))

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([url]) =>
          String(url).includes("projectId=project-1")
        )
      ).toBe(true)
    })
  })

  it("deletes an expense", async () => {
    const fetchMock = stubApi()
    const user = userEvent.setup()

    renderExpenses()
    await screen.findByText("Lunch")

    await user.click(screen.getByRole("button", { name: "Delete Lunch" }))
    await screen.findByRole("alertdialog", { name: "Delete expense?" })
    await user.click(screen.getByRole("button", { name: "Delete expense" }))

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(
          ([url, init]) =>
            String(url).startsWith("/api/expenses/expense-1") &&
            init?.method === "DELETE"
        )
      ).toBe(true)
    })
  })
})
