import { expect, it } from 'vitest'
import { formatApprox } from './homeApprox'

it('formats estimates readably', () => {
  expect(formatApprox(317.37)).toBe('≈ ฿317.37')
  expect(formatApprox(50)).toBe('≈ ฿50')
  expect(formatApprox(12345.67)).toBe('≈ ฿12,346')
  expect(formatApprox(-633.333)).toBe('≈ -฿633.33')
  // 750 JPY at 0.21158 = 158.685: both signs must round the same way.
  expect(formatApprox(158.685)).toBe('≈ ฿158.69')
  expect(formatApprox(-158.685)).toBe('≈ -฿158.69')
})
