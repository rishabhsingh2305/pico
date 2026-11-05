import { describe, expect, it } from "vitest"

import type { ExpenseType } from "@/features/expenses/types"
import {
  nowLocalDateTime,
  validateExpense,
  type ExpenseFormValues,
} from "@/features/expenses/validation"

const validValues: ExpenseFormValues = {
  projectId: "project-1",
  type: "debit",
  amount: "12.99",
  description: "Lunch",
  tags: ["food", "travel"],
  spentAt: "2026-01-02T12:30",
}

describe("validateExpense", () => {
  it("accepts valid input and returns normalized values", () => {
    const result = validateExpense(validValues, "USD")

    expect(result.errors).toEqual({})
    expect(result.type).toBe("debit")
    expect(result.amountMinorUnits).toBe(1299)
    expect(result.tags).toEqual(["food", "travel"])
    expect(result.spentAtIso).toMatch(/Z$/)
  })

  it("returns the credit type", () => {
    const result = validateExpense({ ...validValues, type: "credit" }, "USD")

    expect(result.errors).toEqual({})
    expect(result.type).toBe("credit")
  })

  it("requires a valid type", () => {
    expect(
      validateExpense(
        { ...validValues, type: "transfer" as ExpenseType },
        "USD"
      ).errors.type
    ).toBe("Choose debit or credit.")
  })

  it("requires a project", () => {
    expect(
      validateExpense({ ...validValues, projectId: "" }, "USD").errors.projectId
    ).toBe("Choose a project.")
  })

  it("requires a positive valid amount", () => {
    expect(
      validateExpense({ ...validValues, amount: "" }, "USD").errors.amount
    ).toBe("Enter an amount.")
    expect(
      validateExpense({ ...validValues, amount: "abc" }, "USD").errors.amount
    ).toBe("Enter a valid amount.")
    expect(
      validateExpense({ ...validValues, amount: "0" }, "USD").errors.amount
    ).toBe("Amount must be greater than zero.")
  })

  it("limits the description", () => {
    expect(
      validateExpense(
        {
          ...validValues,
          description: "a".repeat(1001),
        },
        "USD"
      ).errors.description
    ).toBe("Description must be at most 1000 characters.")
  })

  it("limits tags", () => {
    expect(
      validateExpense(
        {
          ...validValues,
          tags: Array.from({ length: 21 }, (_, index) => `tag-${index}`),
        },
        "USD"
      ).errors.tags
    ).toBe("Use at most 20 tags.")
    expect(
      validateExpense(
        {
          ...validValues,
          tags: ["a".repeat(33)],
        },
        "USD"
      ).errors.tags
    ).toBe("Each tag must be at most 32 characters.")
  })

  it("requires a valid spent-at value", () => {
    expect(
      validateExpense({ ...validValues, spentAt: "" }, "USD").errors.spentAt
    ).toBe("Choose a date.")
    expect(
      validateExpense({ ...validValues, spentAt: "not-a-date" }, "USD").errors
        .spentAt
    ).toBe("Choose a valid date and time.")
  })

  it("formats the current local time for datetime-local inputs", () => {
    expect(nowLocalDateTime()).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)
  })
})
