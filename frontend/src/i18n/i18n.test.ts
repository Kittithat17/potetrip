import { describe, expect, it } from 'vitest'
import { makeI18n } from './context'
import { en } from './en'
import { th } from './th'

function keys(o: object, p = ''): string[] {
  return Object.entries(o).flatMap(([k, v]) => (typeof v === 'string' ? [p + k] : keys(v, `${p}${k}.`)))
}

describe('i18n', () => {
  it('Thai has exactly the English keys', () => {
    expect(keys(th).sort()).toEqual(keys(en).sort())
  })

  it('fills variables and looks up runtime keys', () => {
    const i = makeI18n('th', () => {})
    expect(i.t('trip.hello', { name: 'ฝน' })).toBe('สวัสดี ฝน')
    expect(i.tryT('errors.trip_not_found')).toBe('ไม่พบทริปนี้')
    expect(i.tryT('errors.no_such_code')).toBeUndefined()
    expect(makeI18n('en', () => {}).t('trip.people', { n: 3 })).toBe('3 people')
  })
})
