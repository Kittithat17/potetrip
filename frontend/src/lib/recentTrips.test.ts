import { beforeEach, expect, it } from 'vitest'
import { forgetTrip, getRecentTrips, rememberTrip } from './recentTrips'

beforeEach(() => localStorage.clear())

it('remembers trips newest first without duplicates', () => {
  rememberTrip({ id: 'a', code: 'AAAAAAAAAA', name: 'A' }, 1)
  rememberTrip({ id: 'b', code: 'BBBBBBBBBB', name: 'B', myName: 'แบงก์' }, 2)
  rememberTrip({ id: 'a', code: 'AAAAAAAAAA', name: 'A renamed' }, 3)

  const list = getRecentTrips()
  expect(list.map((t) => t.id)).toEqual(['a', 'b'])
  expect(list[0].name).toBe('A renamed')
})

it('keeps a known myName when revisiting without one', () => {
  rememberTrip({ id: 'a', code: 'AAAAAAAAAA', name: 'A', myName: 'ฝน' }, 1)
  rememberTrip({ id: 'a', code: 'AAAAAAAAAA', name: 'A' }, 2)
  expect(getRecentTrips()[0].myName).toBe('ฝน')
})

it('forgets a trip and survives corrupt storage', () => {
  rememberTrip({ id: 'a', code: 'AAAAAAAAAA', name: 'A' }, 1)
  forgetTrip('a')
  expect(getRecentTrips()).toEqual([])
  localStorage.setItem('travel-split:recent-trips', '{not json')
  expect(getRecentTrips()).toEqual([])
})
