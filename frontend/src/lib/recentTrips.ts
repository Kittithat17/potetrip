// Trips this browser has opened, newest first, so people can get back to a
// trip without the invite link. Stored per device only.
export type RecentTrip = {
  id: string
  code: string
  name: string
  myName?: string
  visitedAt: number
}

const KEY = 'travel-split:recent-trips'
const MAX = 20

export function getRecentTrips(): RecentTrip[] {
  try {
    const raw = localStorage.getItem(KEY)
    const list = raw ? (JSON.parse(raw) as RecentTrip[]) : []
    return Array.isArray(list) ? list.sort((a, b) => b.visitedAt - a.visitedAt) : []
  } catch {
    return []
  }
}

function save(list: RecentTrip[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(list.slice(0, MAX)))
  } catch {
    /* storage unavailable */
  }
}

/** Adds or refreshes a trip. Keeps the previously known myName if none is given. */
export function rememberTrip(t: Omit<RecentTrip, 'visitedAt'>, now = Date.now()): void {
  const list = getRecentTrips()
  const prev = list.find((x) => x.id === t.id)
  const entry: RecentTrip = { ...t, myName: t.myName ?? prev?.myName, visitedAt: now }
  save([entry, ...list.filter((x) => x.id !== t.id)])
}

export function forgetTrip(id: string): void {
  save(getRecentTrips().filter((x) => x.id !== id))
}
