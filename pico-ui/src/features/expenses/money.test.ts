import { describe, expect, it } from "vitest"

import {
  currencyFractionDigits,
  formatDirectionalMoney,
  formatMoney,
  parseAmountToMinorUnits,
} from "@/features/expenses/money"

describe("currencyFractionDigits", () => {
  it("uses the currency exponent", () => {
    expect(currencyFractionDigits("USD")).toBe(2)
    expect(currencyFractionDigits("JPY")).toBe(0)
    expect(currencyFractionDigits("KWD")).toBe(3)
  })
})

describe("parseAmountToMinorUnits", () => {
  it("parses 2-decimal currencies", () => {
    expect(parseAmountToMinorUnits("12.99", "USD")).toBe(1299)
    expect(parseAmountToMinorUnits("12.9", "USD")).toBe(1290)
    expect(parseAmountToMinorUnits("0.01", "USD")).toBe(1)
  })

  it("parses 0-decimal currencies", () => {
    expect(parseAmountToMinorUnits("100", "JPY")).toBe(100)
    expect(parseAmountToMinorUnits("100.5", "JPY")).toBeNull()
  })

  it("parses 3-decimal currencies", () => {
    expect(parseAmountToMinorUnits("1.234", "KWD")).toBe(1234)
    expect(parseAmountToMinorUnits("1.23", "KWD")).toBe(1230)
  })

  it("rejects too many decimals and malformed values", () => {
    expect(parseAmountToMinorUnits("12.999", "USD")).toBeNull()
    expect(parseAmountToMinorUnits("12.", "USD")).toBeNull()
    expect(parseAmountToMinorUnits(".99", "USD")).toBeNull()
    expect(parseAmountToMinorUnits("abc", "USD")).toBeNull()
  })
})

describe("formatMoney", () => {
  it("formats minor units using the currency", () => {
    expect(formatMoney(1299, "USD")).toContain("12.99")
    expect(formatMoney(100, "JPY")).toContain("100")
  })

  it("does not format negative zero", () => {
    expect(formatMoney(-0, "USD")).not.toContain("-")
  })
})

describe("formatDirectionalMoney", () => {
  it("prefixes debit and credit amounts with their direction", () => {
    expect(formatDirectionalMoney(1299, "USD", "debit")).toContain("-")
    expect(formatDirectionalMoney(1299, "USD", "debit")).toContain("12.99")
    expect(formatDirectionalMoney(1299, "USD", "credit")).toContain("+")
    expect(formatDirectionalMoney(1299, "USD", "credit")).toContain("12.99")
  })

  it("does not sign a zero amount", () => {
    expect(formatDirectionalMoney(0, "USD", "debit")).not.toContain("-")
    expect(formatDirectionalMoney(0, "USD", "credit")).not.toContain("+")
  })
})
