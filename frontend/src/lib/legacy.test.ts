import { beforeEach, describe, expect, it } from 'vitest'
import { clearLegacy, legacyTokens } from './legacy'

beforeEach(() => localStorage.clear())

describe('legacy device tokens', () => {
  it('collects tokens and then clears every pre-sign-in key, keeping other settings', () => {
    localStorage.setItem('travel-split:device:t1', JSON.stringify({ token: 'aaa', memberId: 'm1' }))
    localStorage.setItem('travel-split:device:t2', JSON.stringify({ token: 'bbb' }))
    localStorage.setItem('travel-split:device:t3', 'not json')
    localStorage.setItem('travel-split:recovery-saved:t1', '1')
    localStorage.setItem('travel-split:recent-trips', '[]')
    localStorage.setItem('travel-split:lang', 'th')

    expect(legacyTokens().sort()).toEqual(['aaa', 'bbb'])
    clearLegacy()
    expect(legacyTokens()).toEqual([])
    expect(Object.keys(localStorage)).toEqual(['travel-split:lang'])
  })
})
