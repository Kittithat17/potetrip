import { useQuery } from '@tanstack/react-query'
import { getFx } from '../api/endpoints'
import { exponent, formatMoney } from './money'

/** The currency people think in. Trips in another currency show estimates in it. */
export const HOME_CURRENCY = 'THB'

/** Formats a home-currency estimate: 2 decimals for small amounts, whole units above 1,000. */
export function formatApprox(value: number, home = HOME_CURRENCY): string {
  const exp = exponent(home)
  const digits = Math.abs(value) >= 1000 ? 0 : exp
  // Round the magnitude so +x and -x always show the same number.
  const minor = Math.sign(value) * Math.round(Math.abs(value) * 10 ** digits) * 10 ** (exp - digits)
  return `≈ ${formatMoney(minor, home)}`.replace(/\.00$/, '')
}

/**
 * Converts minor units of `currency` into an approximate home-currency label
 * using the latest reference rate. Returns null when no estimate is needed
 * (already home currency) or the rate is unavailable.
 */
export function useHomeApprox(currency: string) {
  const enabled = currency !== HOME_CURRENCY
  const q = useQuery({
    queryKey: ['fx', currency, HOME_CURRENCY], // shared with the converter page
    queryFn: () => getFx(currency, HOME_CURRENCY),
    enabled,
    staleTime: 10 * 60_000,
    retry: false,
  })
  const rate = enabled && q.data ? Number(q.data.rate) : null
  const approx = (minor: number): string | null =>
    rate === null || !Number.isFinite(rate) ? null : formatApprox((minor / 10 ** exponent(currency)) * rate)
  return { approx, rate, rateDate: q.data?.rate_date ?? null }
}
