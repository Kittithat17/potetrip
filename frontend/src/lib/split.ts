// Client-side previews of the server's split rules (backend/internal/expense/split.go).
// The server recomputes everything on save; these only show the numbers early.
import { exponent } from './money'

/** Splits total into parts proportional to weights; leftovers go 1 unit at a time in order. */
export function allocate(total: number, weights: number[]): number[] {
  const sum = weights.reduce((s, w) => s + w, 0)
  if (sum <= 0) return weights.map(() => 0)
  const out = weights.map((w) => Math.floor((total * w) / sum))
  let rem = total - out.reduce((s, v) => s + v, 0)
  while (rem > 0) {
    for (let i = 0; i < weights.length && rem > 0; i++) {
      if (weights[i] > 0) {
        out[i]++
        rem--
      }
    }
  }
  return out
}

export type ItemSplit = { amount: number; memberIds: string[] }

/**
 * Spreads total over items by price (so tax or discounts follow), then splits
 * each item equally among its people in trip order. Returns memberId -> amount.
 */
export function splitItems(total: number, items: ItemSplit[], memberOrder: string[]): Map<string, number> {
  const owed = new Map<string, number>()
  const perItem = allocate(
    total,
    items.map((it) => it.amount),
  )
  items.forEach((it, i) => {
    const ids = memberOrder.filter((id) => it.memberIds.includes(id))
    if (ids.length === 0) return
    allocate(
      perItem[i],
      ids.map(() => 1),
    ).forEach((v, j) => owed.set(ids[j], (owed.get(ids[j]) ?? 0) + v))
  })
  return owed
}

/** Converts minor units between currencies at `rate` (to per 1 from), rounding half up. */
export function convertMinor(amount: number, from: string, to: string, rate: number): number {
  return Math.round((amount / 10 ** exponent(from)) * rate * 10 ** exponent(to))
}

/** Accepts a rate like "0.2345" (max 8 decimals); returns null if it isn't a positive number. */
export function parseRate(input: string): number | null {
  const s = input.replace(/,/g, '').trim()
  if (!/^\d*\.?\d{0,8}$/.test(s) || s === '' || s === '.') return null
  const v = Number(s)
  return v > 0 && Number.isFinite(v) ? v : null
}
