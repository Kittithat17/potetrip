import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { I18nContext, LANG_KEY, makeI18n, storedLang, type Lang } from './context'

export function I18nProvider({ children, initialLang }: { children: ReactNode; initialLang?: Lang }) {
  const [lang, setLangState] = useState<Lang>(() => initialLang ?? storedLang())

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  const value = useMemo(
    () =>
      makeI18n(lang, (l) => {
        try {
          localStorage.setItem(LANG_KEY, l)
        } catch {
          /* ignore */
        }
        setLangState(l)
      }),
    [lang],
  )
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}
