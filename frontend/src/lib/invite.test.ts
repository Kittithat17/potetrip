import { expect, it } from 'vitest'
import { parseInviteInput } from './invite'

it('extracts invite codes', () => {
  expect(parseInviteInput('ABCDEFGH23')).toBe('ABCDEFGH23')
  expect(parseInviteInput('  abcdefgh23 ')).toBe('ABCDEFGH23')
  expect(parseInviteInput('https://split.app/t/ABCDEFGH23')).toBe('ABCDEFGH23')
  expect(parseInviteInput('http://localhost:5173/t/abcdefgh23?x=1')).toBe('ABCDEFGH23')
})

it('rejects junk', () => {
  expect(parseInviteInput('')).toBeNull()
  expect(parseInviteInput('short')).toBeNull()
  expect(parseInviteInput('ABCDEFGH10')).toBeNull() // 0, 1 are not in the alphabet
})
