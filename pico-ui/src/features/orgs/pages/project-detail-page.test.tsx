import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { Session } from "@/features/auth/session"
import { RequireSession } from "@/features/auth/auth-provider"
import type { Expense } from "@/features/expenses/types"
import type { Org } from "@/features/orgs/types"
import { ProjectDetailPage } from "@/features/orgs/pages/project-detail-page"
import type { Project } from "@/features/projects/types"
import { AuthTestProvider } from "@/test/auth"

const session: Session = {
  userId: "user-1",
  email: "ada@example.com",
}

const org: Org = {
  id: "org-1",
  name: "Acme",
  userId: "user-1",
  tags: [],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const project: Project = {
  id: "project-1",
  orgId: "org-1",
  userId: "user-1",
  name: "General",
  currency: "USD",
  tags: ["travel"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const lunch: Expense = {
  id: "expense-1",
  orgId: "org-1",
  userId: "user-1",
  projectId: project.id,
  type: "debit",
  amount: 1299,
  currency: "USD",
  description: "Lunch",
  tags: ["food"],
  spentAt: "2026-01-02T12:30:00Z",
  createdAt: "2026-01-02T12:30:00Z",
  updatedAt: "2026-01-02T12:30:00Z",
}

function makeExpense(overrides: Partial<Expense> = {}): Expense {
  return {
    ...lunch,
    id: "expense",
    description: "Expense",
    amount: 100,
    ...overrides,
  }
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
  projectResponse?: Response
  expenses?: Expense[]
  listExpensesResponse?: Response
}

function stubApi(options: ApiOptions = {}) {
  const expenses = [...(options.expenses ?? [])]

  function filterExpenses(url: URL): Expense[] {
    const tags = (url.searchParams.get("tags") ?? "")
      .split(",")
      .filter((tag) => tag !== "")
    const from = url.searchParams.get("from")
    const to = url.searchParams.get("to")

    return expenses
      .filter((expense) => tags.every((tag) => expense.tags.includes(tag)))
      .filter((expense) => !from || expense.spentAt >= from)
      .filter((expense) => !to || expense.spentAt <= to)
      .sort((left, right) => right.spentAt.localeCompare(left.spentAt))
  }

  const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
    const url = new URL(String(input), "http://localhost")
    const method = init?.method ?? "GET"

    if (method === "GET" && url.pathname === "/api/orgs/org-1") {
      return options.orgResponse ?? jsonResponse(org)
    }
    if (method === "GET" && url.pathname === "/api/projects/project-1") {
      return options.projectResponse ?? jsonResponse(project)
    }
    if (method === "GET" && url.pathname === "/api/expenses/summary") {
      const filtered = filterExpenses(url)
      const totalDebit = filtered
        .filter((expense) => expense.type !== "credit")
        .reduce((total, expense) => total + expense.amount, 0)
      const totalCredit = filtered
        .filter((expense) => expense.type === "credit")
        .reduce((total, expense) => total + expense.amount, 0)

      return jsonResponse({
        items:
          filtered.length === 0
            ? []
            : [
                {
                  projectId: project.id,
                  currency: project.currency,
                  totalCount: filtered.length,
                  totalDebit,
                  totalCredit,
                  netTotal: totalCredit - totalDebit,
                },
              ],
      })
    }
    if (method === "GET" && url.pathname === "/api/expenses") {
      if (options.listExpensesResponse) {
        return options.listExpensesResponse
      }

      const limit = Number(url.searchParams.get("limit") ?? "20")
      const offset = Number(url.searchParams.get("offset") ?? "0")
      const filtered = filterExpenses(url)
      const page = filtered.slice(offset, offset + limit)
      const nextOffset =
        offset + limit < filtered.length ? offset + limit : null

      return jsonResponse({
        items: page,
        nextOffset,
      })
    }
    if (method === "POST" && url.pathname === "/api/expenses") {
      const body = JSON.parse(String(init?.body)) as Record<string, unknown>
      const created = makeExpense({
        id: `expense-created-${expenses.length + 1}`,
        projectId: String(body.projectId),
        type: body.type === "credit" ? "credit" : "debit",
        amount: Number(body.amount),
        description: String(body.description),
        tags: Array.isArray(body.tags)
          ? body.tags.filter((tag): tag is string => typeof tag === "string")
          : [],
        spentAt: String(body.spentAt),
        createdAt: String(body.spentAt),
        updatedAt: String(body.spentAt),
      })
      expenses.push(created)
      return jsonResponse(created, 201)
    }
    if (method === "DELETE" && url.pathname.startsWith("/api/expenses/")) {
      const id = url.pathname.split("/").pop() ?? ""
      const index = expenses.findIndex((expense) => expense.id === id)
      if (index >= 0) {
        expenses.splice(index, 1)
      }
      return jsonResponse(undefined, 204)
    }

    return jsonResponse({ error: "not found" }, 404)
  })

  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function expenseListURLs(fetchMock: ReturnType<typeof stubApi>): URL[] {
  return fetchMock.mock.calls
    .filter(
      ([url, init]) =>
        String(url).startsWith("/api/expenses?") &&
        (init?.method ?? "GET") === "GET"
    )
    .map(([url]) => new URL(String(url), "http://localhost"))
}

function expenseSummaryURLs(fetchMock: ReturnType<typeof stubApi>): URL[] {
  return fetchMock.mock.calls
    .filter(
      ([url, init]) =>
        String(url).startsWith("/api/expenses/summary?") &&
        (init?.method ?? "GET") === "GET"
    )
    .map(([url]) => new URL(String(url), "http://localhost"))
}

function renderProjectDetail(initialSession: Session | null = session) {
  return render(
    <AuthTestProvider session={initialSession}>
      <MemoryRouter initialEntries={["/orgs/org-1/projects/project-1"]}>
        <Routes>
          <Route element={<RequireSession />}>
            <Route
              path="/orgs/:orgId/projects/:projectId"
              element={<ProjectDetailPage />}
            />
          </Route>
          <Route
            path="/orgs/:orgId"
            element={<div>Projects destination</div>}
          />
          <Route path="/login" element={<div>Login destination</div>} />
        </Routes>
      </MemoryRouter>
    </AuthTestProvider>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("ProjectDetailPage", () => {
  it("redirects to login without a session", () => {
    renderProjectDetail(null)

    expect(screen.getByText("Login destination")).toBeInTheDocument()
  })

  it("loads the project, expenses, and filtered total", async () => {
    const fetchMock = stubApi({ expenses: [lunch] })

    renderProjectDetail()

    expect(
      await screen.findByRole("heading", { name: "General" })
    ).toBeInTheDocument()
    expect(await screen.findByText("Lunch")).toBeInTheDocument()
    expect(screen.getByText("Expenses in Acme")).toBeInTheDocument()
    expect(screen.getByText("travel")).toBeInTheDocument()
    expect(screen.getByText(/1 expense · .*12\.99/)).toBeInTheDocument()

    const projectCall = fetchMock.mock.calls.find(([url]) =>
      String(url).startsWith("/api/projects/project-1?")
    )
    expect(projectCall).toBeDefined()
    const projectURL = new URL(String(projectCall?.[0]), "http://localhost")
    expect(projectURL.searchParams.get("orgId")).toBe("org-1")
    expect(projectURL.searchParams.get("userId")).toBe("user-1")

    const expenseURL = expenseListURLs(fetchMock)[0]
    expect(expenseURL.searchParams.get("projectId")).toBe("project-1")
    expect(expenseURL.searchParams.get("limit")).toBe("20")
    expect(expenseURL.searchParams.get("offset")).toBe("0")
  })

  it("shows debit and credit badges with signed amounts and a net total", async () => {
    stubApi({
      expenses: [
        lunch,
        makeExpense({
          id: "expense-credit",
          type: "credit",
          description: "Client payment",
          amount: 2000,
          spentAt: "2026-01-03T12:30:00Z",
        }),
      ],
    })

    renderProjectDetail()

    expect(await screen.findByText("Lunch")).toBeInTheDocument()
    expect(screen.getByText("Client payment")).toBeInTheDocument()
    expect(screen.getByText(/2 expenses · .*7\.01 net/)).toBeInTheDocument()

    const debitRow = screen.getByText("Lunch").closest("li")
    expect(debitRow).not.toBeNull()
    expect(
      within(debitRow as HTMLElement).getByText("Debit")
    ).toBeInTheDocument()
    expect(
      within(debitRow as HTMLElement).getByText(/12\.99/)
    ).toBeInTheDocument()

    const creditRow = screen.getByText("Client payment").closest("li")
    expect(creditRow).not.toBeNull()
    expect(
      within(creditRow as HTMLElement).getByText("Credit")
    ).toBeInTheDocument()
    expect(
      within(creditRow as HTMLElement).getByText(/\+.*20\.00/)
    ).toBeInTheDocument()
  })

  it("keeps the large net styling when the total is negative", async () => {
    stubApi({ expenses: [lunch] })

    renderProjectDetail()

    expect(await screen.findByText("Lunch")).toBeInTheDocument()
    const metric = document.querySelector(".text-metric-fluid")
    expect(metric).not.toBeNull()
    expect(metric?.parentElement).toHaveClass("text-destructive")
  })

  it("shows the empty state and zero total when the project has no expenses", async () => {
    stubApi({ expenses: [] })

    renderProjectDetail()

    expect(await screen.findByText("No expenses yet")).toBeInTheDocument()
    expect(
      screen.getByText("Use New expense to add the first debit or credit.")
    ).toBeInTheDocument()
    expect(screen.getByText(/0 expenses · .*0\.00/)).toBeInTheDocument()
  })

  it("shows the create expense form only in a modal", async () => {
    stubApi({ expenses: [] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("No expenses yet")

    expect(screen.queryByLabelText("Amount")).not.toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "New expense" })
    )

    expect(
      await screen.findByRole("dialog", { name: "Add expense" })
    ).toBeInTheDocument()
    expect(screen.getByLabelText("Amount")).toBeInTheDocument()
    expect(screen.getByRole("radio", { name: /Debit/ })).toBeChecked()
    expect(screen.getByRole("radio", { name: /Credit/ })).not.toBeChecked()

    await userEventInstance.click(
      screen.getByRole("button", { name: "Cancel" })
    )

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    })
    expect(screen.queryByLabelText("Amount")).not.toBeInTheDocument()
  })

  it("turns filter tags into removable badges", async () => {
    stubApi({ expenses: [lunch] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("Lunch")

    const filterForm = screen.getByRole("form", { name: "Filter expenses" })
    const tagsInput = within(filterForm).getByLabelText("Filter tags (all)")

    await userEventInstance.type(tagsInput, "food,")

    expect(within(filterForm).getByText("food")).toBeInTheDocument()
    expect(tagsInput).toHaveValue("")

    await userEventInstance.click(
      within(filterForm).getByRole("button", { name: "Remove tag food" })
    )

    expect(within(filterForm).queryByText("food")).not.toBeInTheDocument()
  })

  it("creates an expense and reloads the current page total", async () => {
    const fetchMock = stubApi({ expenses: [] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("No expenses yet")

    await userEventInstance.click(
      screen.getByRole("button", { name: "New expense" })
    )
    const dialog = await screen.findByRole("dialog", { name: "Add expense" })
    await userEventInstance.type(screen.getByLabelText("Amount"), "12.99")
    await userEventInstance.type(
      screen.getByLabelText("Description"),
      "Team lunch"
    )
    await userEventInstance.type(within(dialog).getByLabelText("Tags"), "food,")
    expect(within(dialog).getByText("food")).toBeInTheDocument()
    await userEventInstance.click(
      screen.getByRole("button", { name: "Add expense" })
    )

    expect(await screen.findByText("Team lunch")).toBeInTheDocument()
    expect(screen.getByText(/1 expense · .*12\.99/)).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    })

    const createExpenseCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).startsWith("/api/expenses?") && init?.method === "POST"
    )
    expect(createExpenseCall).toBeDefined()
    expect(JSON.parse(String(createExpenseCall?.[1]?.body))).toMatchObject({
      projectId: project.id,
      type: "debit",
      amount: 1299,
      description: "Team lunch",
      tags: ["food"],
    })

    const listCalls = expenseListURLs(fetchMock)
    expect(listCalls.length).toBeGreaterThanOrEqual(2)
    expect(listCalls.at(-1)?.searchParams.get("offset")).toBe("0")
  })

  it("creates a credit and shows money in", async () => {
    const fetchMock = stubApi({ expenses: [] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("No expenses yet")

    await userEventInstance.click(
      screen.getByRole("button", { name: "New expense" })
    )
    const dialog = await screen.findByRole("dialog", { name: "Add expense" })
    await userEventInstance.click(
      within(dialog).getByRole("radio", { name: /Credit/ })
    )
    expect(within(dialog).getByRole("radio", { name: /Credit/ })).toBeChecked()
    await userEventInstance.type(screen.getByLabelText("Amount"), "20.00")
    await userEventInstance.type(
      screen.getByLabelText("Description"),
      "Client payment"
    )
    await userEventInstance.click(
      screen.getByRole("button", { name: "Add expense" })
    )

    expect(await screen.findByText("Client payment")).toBeInTheDocument()
    expect(screen.getByText(/1 expense · .*20\.00 net/)).toBeInTheDocument()

    const creditRow = screen.getByText("Client payment").closest("li")
    expect(creditRow).not.toBeNull()
    expect(
      within(creditRow as HTMLElement).getByText("Credit")
    ).toBeInTheDocument()
    expect(
      within(creditRow as HTMLElement).getByText(/\+.*20\.00/)
    ).toBeInTheDocument()

    const createExpenseCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).startsWith("/api/expenses?") && init?.method === "POST"
    )
    expect(createExpenseCall).toBeDefined()
    expect(JSON.parse(String(createExpenseCall?.[1]?.body))).toMatchObject({
      projectId: project.id,
      type: "credit",
      amount: 2000,
      description: "Client payment",
    })
  })

  it("deletes an expense after confirmation and refreshes the total", async () => {
    const fetchMock = stubApi({ expenses: [lunch] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("Lunch")
    expect(screen.getByText(/1 expense · .*12\.99/)).toBeInTheDocument()

    await userEventInstance.click(
      screen.getByRole("button", { name: "Delete Lunch" })
    )
    await userEventInstance.click(
      screen.getByRole("button", { name: "Delete expense" })
    )

    expect(await screen.findByText("No expenses yet")).toBeInTheDocument()
    expect(await screen.findByText(/0 expenses · .*0\.00/)).toBeInTheDocument()

    const deleteCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        String(url).startsWith("/api/expenses/expense-1?") &&
        init?.method === "DELETE"
    )
    expect(deleteCall).toBeDefined()
  })

  it("paginates without changing the filtered total", async () => {
    const expenses = Array.from({ length: 21 }, (_, index) =>
      makeExpense({
        id: `expense-${index + 1}`,
        description: `Expense ${index + 1}`,
        spentAt: `2026-01-${String(index + 1).padStart(2, "0")}T12:00:00Z`,
      })
    )
    const fetchMock = stubApi({ expenses })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()

    expect(await screen.findByText("Expense 21")).toBeInTheDocument()
    expect(screen.getByText(/21 expenses · .*21\.00/)).toBeInTheDocument()
    expect(screen.getByText("Showing 1–20 of 21")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled()
    expect(expenseSummaryURLs(fetchMock)).toHaveLength(1)

    await userEventInstance.click(screen.getByRole("button", { name: "Next" }))

    expect(await screen.findByText("Expense 1")).toBeInTheDocument()
    expect(screen.getByText(/21 expenses · .*21\.00/)).toBeInTheDocument()
    expect(screen.getByText("Showing 21–21 of 21")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled()
    expect(expenseSummaryURLs(fetchMock)).toHaveLength(1)

    expect(
      expenseListURLs(fetchMock).map((url) => url.searchParams.get("offset"))
    ).toEqual(["0", "20"])
  })

  it("applies tags and dates and resets pagination to the first page", async () => {
    const today = new Date()
    const daysAgo = (days: number) =>
      new Date(
        today.getFullYear(),
        today.getMonth(),
        today.getDate() - days,
        12
      ).toISOString()

    const genericExpenses = Array.from({ length: 25 }, (_, index) =>
      makeExpense({
        id: `expense-${index + 1}`,
        description: `Expense ${index + 1}`,
        spentAt: daysAgo(25 - index),
      })
    )
    const filteredExpense = makeExpense({
      id: "expense-filtered",
      description: "Filtered dinner",
      amount: 2500,
      tags: ["food", "travel"],
      spentAt: daysAgo(0),
    })
    const fetchMock = stubApi({
      expenses: [...genericExpenses, filteredExpense],
    })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("Expense 25")
    await userEventInstance.click(screen.getByRole("button", { name: "Next" }))
    await screen.findByText("Expense 5")

    await userEventInstance.type(
      screen.getByLabelText("Filter tags (all)"),
      "food, travel"
    )
    await userEventInstance.click(
      screen.getByRole("button", { name: "Date range: Any date" })
    )
    await userEventInstance.click(screen.getByRole("button", { name: "Today" }))
    await userEventInstance.click(screen.getByRole("button", { name: "Apply" }))

    expect(await screen.findByText("Filtered dinner")).toBeInTheDocument()
    expect(screen.getByText(/1 expense · .*25\.00/)).toBeInTheDocument()
    expect(screen.queryByText("Expense 25")).not.toBeInTheDocument()

    const lastExpenseURL = expenseListURLs(fetchMock).at(-1)
    expect(lastExpenseURL?.searchParams.get("offset")).toBe("0")
    expect(lastExpenseURL?.searchParams.get("tags")).toBe("food,travel")
    expect(lastExpenseURL?.searchParams.get("from")).toBeTruthy()
    expect(lastExpenseURL?.searchParams.get("to")).toBeTruthy()
  })

  it("clears filters and shows all expenses again", async () => {
    const fetchMock = stubApi({ expenses: [lunch] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("Lunch")

    await userEventInstance.type(
      screen.getByLabelText("Filter tags (all)"),
      "travel"
    )
    await userEventInstance.click(screen.getByRole("button", { name: "Apply" }))

    expect(await screen.findByText("No matching expenses")).toBeInTheDocument()
    expect(screen.getByText(/0 expenses · .*0\.00/)).toBeInTheDocument()

    await userEventInstance.click(screen.getByRole("button", { name: "Clear" }))

    expect(await screen.findByText("Lunch")).toBeInTheDocument()
    expect(screen.getByText(/1 expense · .*12\.99/)).toBeInTheDocument()

    const unfilteredURL = expenseListURLs(fetchMock).find(
      (url) =>
        url.searchParams.get("offset") === "0" &&
        url.searchParams.get("tags") === null &&
        url.searchParams.get("from") === null &&
        url.searchParams.get("to") === null
    )
    expect(unfilteredURL).toBeDefined()
  })

  it("normalizes a custom range selected end-first", async () => {
    const fetchMock = stubApi({ expenses: [lunch] })
    const userEventInstance = userEvent.setup()

    renderProjectDetail()
    await screen.findByText("Lunch")
    const requestCount = expenseListURLs(fetchMock).length

    const now = new Date()
    const dayKey = (day: number) =>
      `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`

    await userEventInstance.click(
      screen.getByRole("button", { name: "Date range: Any date" })
    )

    const later = document.querySelector<HTMLButtonElement>(
      `[data-day="${dayKey(20)}"] button`
    )
    expect(later).not.toBeNull()
    await userEventInstance.click(later as HTMLButtonElement)

    const earlier = document.querySelector<HTMLButtonElement>(
      `[data-day="${dayKey(10)}"] button`
    )
    expect(earlier).not.toBeNull()
    await userEventInstance.click(earlier as HTMLButtonElement)

    await userEventInstance.click(screen.getByRole("button", { name: "Apply" }))

    await waitFor(() => {
      expect(expenseListURLs(fetchMock).length).toBe(requestCount + 1)
    })
    const lastExpenseURL = expenseListURLs(fetchMock).at(-1)
    const from = lastExpenseURL?.searchParams.get("from")
    const to = lastExpenseURL?.searchParams.get("to")
    expect(from).toBeTruthy()
    expect(to).toBeTruthy()
    expect(new Date(from as string).getTime()).toBeLessThan(
      new Date(to as string).getTime()
    )
    expect(
      screen.queryByText("From date must be before the to date.")
    ).not.toBeInTheDocument()
  })

  it("shows a retry action when expenses fail to load", async () => {
    stubApi({
      listExpensesResponse: jsonResponse({ error: "list failed" }, 500),
    })

    renderProjectDetail()

    expect(
      await screen.findByText("Something went wrong. Please try again.")
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Try again" })
    ).toBeInTheDocument()
  })

  it("shows project-not-found with a back action", async () => {
    stubApi({
      projectResponse: jsonResponse({ error: "project not found" }, 404),
    })

    renderProjectDetail()

    expect(await screen.findByText("Project not found.")).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Back to projects" })
    ).toBeInTheDocument()
  })

  it("clears a stale user session", async () => {
    stubApi({
      orgResponse: jsonResponse({ error: "user not found" }, 404),
    })

    renderProjectDetail()

    await waitFor(() => {
      expect(screen.getByText("Login destination")).toBeInTheDocument()
    })
  })
})
