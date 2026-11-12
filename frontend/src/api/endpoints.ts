import { apiFetch } from './client'
import type { Balance, Expense, ExpenseInput, JoinRequest, Me, Member, PublicMember, SlipOutcome, Transfer, Trip } from './types'

export const createTrip = (body: { name: string; base_currency: string; creator_name: string }) =>
  apiFetch<{ trip: Trip; member: Member; token: string }>('/trips', { method: 'POST', body })

export const getTripByInvite = (code: string) =>
  apiFetch<{ trip: Trip; members: PublicMember[] }>(`/trips/${encodeURIComponent(code)}`)

export const requestJoin = (tripId: string, body: { name?: string; member_id?: string; promptpay?: string | null }) =>
  apiFetch<{ token: string; request: JoinRequest }>(`/trips/${tripId}/join`, { method: 'POST', body })

export const getMe = (tripId: string, token: string) => apiFetch<Me>(`/trips/${tripId}/me`, { token })

export const updateTrip = (tripId: string, token: string, body: { name?: string; base_currency?: string }) =>
  apiFetch<Trip>(`/trips/${tripId}`, { method: 'PATCH', body, token })

export const listMembers = (tripId: string, token: string) => apiFetch<Member[]>(`/trips/${tripId}/members`, { token })

export const listJoinRequests = (tripId: string, token: string) =>
  apiFetch<JoinRequest[]>(`/trips/${tripId}/join-requests`, { token })

export const decideJoin = (tripId: string, token: string, requestId: string, decision: 'approve' | 'reject') =>
  apiFetch<unknown>(`/trips/${tripId}/join-requests/${requestId}/${decision}`, { method: 'POST', token })

export const listExpenses = (tripId: string, token: string) =>
  apiFetch<Expense[]>(`/trips/${tripId}/expenses`, { token })

export const getExpense = (tripId: string, token: string, id: string) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses/${id}`, { token })

export const createExpense = (tripId: string, token: string, body: ExpenseInput) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses`, { method: 'POST', body, token })

export const updateExpense = (tripId: string, token: string, id: string, body: ExpenseInput) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses/${id}`, { method: 'PUT', body, token })

export const deleteExpense = (tripId: string, token: string, id: string) =>
  apiFetch<void>(`/trips/${tripId}/expenses/${id}`, { method: 'DELETE', token })

export const getBalances = (tripId: string, token: string) =>
  apiFetch<{ currency: string; balances: Balance[] }>(`/trips/${tripId}/balances`, { token })

export const getSettlement = (tripId: string, token: string) =>
  apiFetch<{ currency: string; transfers: Transfer[] }>(`/trips/${tripId}/settlement`, { token })

export const markPaid = (tripId: string, token: string, body: Transfer) =>
  apiFetch<unknown>(`/trips/${tripId}/settlements/paid`, { method: 'POST', body, token })

/** Uploads one image so the UI can show per-image progress; the API also accepts several at once. */
export function uploadSlip(tripId: string, token: string, image: Blob, filename: string) {
  const form = new FormData()
  form.append('files', image, filename)
  return apiFetch<{ results: SlipOutcome[] }>(`/trips/${tripId}/slips`, { method: 'POST', body: form, token }).then(
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

export const getFx = (from: string, to: string) =>
  apiFetch<FxQuote>(`/fx?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`)

export const getFxCurrencies = () => apiFetch<{ currencies: string[] }>('/fx/currencies').then((r) => r.currencies)
