import { describe, expect, it } from 'vitest'
import { formatAmount, formatMoney, parseAmount, splitEqual } from './money'

describe('parseAmount', () => {
  it.each([
    ['125.50', 'THB', 12550],
    ['1,250.5', 'THB', 125050],
    ['100', 'THB', 10000],
    ['0.01', 'THB', 1],
    ['3000', 'JPY', 3000],
    [' 12 ', 'THB', 1200],
    ['12.', 'THB', 1200],
  ])('%s %s -> %d', (input, cur, want) => {
    expect(parseAmount(input, cur)).toBe(want)
  })

  it.each([['', 'THB'], ['-5', 'THB'], ['1.234', 'THB'], ['10.5', 'JPY'], ['abc', 'THB'], ['1e3', 'THB']])(
    'rejects %s %s',
    (input, cur) => {
      expect(parseAmount(input, cur)).toBeNull()
    },
  )
})

describe('formatAmount / formatMoney', () => {
  it('formats minor units', () => {
    expect(formatAmount(125050, 'THB')).toBe('1,250.50')
    expect(formatAmount(5, 'THB')).toBe('0.05')
    expect(formatAmount(1234567, 'JPY')).toBe('1,234,567')
    expect(formatAmount(125050, 'THB', false)).toBe('1250.50')
    expect(formatAmount(-3333, 'THB')).toBe('-33.33')
  })

  it('adds a symbol or code', () => {
    expect(formatMoney(12550, 'THB')).toBe('฿125.50')
    expect(formatMoney(-12550, 'THB')).toBe('-฿125.50')
    expect(formatMoney(3000, 'JPY')).toBe('¥3,000')
    expect(formatMoney(1000, 'MYR')).toBe('10.00 MYR')
  })

  it('round-trips through parseAmount', () => {
    for (const v of [1, 99, 100, 12550, 987654321]) {
      expect(parseAmount(formatAmount(v, 'THB'), 'THB')).toBe(v)
    }
  })
})

describe('splitEqual', () => {
  it('gives the remainder to the first people', () => {
    expect(splitEqual(10000, 3)).toEqual([3334, 3333, 3333])
    expect(splitEqual(10001, 3)).toEqual([3334, 3334, 3333])
    expect(splitEqual(1, 3)).toEqual([1, 0, 0])
    expect(splitEqual(100, 0)).toEqual([])
  })
})
