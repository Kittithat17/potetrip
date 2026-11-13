// Before sign-in existed, each trip's access was a device token kept in
// localStorage. After the first sign-in we hand those tokens to the server
// (which moves the memberships onto the account and voids the tokens), then
// delete everything that era left behind. Nothing secret is stored locally
// any more.
const DEVICE_PREFIX = 'travel-split:device:'
const LEFTOVER = [/^travel-split:device:/, /^travel-split:recovery-saved:/, /^travel-split:recent-trips$/]

function keys(): string[] {
  try {
    return Object.keys(localStorage)
  } catch {
    return []
  }
}

/** Device tokens still on this browser, if any. */
export function legacyTokens(): string[] {
  const out: string[] = []
  for (const k of keys()) {
    if (!k.startsWith(DEVICE_PREFIX)) continue
    try {
      const v = JSON.parse(localStorage.getItem(k) ?? 'null')
      if (v && typeof v.token === 'string') out.push(v.token)
    } catch {
      /* unreadable entry: it gets removed below anyway */
    }
  }
  return out
}

/** Removes every pre-sign-in entry (tokens, reminders, the old trip list). */
export function clearLegacy(): void {
  for (const k of keys()) {
    if (LEFTOVER.some((re) => re.test(k))) {
      try {
        localStorage.removeItem(k)
      } catch {
        /* ignore */
      }
    }
  }
}
