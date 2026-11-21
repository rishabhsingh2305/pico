import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { formatMoney } from "@/features/expenses/format"
import type { Expense } from "@/features/expenses/types"
import { SharedPage } from "@/features/sharing/pages/shared-page"
import { clearShareToken } from "@/lib/share-token"

const category = {
  id: "project-1",
  name: "Food",
  currency: "INR",
}

const expense: Expense = {
  id: "expense-1",
  projectId: "project-1",
  type: "debit",
  amount: 1_299,
  currency: "INR",
  description: "Lunch",
  tags: ["food"],
  spentAt: "2026-11-20T10:00:00Z",
}

const secondExpense: Expense = {
  id: "expense-2",
  projectId: "project-1",
  type: "debit",
  amount: 499,
  currency: "INR",
  description: "Coffee",
  tags: [],
  spentAt: "2026-11-21T10:00:00Z",
}

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response
}

function setShareLink(query: string) {
  window.history.replaceState(null, "", `/shared?${query}#share=raw-token`)
}

function stubSharedApi({
  expenses = [expense],
  summary = [],
  sharedExpense = expense,
}: {
  expenses?: Expense[]
  summary?: Array<{
    projectId: string
    currency: string
    totalCount: number
    totalDebit: number
    totalCredit: number
    netTotal: number
  }>
  sharedExpense?: Expense
} = {}) {
  const fetchMock = vi.fn<typeof fetch>(async (input) => {
    const url = new URL(String(input), "http://localhost")

    if (url.pathname === "/api/projects/project-1") {
      return jsonResponse(category)
    }
    if (url.pathname === "/api/expenses") {
      return jsonResponse({ items: expenses, nextOffset: null })
    }
    if (url.pathname === "/api/expenses/summary") {
      return jsonResponse({ items: summary })
    }
    if (url.pathname === "/api/expenses/expense-1") {
      return jsonResponse(sharedExpense)
    }
    return jsonResponse({ error: "not found" }, 404)
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

function stubPaginatedApi() {
  const fetchMock = vi.fn<typeof fetch>(async (input) => {
    const url = new URL(String(input), "http://localhost")

    if (url.pathname === "/api/projects/project-1") {
      return jsonResponse(category)
    }
    if (url.pathname === "/api/expenses/summary") {
      return jsonResponse({
        items: [
          {
            projectId: "project-1",
            currency: "INR",
            totalCount: 2,
            totalDebit: 1_798,
            totalCredit: 0,
            netTotal: -1_798,
          },
        ],
      })
    }
    if (url.pathname === "/api/expenses") {
      const offset = url.searchParams.get("offset")
      if (offset === null || offset === "0") {
        return jsonResponse({ items: [expense], nextOffset: 20 })
      }
      if (offset === "20") {
        return jsonResponse({ items: [secondExpense], nextOffset: null })
      }
    }
    return jsonResponse({ error: "not found" }, 404)
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  clearShareToken()
  window.history.replaceState(null, "", "/")
})

describe("SharedPage", () => {
  it("shows credit and debit totals with a signed negative net", async () => {
    setShareLink(
      "orgId=org-1&resourceType=project&resourceId=project-1&projectId=project-1"
    )
    stubSharedApi({
      summary: [
        {
          projectId: "project-1",
          currency: "INR",
          totalCount: 3,
          totalDebit: 500_000,
          totalCredit: 200_000,
          netTotal: -300_000,
        },
      ],
    })

    render(<SharedPage />)

    expect(await screen.findByText("Food")).toBeTruthy()
    expect(screen.getByText("Net balance")).toBeTruthy()
    expect(screen.getByText("Credit")).toBeTruthy()
    expect(screen.getByText("Debit")).toBeTruthy()
    expect(screen.getByText("Money in")).toBeTruthy()
    expect(screen.getByText("Money out")).toBeTruthy()

    const net = screen.getByText(formatMoney(-300_000, "INR"))
    expect(net.className).toContain("text-spend")
    expect(net.textContent?.startsWith("-")).toBe(true)
    expect(screen.getByText(formatMoney(200_000, "INR"))).toBeTruthy()
    expect(screen.getByText(formatMoney(500_000, "INR"))).toBeTruthy()
  })

  it("combines same-currency rows and shows every currency", async () => {
    setShareLink("orgId=org-1&resourceType=org&resourceId=org-1")
    stubSharedApi({
      summary: [
        {
          projectId: "project-1",
          currency: "INR",
          totalCount: 2,
          totalDebit: 5_000,
          totalCredit: 1_000,
          netTotal: -4_000,
        },
        {
          projectId: "project-2",
          currency: "INR",
          totalCount: 1,
          totalDebit: 1_000,
          totalCredit: 3_000,
          netTotal: 2_000,
        },
        {
          projectId: "project-3",
          currency: "USD",
          totalCount: 1,
          totalDebit: 0,
          totalCredit: 2_500,
          netTotal: 2_500,
        },
        {
          projectId: "project-4",
          currency: "EUR",
          totalCount: 1,
          totalDebit: 750,
          totalCredit: 0,
          netTotal: -750,
        },
      ],
    })

    render(<SharedPage />)

    expect(await screen.findByText("Shared expenses")).toBeTruthy()
    expect(window.location.hash).toBe("#share=raw-token")
    expect(screen.getByText("INR")).toBeTruthy()
    expect(screen.getByText("USD")).toBeTruthy()
    expect(screen.getByText("EUR")).toBeTruthy()
    expect(screen.getByText("3 expenses · 2 categories")).toBeTruthy()
    expect(screen.getAllByText("Net balance")).toHaveLength(3)
  })

  it("derives totals for a single-expense share", async () => {
    setShareLink(
      "orgId=org-1&resourceType=expense&resourceId=expense-1&projectId=project-1"
    )
    stubSharedApi({
      sharedExpense: { ...expense, type: "credit", amount: 250_000 },
    })

    render(<SharedPage />)

    expect(await screen.findByText("Food")).toBeTruthy()
    expect(screen.getByText("Net balance")).toBeTruthy()
    expect(screen.getByText("1 expense")).toBeTruthy()
    expect(screen.getByText("Credit")).toBeTruthy()
    expect(screen.getByText("Debit")).toBeTruthy()
    expect(
      screen.getAllByText(formatMoney(250_000, "INR")).length
    ).toBeGreaterThan(1)
  })

  it("loads the next page when the load more button is pressed", async () => {
    setShareLink(
      "orgId=org-1&resourceType=project&resourceId=project-1&projectId=project-1"
    )
    const fetchMock = stubPaginatedApi()

    render(<SharedPage />)

    await screen.findByText("Lunch")
    fireEvent.click(screen.getByRole("button", { name: "Load more" }))

    expect(await screen.findByText("Coffee")).toBeTruthy()
    expect(screen.getByText("All expenses loaded.")).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull()

    const listURLs = fetchMock.mock.calls
      .map(([input]) => new URL(String(input), "http://localhost"))
      .filter((url) => url.pathname === "/api/expenses")
    expect(listURLs.map((url) => url.searchParams.get("offset"))).toEqual([
      null,
      "20",
    ])

    const summaryCalls = fetchMock.mock.calls.filter(
      ([input]) =>
        new URL(String(input), "http://localhost").pathname ===
        "/api/expenses/summary"
    )
    expect(summaryCalls).toHaveLength(1)
  })

  it("loads the next page automatically when the sentinel intersects", async () => {
    class MockIntersectionObserver {
      static instances: MockIntersectionObserver[] = []
      callback: IntersectionObserverCallback
      root = null
      rootMargin = ""
      thresholds = []

      constructor(callback: IntersectionObserverCallback) {
        this.callback = callback
        MockIntersectionObserver.instances.push(this)
      }

      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return []
      }

      trigger() {
        this.callback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          this as unknown as IntersectionObserver
        )
      }
    }

    vi.stubGlobal("IntersectionObserver", MockIntersectionObserver)
    setShareLink(
      "orgId=org-1&resourceType=project&resourceId=project-1&projectId=project-1"
    )
    stubPaginatedApi()

    render(<SharedPage />)

    await screen.findByText("Lunch")
    await waitFor(() => {
      expect(MockIntersectionObserver.instances.length).toBeGreaterThan(0)
    })

    await act(async () => {
      MockIntersectionObserver.instances[0]?.trigger()
    })

    expect(await screen.findByText("Coffee")).toBeTruthy()
  })
})
