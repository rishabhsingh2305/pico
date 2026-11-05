export const CURRENCIES = [
  "USD",
  "EUR",
  "GBP",
  "INR",
  "JPY",
  "KWD",
  "AUD",
  "CAD",
  "CHF",
] as const

export type CurrencyCode = (typeof CURRENCIES)[number]
