const pad = (n: number) => String(n).padStart(2, '0')

/** ISO string -> value for <input type="datetime-local"> in local time. */
export function toLocalInput(iso: string | Date): string {
  const d = typeof iso === 'string' ? new Date(iso) : iso
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function fromLocalInput(v: string): string {
  return new Date(v).toISOString()
}

const fmts = new Map<string, Intl.DateTimeFormat>()

export function formatShortDateTime(iso: string, locale = 'en-GB'): string {
  let f = fmts.get(locale)
  if (!f) {
    f = new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
    fmts.set(locale, f)
  }
  return f.format(new Date(iso))
}

export function formatShortDate(isoDate: string, locale = 'en-GB'): string {
  return new Date(isoDate).toLocaleDateString(locale, { day: 'numeric', month: 'short', year: '2-digit' })
}
