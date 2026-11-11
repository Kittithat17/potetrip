import { apiFetch } from './client'
import type { Balance, Expense, ExpenseInput, Member, Transfer, Trip } from './types'

export const createTrip = (body: { name: string; base_currency: string }) =>
  apiFetch<Trip>('/trips', { method: 'POST', body })

export const getTripByInvite = (code: string) => apiFetch<Trip>(`/trips/${encodeURIComponent(code)}`)

export const listMembers = (tripId: string) => apiFetch<Member[]>(`/trips/${tripId}/members`)

export const addMember = (tripId: string, body: { name: string; promptpay?: string | null }) =>
  apiFetch<Member>(`/trips/${tripId}/members`, { method: 'POST', body })

export const listExpenses = (tripId: string, me: string) =>
  apiFetch<Expense[]>(`/trips/${tripId}/expenses`, { memberId: me })

export const getExpense = (tripId: string, me: string, id: string) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses/${id}`, { memberId: me })

export const createExpense = (tripId: string, me: string, body: ExpenseInput) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses`, { method: 'POST', body, memberId: me })

export const updateExpense = (tripId: string, me: string, id: string, body: ExpenseInput) =>
  apiFetch<Expense>(`/trips/${tripId}/expenses/${id}`, { method: 'PUT', body, memberId: me })

export const deleteExpense = (tripId: string, me: string, id: string) =>
  apiFetch<void>(`/trips/${tripId}/expenses/${id}`, { method: 'DELETE', memberId: me })

export const getBalances = (tripId: string, me: string) =>
  apiFetch<{ currency: string; balances: Balance[] }>(`/trips/${tripId}/balances`, { memberId: me })

export const getSettlement = (tripId: string, me: string) =>
  apiFetch<{ currency: string; transfers: Transfer[] }>(`/trips/${tripId}/settlement`, { memberId: me })

export const markPaid = (tripId: string, me: string, body: Transfer) =>
  apiFetch<unknown>(`/trips/${tripId}/settlements/paid`, { method: 'POST', body, memberId: me })
