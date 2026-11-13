import { describe, expect, it } from 'vitest'
import { allocate, convertMinor, parseRate, splitItems } from './split'

describe('allocate', () => {
  it('matches the server for remainders', () => {
    expect(allocate(10000, [1, 1, 1])).toEqual([3334, 3333, 3333])
    expect(allocate(100000, [60000, 30000])).toEqual([66667, 33333])
    expect(allocate(5, [0, 1, 1])).toEqual([0, 3, 2])
  })
})

describe('splitItems', () => {
  it('spreads VAT by price and splits each item in trip order', () => {
    const owed = splitItems(
      107000,
      [
        { amount: 60000, memberIds: ['a', 'b'] },
        { amount: 30000, memberIds: ['c'] },
        { amount: 10000, memberIds: ['c', 'b', 'a'] },
      ],
      ['a', 'b', 'c'],
    )
    // Same numbers as the backend test TestItemSplitAndSummary.
    expect(owed.get('a')).toBe(32100 + 3567)
    expect(owed.get('b')).toBe(32100 + 3567)
    expect(owed.get('c')).toBe(32100 + 3566)
  })
})

it('converts between currencies with different exponents', () => {
  expect(convertMinor(1500, 'JPY', 'THB', 0.2345)).toBe(35175)
  expect(convertMinor(12550, 'THB', 'JPY', 4.2645)).toBe(535)
})

it('parses rates', () => {
  expect(parseRate('0.2345')).toBe(0.2345)
  expect(parseRate('1,234.5')).toBe(1234.5)
  expect(parseRate('0')).toBeNull()
  expect(parseRate('1.123456789')).toBeNull()
  expect(parseRate('abc')).toBeNull()
})
