export function currencyFractionDigits(currency: string): number {
  try {
    const options = new Intl.NumberFormat("en", {
      style: "currency",
      currency,
    }).resolvedOptions()
    return options.maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

export function toMinorUnits(value: string, currency: string): number | null {
  const normalized = value.trim().replace(/,/g, "")
  if (normalized === "") {
    return null
  }

  const amount = Number(normalized)
  if (!Number.isFinite(amount) || amount <= 0) {
    return null
  }

  const minor = Math.round(amount * 10 ** currencyFractionDigits(currency))
  if (!Number.isSafeInteger(minor) || minor <= 0) {
    return null
  }
  return minor
}

export function formatMoney(minor: number, currency: string): string {
  const digits = currencyFractionDigits(currency)
  const locale = currency === "INR" ? "en-IN" : undefined
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(minor / 10 ** digits)
}

export function formatCompactMoney(minor: number, currency: string): string {
  const digits = currencyFractionDigits(currency)
  const amount = minor / 10 ** digits

  if (Math.abs(amount) < 1000) {
    return formatMoney(minor, currency)
  }

  const locale = currency === "INR" ? "en-IN" : undefined
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    notation: "compact",
    compactDisplay: "short",
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(amount)
}

export function formatExpenseDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(new Date(value))
}

export function formatExpenseDateShort(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    day: "numeric",
    month: "short",
  }).format(new Date(value))
}

export function todayInputValue(): string {
  const now = new Date()
  const local = new Date(now.getTime() - now.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 10)
}

export function dateToSpentAt(value: string): string | null {
  if (!value) {
    return null
  }
  const date = new Date(`${value}T12:00:00`)
  if (Number.isNaN(date.getTime())) {
    return null
  }
  return date.toISOString()
}
