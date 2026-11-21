import { describe, expect, it } from "vitest"

import {
  summarizeByCurrency,
  totalsFromExpense,
} from "@/features/expenses/summary"
import type { Expense } from "@/features/expenses/types"

describe("summarizeByCurrency", () => {
  it("combines projects that use the same currency", () => {
    expect(
      summarizeByCurrency([
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
      ])
    ).toEqual([
      {
        currency: "INR",
        projectCount: 2,
        totalCount: 3,
        totalDebit: 6_000,
        totalCredit: 4_000,
        netTotal: -2_000,
      },
    ])
  })

  it("keeps currencies separate and sorted", () => {
    expect(
      summarizeByCurrency([
        {
          projectId: "project-1",
          currency: "USD",
          totalCount: 1,
          totalDebit: 0,
          totalCredit: 2_500,
          netTotal: 2_500,
        },
        {
          projectId: "project-2",
          currency: "INR",
          totalCount: 1,
          totalDebit: 1_500,
          totalCredit: 0,
          netTotal: -1_500,
        },
      ])
    ).toEqual([
      {
        currency: "INR",
        projectCount: 1,
        totalCount: 1,
        totalDebit: 1_500,
        totalCredit: 0,
        netTotal: -1_500,
      },
      {
        currency: "USD",
        projectCount: 1,
        totalCount: 1,
        totalDebit: 0,
        totalCredit: 2_500,
        netTotal: 2_500,
      },
    ])
  })
})

describe("totalsFromExpense", () => {
  const baseExpense: Expense = {
    id: "expense-1",
    projectId: "project-1",
    type: "debit",
    amount: 1_299,
    currency: "INR",
    description: "Lunch",
    tags: [],
    spentAt: "2026-11-20T10:00:00Z",
  }

  it("records a debit as spend", () => {
    expect(totalsFromExpense(baseExpense)).toEqual({
      currency: "INR",
      projectCount: 1,
      totalCount: 1,
      totalDebit: 1_299,
      totalCredit: 0,
      netTotal: -1_299,
    })
  })

  it("records a credit as received", () => {
    expect(
      totalsFromExpense({ ...baseExpense, type: "credit", amount: 2_500 })
    ).toEqual({
      currency: "INR",
      projectCount: 1,
      totalCount: 1,
      totalDebit: 0,
      totalCredit: 2_500,
      netTotal: 2_500,
    })
  })
})
