import { Check, UserPlus, X } from 'lucide-react'
import { useDecideJoin, useJoinRequests } from '../../api/hooks'
import { Avatar, ErrorText } from '../../components/ui'
import { useI18n } from '../../i18n/context'
import { useTrip } from './context'

/** Owner-only: people waiting to be let into the trip. Hidden when empty. */
export function JoinRequestsCard() {
  const { t } = useI18n()
  const { trip } = useTrip()
  const q = useJoinRequests(trip.id, true)
  const decide = useDecideJoin(trip.id)
  const requests = q.data ?? []
  if (requests.length === 0) return null

  return (
    <section className="rounded-3xl bg-amber-50 p-4 ring-1 ring-amber-200">
      <h2 className="mb-2 flex items-center gap-2 font-bold text-amber-900">
        <UserPlus className="size-5" /> {t('requests.title', { n: requests.length })}
      </h2>
      <ul className="space-y-2">
        {requests.map((r) => (
          <li key={r.id} className="flex items-center gap-3 rounded-2xl bg-white p-3">
            <Avatar name={r.requested_name} size="sm" />
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{r.requested_name}</p>
              <p className="text-xs text-slate-500">{r.requested_member_id ? t('requests.claims') : t('requests.newMember')}</p>
            </div>
            <button
              onClick={() => decide.mutate({ id: r.id, decision: 'reject' })}
              disabled={decide.isPending}
              aria-label={t('requests.reject', { name: r.requested_name })}
              className="rounded-full p-2 text-slate-500 ring-1 ring-slate-200 hover:bg-slate-100"
            >
              <X className="size-5" />
            </button>
            <button
              onClick={() => decide.mutate({ id: r.id, decision: 'approve' })}
              disabled={decide.isPending}
              aria-label={t('requests.approve', { name: r.requested_name })}
              className="rounded-full bg-brand-600 p-2 text-white hover:bg-brand-700"
            >
              <Check className="size-5" />
            </button>
          </li>
        ))}
      </ul>
      <ErrorText error={decide.error} />
    </section>
  )
}
