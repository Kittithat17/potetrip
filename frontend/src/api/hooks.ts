import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from './client'
import * as api from './endpoints'
import type { ExpenseInput, Transfer } from './types'

// Spec: poll every 10s so everyone in the trip sees the same data (MVP sync).
const POLL = 10_000

export const keys = {
  invite: (code: string) => ['invite', code] as const,
  trip: (tripId: string) => ['trip', tripId] as const,
  members: (tripId: string) => ['trip', tripId, 'members'] as const,
  expenses: (tripId: string) => ['trip', tripId, 'expenses'] as const,
  expense: (tripId: string, id: string) => ['trip', tripId, 'expenses', id] as const,
  balances: (tripId: string) => ['trip', tripId, 'balances'] as const,
  settlement: (tripId: string) => ['trip', tripId, 'settlement'] as const,
}

// Client errors (bad invite code, not a member) won't fix themselves on retry.
export const retryUnlessClientError = (count: number, err: Error) =>
  !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2

export function useTripByInvite(code: string) {
  return useQuery({ queryKey: keys.invite(code), queryFn: () => api.getTripByInvite(code), retry: retryUnlessClientError })
}

export function useMembers(tripId: string | undefined) {
  return useQuery({
    queryKey: keys.members(tripId ?? ''),
    queryFn: () => api.listMembers(tripId!),
    enabled: !!tripId,
    refetchInterval: POLL,
  })
}

export function useExpenses(tripId: string, me: string) {
  return useQuery({ queryKey: keys.expenses(tripId), queryFn: () => api.listExpenses(tripId, me), refetchInterval: POLL })
}

export function useExpense(tripId: string, me: string, id: string | undefined) {
  return useQuery({
    queryKey: keys.expense(tripId, id ?? ''),
    queryFn: () => api.getExpense(tripId, me, id!),
    enabled: !!id,
  })
}

export function useBalances(tripId: string, me: string) {
  return useQuery({ queryKey: keys.balances(tripId), queryFn: () => api.getBalances(tripId, me), refetchInterval: POLL })
}

export function useSettlement(tripId: string, me: string) {
  return useQuery({ queryKey: keys.settlement(tripId), queryFn: () => api.getSettlement(tripId, me), refetchInterval: POLL })
}

/** Any write may change balances, so refresh everything under the trip. */
function useInvalidateTrip(tripId: string) {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: keys.trip(tripId) })
}

export function useAddMember(tripId: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({
    mutationFn: (body: { name: string; promptpay?: string | null }) => api.addMember(tripId, body),
    onSuccess: invalidate,
  })
}

export function useSaveExpense(tripId: string, me: string, id?: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({
    mutationFn: (body: ExpenseInput) =>
      id ? api.updateExpense(tripId, me, id, body) : api.createExpense(tripId, me, body),
    onSuccess: invalidate,
  })
}

export function useDeleteExpense(tripId: string, me: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (id: string) => api.deleteExpense(tripId, me, id), onSuccess: invalidate })
}

export function useMarkPaid(tripId: string, me: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (t: Transfer) => api.markPaid(tripId, me, t), onSuccess: invalidate })
}
