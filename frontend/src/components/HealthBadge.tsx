import { useQuery } from '@tanstack/react-query'
import { getHealth } from '../api/client'
import { useI18n } from '../i18n/context'

/** Server status. `compact` renders a small dot for use on the blue header. */
export function HealthBadge({ compact = false }: { compact?: boolean }) {
  const { t } = useI18n()
  const { data, isPending, isError } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 10_000,
    retry: false,
  })

  const state = isPending ? 'pending' : isError || data?.status !== 'ok' ? 'down' : 'ok'
  const label = t(`health.${state}`)

  if (compact) {
    const dot = { pending: 'bg-slate-300', down: 'bg-red-400', ok: 'bg-emerald-400' }[state]
    return (
      <span role="status" title={label} className="flex items-center gap-1.5 rounded-full bg-white/15 px-2.5 py-1 text-xs">
        <span className={`size-2 rounded-full ${dot}`} />
        {state === 'down' ? t('health.offline') : state === 'ok' ? t('health.online') : '…'}
        <span className="sr-only">{label}</span>
      </span>
    )
  }

  const cls = { pending: 'bg-slate-200 text-slate-600', down: 'bg-red-100 text-red-700', ok: 'bg-emerald-100 text-emerald-700' }[state]
  return (
    <span role="status" className={`inline-flex items-center rounded-full px-3 py-1 text-sm ${cls}`}>
      {label}
    </span>
  )
}
