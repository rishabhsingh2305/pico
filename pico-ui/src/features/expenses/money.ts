import type { ExpenseType } from "@/features/expenses/types"

export function currencyFractionDigits(currency: string): number {
  try {
    return (
      new Intl.NumberFormat("en", {
        style: "currency",
        currency,
      }).resolvedOptions().maximumFractionDigits ?? 2
    )
  } catch {
    return 2
  }
}

export function parseAmountToMinorUnits(
  raw: string,
  currency: string
): number | null {
  const value = raw.trim()
  const digits = currencyFractionDigits(currency)
  const pattern =
    digits === 0 ? /^\d+$/ : new RegExp(`^\\d+(?:\\.\\d{1,${digits}})?$`)

  if (!pattern.test(value)) {
    return null
  }

  const [whole, fraction = ""] = value.split(".")
  const normalizedWhole = whole.replace(/^0+(?=\d)/, "")
  const paddedFraction = fraction.padEnd(digits, "0")
  const minorUnits = Number(`${normalizedWhole}${paddedFraction}`)

  return Number.isSafeInteger(minorUnits) ? minorUnits : null
}

export function formatMoney(amount: number, currency: string): string {
  const digits = currencyFractionDigits(currency)
  const value = amount === 0 ? 0 : amount
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
    }).format(value / 10 ** digits)
  } catch {
    return `${(value / 10 ** digits).toFixed(digits)} ${currency}`
  }
}

export function formatDirectionalMoney(
  amount: number,
  currency: string,
  type: ExpenseType
): string {
  const formatted = formatMoney(amount, currency)
  if (amount === 0) {
    return formatted
  }
  return `${type === "credit" ? "+" : "-"}${formatted}`
}
