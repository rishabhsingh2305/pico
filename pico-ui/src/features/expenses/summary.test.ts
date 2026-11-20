import { describe, expect, it } from "vitest"

import { summarizeByCurrency } from "@/features/expenses/summary"

describe("summarizeByCurrency", () => {
  it("groups totals by currency and never mixes currencies", () => {
    const summaries = summarizeByCurrency([
      {
        projectId: "project-1",
        currency: "INR",
        totalCount: 2,
        totalDebit: 3000,
        totalCredit: 1000,
        netTotal: -2000,
      },
      {
        projectId: "project-2",
        currency: "INR",
        totalCount: 1,
        totalDebit: 0,
        totalCredit: 5000,
        netTotal: 5000,
      },
      {
        projectId: "project-3",
        currency: "USD",
        totalCount: 1,
        totalDebit: 1000,
        totalCredit: 0,
        netTotal: -1000,
      },
    ])

    expect(summaries).toHaveLength(2)
    expect(summaries[0]).toMatchObject({
      currency: "INR",
      projectCount: 2,
      totalCount: 3,
      totalDebit: 3000,
      totalCredit: 6000,
      netTotal: 3000,
    })
    expect(summaries[1]).toMatchObject({
      currency: "USD",
      projectCount: 1,
      netTotal: -1000,
    })
  })
})
