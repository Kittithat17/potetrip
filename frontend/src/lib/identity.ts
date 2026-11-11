// Remembers which member "I am" per trip. Storage can be unavailable
// (private mode, blocked site data), so every access is guarded.
const key = (tripId: string) => `travel-split:member:${tripId}`

export function getMyMemberId(tripId: string): string | null {
  try {
    return localStorage.getItem(key(tripId))
  } catch {
    return null
  }
}

export function setMyMemberId(tripId: string, memberId: string): void {
  try {
    localStorage.setItem(key(tripId), memberId)
  } catch {
    /* ignore */
  }
}

export function clearMyMemberId(tripId: string): void {
  try {
    localStorage.removeItem(key(tripId))
  } catch {
    /* ignore */
  }
}
