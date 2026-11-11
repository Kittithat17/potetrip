import { expect, it } from 'vitest'
import { mapLimit } from './image'

it('mapLimit never exceeds the limit and visits every item', async () => {
  let inFlight = 0
  let peak = 0
  const seen: number[] = []
  await mapLimit([1, 2, 3, 4, 5, 6, 7], 3, async (n) => {
    inFlight++
    peak = Math.max(peak, inFlight)
    await new Promise((r) => setTimeout(r, 5))
    seen.push(n)
    inFlight--
  })
  expect(peak).toBe(3)
  expect(seen.sort()).toEqual([1, 2, 3, 4, 5, 6, 7])
})
