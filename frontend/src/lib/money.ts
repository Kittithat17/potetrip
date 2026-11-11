// Amounts are integers in a currency's minor unit (ISO 4217), never floats.
// Keep this table in sync with backend/internal/money/currency.go.
export const CURRENCY_EXPONENTS: Record<string, number> = {
  THB: 2, USD: 2, EUR: 2, GBP: 2, JPY: 0, KRW: 0, CNY: 2, HKD: 2, TWD: 2, SGD: 2,
  MYR: 2, IDR: 2, PHP: 2, VND: 0, LAK: 2, KHR: 2, MMK: 2, INR: 2, AUD: 2, NZD: 2,
  CAD: 2, CHF: 2, SEK: 2, NOK: 2, DKK: 2, AED: 2, TRY: 2, MOP: 2, CZK: 2, HUF: 2,
  PLN: 2, ISK: 0,
}

export const CURRENCIES = Object.keys(CURRENCY_EXPONENTS)

const SYMBOLS: Record<string, string> = {
  THB: '฿', USD: '$', EUR: '€', GBP: '£', JPY: '¥', KRW: '₩', CNY: '¥', VND: '₫', INR: '₹',
}

export function exponent(currency: string): number {
  return CURRENCY_EXPONENTS[currency] ?? 2
}

/**
 * Parses user input like "1,250.5" into minor units (125050 for THB).
 * Returns null for empty, negative, malformed, or over-precise input.
 */
export function parseAmount(input: string, currency: string): number | null {
  const s = input.replace(/[,\s]/g, '')
  const exp = exponent(currency)
  const m = /^(\d+)(?:\.(\d*))?$/.exec(s)
  if (!m) return null
  const [, whole, frac = ''] = m
  if (frac.length > exp) return null
  const minor = Number(whole + frac.padEnd(exp, '0'))
  return Number.isSafeInteger(minor) ? minor : null
}

/** Formats minor units as a plain decimal string, e.g. 125050 -> "1,250.50". */
export function formatAmount(minor: number, currency: string, grouping = true): string {
  const exp = exponent(currency)
  const neg = minor < 0
  const digits = String(Math.abs(minor)).padStart(exp + 1, '0')
  const whole = digits.slice(0, digits.length - exp)
  const frac = exp > 0 ? digits.slice(-exp) : ''
  const grouped = grouping ? whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : whole
  return (neg ? '-' : '') + grouped + (frac ? '.' + frac : '')
}

/** Formats with a currency symbol: "฿1,250.50", or "1,250.50 MYR" when no symbol is known. */
export function formatMoney(minor: number, currency: string): string {
  const sym = SYMBOLS[currency]
  const neg = minor < 0
  const body = formatAmount(Math.abs(minor), currency)
  const s = sym ? `${sym}${body}` : `${body} ${currency}`
  return neg ? `-${s}` : s
}

/** Equal split; earlier people absorb the remainder (mirrors the backend). */
export function splitEqual(total: number, n: number): number[] {
  if (n <= 0) return []
  const base = Math.floor(total / n)
  const rem = total - base * n
  return Array.from({ length: n }, (_, i) => base + (i < rem ? 1 : 0))
}
