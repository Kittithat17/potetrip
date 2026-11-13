import { API_BASE_URL, apiFetch } from './client'
import type { AuthConfig, Balance, Expense, ExpenseInput, JoinRequest, Me, Member, MyTrip, Passkey, PublicMember, SlipOutcome, Summary, Transfer, Trip, User } from './types'

export const createTrip = (body: { name: string; base_currency: string; creator_name?: string }) =>
  apiFetch<{ trip: Trip; member: Member }>('/trips', { method: 'POST', body })

// ---- account ----

export const getAuthConfig = () => apiFetch<AuthConfig>('/auth/config')
export const getCurrentUser = () => apiFetch<User>('/auth/me')
export const mockLogin = (name: string) => apiFetch<User>('/auth/mock', { method: 'POST', body: { name } })
export const logout = () => apiFetch<void>('/auth/logout', { method: 'POST' })
export const claimDevices = (tokens: string[]) => apiFetch<{ claimed: number }>('/auth/claim', { method: 'POST', body: { tokens } })
export const listMyTrips = () => apiFetch<MyTrip[]>('/me/trips')

// Passkeys: each ceremony is begin (options for the browser) + finish (its signed answer).
// The options/answers are WebAuthn JSON, passed through untouched.
export const passkeyLoginBegin = () => apiFetch<unknown>('/auth/passkey/login/begin', { method: 'POST' })
export const passkeyLoginFinish = (answer: unknown) => apiFetch<User>('/auth/passkey/login/finish', { method: 'POST', body: answer })
export const passkeySignupBegin = (name: string) => apiFetch<unknown>('/auth/passkey/signup/begin', { method: 'POST', body: { name } })
export const passkeySignupFinish = (answer: unknown) => apiFetch<User>('/auth/passkey/signup/finish', { method: 'POST', body: answer })
export const passkeyRegisterBegin = () => apiFetch<unknown>('/auth/passkey/register/begin', { method: 'POST' })
export const passkeyRegisterFinish = (answer: unknown) => apiFetch<Passkey[]>('/auth/passkey/register/finish', { method: 'POST', body: answer })
export const listPasskeys = () => apiFetch<Passkey[]>('/auth/passkeys')
export const deletePasskey = (id: string) => apiFetch<void>(`/auth/passkeys/${encodeURIComponent(id)}`, { method: 'DELETE' })

/** Full-page navigation: Google's sign-in page can't be fetched. */
export const googleSignInUrl = (next: string) => `${API_BASE_URL}/auth/google/start?next=${encodeURIComponent(next)}`

// ---- trips ----

export const getTripByInvite = (code: string) =>
  apiFetch<{ trip: Trip; members: PublicMember[] }>(`/trips/${encodeURIComponent(code)}`)

export const requestJoin = (tripId: string, body: { name?: string; member_id?: string; promptpay?: string | null }) =>
  apiFetch<{ request: JoinRequest }>(`/trips/${tripId}/join`, { method: 'POST', body })

export const getMe = (tripId: string) => apiFetch<Me>(`/trips/${tripId}/me`)

export const leaveTrip = (tripId: string) => apiFetch<void>(`/trips/${tripId}/me`, { method: 'DELETE' })

export const updateTrip = (tripId: string, body: { name?: string; base_currency?: string }) =>
  apiFetch<Trip>(`/trips/${tripId}`, { method: 'PATCH', body })

export const updateMe = (tripId: string, body: { promptpay: string | null }) =>
  apiFetch<Member>(`/trips/${tripId}/me`, { method: 'PATCH', body })

export const getSummary = (tripId: string) => apiFetch<Summary>(`/trips/${tripId}/summary`)

export const listMembers = (tripId: string) => apiFetch<Member[]>(`/trips/${tripId}/members`)

export const listJoinRequests = (tripId: string) =>
  apiFetch<JoinRequest[]>(`/trips/${tripId}/join-requests`)

export const decideJoin = (tripId: string, requestId: string, decision: 'approve' | 'reject') =>
  apiFetch<unknown>(`/trips/${tripId}/join-requests/${requestId}/${decision}`, { method: 'POST' })

export const listExpenses = (tripId: string) =>
  apiFetch<Expense[]>(`/trips/${tripId}/expenses`)

export const getExpense = (tripId: string, id: string) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses/${id}`)

export const createExpense = (tripId: string, body: ExpenseInput) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses`, { method: 'POST', body })

export const updateExpense = (tripId: string, id: string, body: ExpenseInput) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses/${id}`, { method: 'PUT', body })

export const deleteExpense = (tripId: string, id: string) =>
  apiFetch<void>(`/trips/${tripId}/expenses/${id}`, { method: 'DELETE' })

export const getBalances = (tripId: string) =>
  apiFetch<{ currency: string; balances: Balance[] }>(`/trips/${tripId}/balances`)

export const getSettlement = (tripId: string) =>
  apiFetch<{ currency: string; transfers: Transfer[] }>(`/trips/${tripId}/settlement`)

export const markPaid = (tripId: string, body: Transfer) =>
  apiFetch<unknown>(`/trips/${tripId}/settlements/paid`, { method: 'POST', body })

/** Uploads one image so the UI can show per-image progress; the API also accepts several at once. */
export function uploadSlip(tripId: string, image: Blob, filename: string) {
  const form = new FormData()
  form.append('files', image, filename)
  return apiFetch<{ results: SlipOutcome[] }>(`/trips/${tripId}/slips`, { method: 'POST', body: form }).then(
    (r) => r.results[0],
  )
}

export type FxQuote = {
  from: string
  to: string
  rate: string
  rate_date: string
  requested_date: string
  source: string
  stale: boolean
}

/** Reference rate for a day (YYYY-MM-DD), or the latest one. */
export const getFx = (from: string, to: string, date?: string) =>
  apiFetch<FxQuote>(
    `/fx?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}${date ? `&date=${encodeURIComponent(date)}` : ''}`,
  )

export const getFxCurrencies = () => apiFetch<{ currencies: string[] }>('/fx/currencies').then((r) => r.currencies)
