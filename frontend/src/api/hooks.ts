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

export function useMe(tripId: string | undefined, token: string | undefined) {
  return useQuery({
    queryKey: [...keys.trip(tripId ?? ''), 'me', token],
    queryFn: () => api.getMe(tripId!, token!),
    enabled: !!tripId && !!token,
    // A pending device watches closely for the owner's decision.
    refetchInterval: (q) => (q.state.data?.status === 'pending' ? 4_000 : 30_000),
  })
}

export function useMembers(tripId: string | undefined, token: string | undefined) {
  return useQuery({
    queryKey: keys.members(tripId ?? ''),
    queryFn: () => api.listMembers(tripId!, token!),
    enabled: !!tripId && !!token,
    refetchInterval: POLL,
  })
}

export function useUpdateTrip(tripId: string, token: string, inviteCode: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name?: string; base_currency?: string }) => api.updateTrip(tripId, token, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.invite(inviteCode) })
      qc.invalidateQueries({ queryKey: keys.trip(tripId) })
    },
  })
}

export function useJoinRequests(tripId: string, token: string, enabled: boolean) {
  return useQuery({
    queryKey: [...keys.trip(tripId), 'join-requests'],
    queryFn: () => api.listJoinRequests(tripId, token),
    enabled,
    refetchInterval: POLL,
  })
}

export function useDecideJoin(tripId: string, token: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'approve' | 'reject' }) => api.decideJoin(tripId, token, id, decision),
    onSuccess: invalidate,
  })
}

export function useExpenses(tripId: string, token: string) {
  return useQuery({ queryKey: keys.expenses(tripId), queryFn: () => api.listExpenses(tripId, token), refetchInterval: POLL })
}

export function useExpense(tripId: string, token: string, id: string | undefined) {
  return useQuery({
    queryKey: keys.expense(tripId, id ?? ''),
    queryFn: () => api.getExpense(tripId, token, id!),
    enabled: !!id,
  })
}

export function useBalances(tripId: string, token: string) {
  return useQuery({ queryKey: keys.balances(tripId), queryFn: () => api.getBalances(tripId, token), refetchInterval: POLL })
}

export function useSettlement(tripId: string, token: string) {
  return useQuery({ queryKey: keys.settlement(tripId), queryFn: () => api.getSettlement(tripId, token), refetchInterval: POLL })
}

/** Any write may change balances, so refresh everything under the trip. */
function useInvalidateTrip(tripId: string) {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: keys.trip(tripId) })
}

export function useSaveExpense(tripId: string, token: string, id?: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({
    mutationFn: (body: ExpenseInput) =>
      id ? api.updateExpense(tripId, token, id, body) : api.createExpense(tripId, token, body),
    onSuccess: invalidate,
  })
}

export function useDeleteExpense(tripId: string, token: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (id: string) => api.deleteExpense(tripId, token, id), onSuccess: invalidate })
}

export function useMarkPaid(tripId: string, token: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (t: Transfer) => api.markPaid(tripId, token, t), onSuccess: invalidate })
}
