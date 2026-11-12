import { expect, it } from 'vitest'
import { currencyFlag, currencyName } from './currencyInfo'

it('names and flags currencies', () => {
  expect(currencyName('JPY', 'en-GB')).toBe('Japanese Yen')
  expect(currencyName('JPY', 'th-TH')).toMatch(/เยน/)
  expect(currencyName('XYZ')).toBe('XYZ')
  expect(currencyFlag('THB')).toBe('🇹🇭')
  expect(currencyFlag('EUR')).toBe('🇪🇺')
  expect(currencyFlag('1')).toBe('💱')
})
