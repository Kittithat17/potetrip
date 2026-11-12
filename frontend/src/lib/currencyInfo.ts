/** Currency name in the UI language, from the browser's own data (e.g. "Japanese Yen" / "เยนญี่ปุ่น"). */
export function currencyName(code: string, locale = 'en-GB'): string {
  try {
    return new Intl.DisplayNames([locale], { type: 'currency' }).of(code) ?? code
  } catch {
    return code
  }
}

/** Flag emoji from the ISO 4217 code's country prefix (EUR -> EU flag). */
export function currencyFlag(code: string): string {
  const cc = code.slice(0, 2).toUpperCase()
  if (!/^[A-Z]{2}$/.test(cc)) return '💱'
  return String.fromCodePoint(...[...cc].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65))
}
