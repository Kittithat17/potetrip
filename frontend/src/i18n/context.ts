import { createContext, useContext } from 'react'
import { en, type Messages } from './en'
import { th } from './th'

export type Lang = 'en' | 'th'
export const LANGS: { value: Lang; label: string }[] = [
  { value: 'en', label: 'English' },
  { value: 'th', label: 'ไทย' },
]
const DICTS: Record<Lang, Messages> = { en, th }
const LOCALES: Record<Lang, string> = { en: 'en-GB', th: 'th-TH' }
export const LANG_KEY = 'travel-split:lang'
export const DEFAULT_LANG: Lang = 'en'

// "trip.hello" style paths to every string in the dictionary.
type Leaves<T, P extends string = ''> = {
  [K in keyof T & string]: T[K] extends string ? `${P}${K}` : Leaves<T[K], `${P}${K}.`>
}[keyof T & string]
export type MsgKey = Leaves<Messages>
type Vars = Record<string, string | number>

export type I18n = {
  lang: Lang
  locale: string // for Intl date/number formatting
  setLang: (l: Lang) => void
  t: (key: MsgKey, vars?: Vars) => string
  /** Looks up a key built at runtime (e.g. an API error code); undefined if missing. */
  tryT: (key: string, vars?: Vars) => string | undefined
}

export function storedLang(): Lang {
  try {
    return localStorage.getItem(LANG_KEY) === 'th' ? 'th' : DEFAULT_LANG
  } catch {
    return DEFAULT_LANG
  }
}

function lookup(dict: Messages, key: string): string | undefined {
  let cur: unknown = dict
  for (const part of key.split('.')) {
    if (cur === null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[part]
  }
  return typeof cur === 'string' ? cur : undefined
}

const fill = (s: string, vars?: Vars) => (vars ? s.replace(/\{(\w+)\}/g, (m, k) => (k in vars ? String(vars[k]) : m)) : s)

export function makeI18n(lang: Lang, setLang: (l: Lang) => void): I18n {
  const dict = DICTS[lang]
  const tryT = (key: string, vars?: Vars) => {
    const s = lookup(dict, key) ?? lookup(en, key)
    return s === undefined ? undefined : fill(s, vars)
  }
  return { lang, locale: LOCALES[lang], setLang, tryT, t: (key, vars) => tryT(key, vars) ?? key }
}

export const I18nContext = createContext<I18n | null>(null)

/** Works without a provider too (falls back to the stored language). */
export function useI18n(): I18n {
  return useContext(I18nContext) ?? makeI18n(storedLang(), () => {})
}
