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
  myTrips: ['me', 'trips'] as const,
}

// Client errors (bad invite code, not a member) won't fix themselves on retry.
export const retryUnlessClientError = (count: number, err: Error) =>
  !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2

/** Trips the signed-in user is in or waiting to join (kept on the server). */
export function useMyTrips() {
  return useQuery({ queryKey: keys.myTrips, queryFn: api.listMyTrips, staleTime: 30_000 })
}

export function useTripByInvite(code: string) {
  return useQuery({ queryKey: keys.invite(code), queryFn: () => api.getTripByInvite(code), retry: retryUnlessClientError })
}

export function useMe(tripId: string | undefined) {
  return useQuery({
    queryKey: [...keys.trip(tripId ?? ''), 'me'],
    queryFn: () => api.getMe(tripId!),
    enabled: !!tripId,
    // A pending device watches closely for the owner's decision.
    refetchInterval: (q) => (q.state.data?.status === 'pending' ? 4_000 : 30_000),
  })
}

export function useMembers(tripId: string | undefined) {
  return useQuery({
    queryKey: keys.members(tripId ?? ''),
    queryFn: () => api.listMembers(tripId!),
    enabled: !!tripId,
    refetchInterval: POLL,
  })
}

export function useUpdateTrip(tripId: string, inviteCode: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name?: string; base_currency?: string }) => api.updateTrip(tripId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.invite(inviteCode) })
      qc.invalidateQueries({ queryKey: keys.trip(tripId) })
    },
  })
}

export function useUpdateMe(tripId: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (promptpay: string | null) => api.updateMe(tripId, { promptpay }), onSuccess: invalidate })
}

export function useSummary(tripId: string) {
  return useQuery({ queryKey: [...keys.trip(tripId), 'summary'], queryFn: () => api.getSummary(tripId), refetchInterval: POLL })
}

/** The rate an expense on `date` would be converted at (none needed for the base currency). */
export function useExpenseRate(from: string, to: string, date: string | undefined) {
  return useQuery({
    queryKey: ['fx', from, to, date ?? 'latest'],
    queryFn: () => api.getFx(from, to, date),
    enabled: from !== to,
    staleTime: 10 * 60_000,
    retry: false,
  })
}

export function useJoinRequests(tripId: string, enabled: boolean) {
  return useQuery({
    queryKey: [...keys.trip(tripId), 'join-requests'],
    queryFn: () => api.listJoinRequests(tripId),
    enabled,
    refetchInterval: POLL,
  })
}

export function useDecideJoin(tripId: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'approve' | 'reject' }) => api.decideJoin(tripId, id, decision),
    onSuccess: invalidate,
  })
}

export function useRejectAllJoins(tripId: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: () => api.rejectAllJoins(tripId), onSuccess: invalidate })
}

export function useActivity(tripId: string) {
  return useQuery({ queryKey: [...keys.trip(tripId), 'activity'], queryFn: () => api.getActivity(tripId), refetchInterval: POLL })
}

export function useExpenses(tripId: string) {
  return useQuery({ queryKey: keys.expenses(tripId), queryFn: () => api.listExpenses(tripId), refetchInterval: POLL })
}

export function useExpense(tripId: string, id: string | undefined) {
  return useQuery({
    queryKey: keys.expense(tripId, id ?? ''),
    queryFn: () => api.getExpense(tripId, id!),
    enabled: !!id,
  })
}

export function useBalances(tripId: string) {
  return useQuery({ queryKey: keys.balances(tripId), queryFn: () => api.getBalances(tripId), refetchInterval: POLL })
}

export function useSettlement(tripId: string) {
  return useQuery({ queryKey: keys.settlement(tripId), queryFn: () => api.getSettlement(tripId), refetchInterval: POLL })
}

/** Any write may change balances, so refresh everything under the trip. */
function useInvalidateTrip(tripId: string) {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: keys.trip(tripId) })
}

export function useSaveExpense(tripId: string, id?: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({
    mutationFn: (body: ExpenseInput) =>
      id ? api.updateExpense(tripId, id, body) : api.createExpense(tripId, body),
    onSuccess: invalidate,
  })
}

export function useDeleteExpense(tripId: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (id: string) => api.deleteExpense(tripId, id), onSuccess: invalidate })
}

export function useMarkPaid(tripId: string) {
  const invalidate = useInvalidateTrip(tripId)
  return useMutation({ mutationFn: (t: Transfer) => api.markPaid(tripId, t), onSuccess: invalidate })
}
