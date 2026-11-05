import { describe, expect, it } from "vitest"

import {
  flowWidth,
  maxFlowValue,
  summarizeCurrencies,
  type ProjectTotals,
} from "@/features/dashboard/data"
import type { ExpensePage } from "@/features/expenses/types"
import type { Project } from "@/features/projects/types"

function makeProject(overrides: Partial<Project> = {}): Project {
  return {
    id: "project-1",
    orgId: "org-1",
    userId: "user-1",
    name: "General",
    currency: "USD",
    tags: [],
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

function makeTotals(overrides: Partial<ExpensePage> = {}): ExpensePage {
  return {
    items: [],
    totalCount: 0,
    totalDebit: 0,
    totalCredit: 0,
    netTotal: 0,
    ...overrides,
  }
}

function makeEntry(
  project: Partial<Project>,
  totals: Partial<ExpensePage>
): ProjectTotals {
  return { project: makeProject(project), totals: makeTotals(totals) }
}

describe("summarizeCurrencies", () => {
  it("groups projects by currency and sums their totals", () => {
    const summaries = summarizeCurrencies([
      makeEntry(
        { id: "project-1", name: "General", currency: "USD" },
        { totalCount: 2, totalDebit: 5000, totalCredit: 15000 }
      ),
      makeEntry(
        { id: "project-2", name: "Travel", currency: "USD" },
        { totalCount: 1, totalDebit: 2000, totalCredit: 9000 }
      ),
      makeEntry(
        { id: "project-3", name: "Berlin", currency: "EUR" },
        { totalCount: 3, totalDebit: 1000, totalCredit: 4000 }
      ),
    ])

    expect(summaries).toHaveLength(2)
    const usd = summaries.find((summary) => summary.currency === "USD")
    expect(usd).toMatchObject({
      totalCount: 3,
      totalDebit: 7000,
      totalCredit: 24000,
      netTotal: 17000,
    })
    expect(usd?.projects).toHaveLength(2)

    const eur = summaries.find((summary) => summary.currency === "EUR")
    expect(eur).toMatchObject({
      totalCount: 3,
      totalDebit: 1000,
      totalCredit: 4000,
      netTotal: 3000,
    })
  })

  it("never adds minor units across currencies", () => {
    const summaries = summarizeCurrencies([
      makeEntry({ currency: "USD" }, { totalCredit: 10000 }),
      makeEntry({ id: "project-2", currency: "EUR" }, { totalCredit: 5000 }),
    ])

    expect(summaries.map((summary) => summary.netTotal)).toEqual([10000, 5000])
  })

  it("orders projects inside a currency by net total", () => {
    const summaries = summarizeCurrencies([
      makeEntry(
        { id: "project-1", name: "Low", currency: "USD" },
        { totalCredit: 1000 }
      ),
      makeEntry(
        { id: "project-2", name: "High", currency: "USD" },
        { totalCredit: 9000 }
      ),
      makeEntry(
        { id: "project-3", name: "Negative", currency: "USD" },
        { totalDebit: 5000 }
      ),
    ])

    expect(summaries[0].projects.map((entry) => entry.project.name)).toEqual([
      "High",
      "Low",
      "Negative",
    ])
  })
})

describe("flow helpers", () => {
  it("uses the largest credit or debit as the scale", () => {
    expect(
      maxFlowValue([
        makeEntry({}, { totalCredit: 500, totalDebit: 2000 }),
        makeEntry({ id: "project-2" }, { totalCredit: 1500 }),
      ])
    ).toBe(2000)
  })

  it("keeps positive bars visible and caps them at 100", () => {
    expect(flowWidth(0, 100)).toBe(0)
    expect(flowWidth(1, 1000)).toBe(2)
    expect(flowWidth(500, 1000)).toBe(50)
    expect(flowWidth(2000, 1000)).toBe(100)
  })
})
