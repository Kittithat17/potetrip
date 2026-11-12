// This device's secret credential per trip. The token proves the device was
// approved into the trip; memberId is cached so lists can find "me" without
// a round trip. Storage can be unavailable, so every access is guarded.
export type TripIdentity = { token: string; memberId?: string }

const key = (tripId: string) => `travel-split:device:${tripId}`

export function getIdentity(tripId: string): TripIdentity | null {
  try {
    const v = JSON.parse(localStorage.getItem(key(tripId)) ?? 'null')
    return v && typeof v.token === 'string' ? v : null
  } catch {
    return null
  }
}

export function setIdentity(tripId: string, id: TripIdentity): void {
  try {
    localStorage.setItem(key(tripId), JSON.stringify(id))
  } catch {
    /* ignore */
  }
}

export function clearIdentity(tripId: string): void {
  try {
    localStorage.removeItem(key(tripId))
  } catch {
    /* ignore */
  }
}
