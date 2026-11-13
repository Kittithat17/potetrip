import { useQueries } from '@tanstack/react-query'
import { ChevronLeft, Plus } from 'lucide-react'
import { Link } from 'react-router'
import { getBalances, getMe } from '../api/endpoints'
import { useMyTrips } from '../api/hooks'
import { AppFrame, BlueHeader, TripCover } from '../components/shell'
import { ErrorText, Spinner } from '../components/ui'
import { useI18n } from '../i18n/context'
import { mainNav } from '../lib/nav'
import { formatMoney } from '../lib/money'
import { Approx } from '../components/Approx'

export function MyTripsPage() {
  const { t } = useI18n()
  const tripsQ = useMyTrips()
  const trips = tripsQ.data ?? []

  // My balance in each trip I'm a member of.
  const balances = useQueries({
    queries: trips.map((tr) => ({
      queryKey: ['trip', tr.id, 'my-balance'],
      queryFn: async () => {
        const [me, b] = await Promise.all([getMe(tr.id), getBalances(tr.id)])
        return { currency: b.currency, net: b.balances.find((x) => x.member_id === me.member?.id)?.net }
      },
      enabled: tr.status === 'approved',
      retry: false,
      staleTime: 30_000,
    })),
  })

  return (
    <AppFrame nav={mainNav}>
      <BlueHeader className="pb-6">
        <div className="flex items-center gap-1">
          <Link to="/" aria-label={t('nav.home')} className="-ml-2 rounded-full p-1 hover:bg-white/15">
            <ChevronLeft className="size-7" />
          </Link>
          <h1 className="text-xl font-bold">{t('myTrips.title')}</h1>
        </div>
        <p className="mt-0.5 pl-8 text-sm text-brand-100">{t('myTrips.subtitle')}</p>
      </BlueHeader>

      <main className="mx-auto max-w-md space-y-3 px-4 pt-4">
        {tripsQ.isPending ? (
          <Spinner />
        ) : tripsQ.error ? (
          <ErrorText error={tripsQ.error} />
        ) : trips.length === 0 ? (
          <div className="space-y-4 rounded-3xl bg-white p-8 text-center shadow-sm">
            <p className="text-5xl">🧳</p>
            <p className="text-slate-600">{t('myTrips.empty')}</p>
            <Link to="/new" className="inline-flex items-center gap-1 rounded-full bg-brand-600 px-5 py-2 font-medium text-white">
              <Plus className="size-4" /> {t('myTrips.create')}
            </Link>
          </div>
        ) : (
          <ul className="space-y-3">
            {trips.map((tr, i) => {
              const b = balances[i]?.data
              return (
                <li key={tr.id} className="relative flex overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200/70">
                  <Link to={`/t/${tr.invite_code}`} className="flex min-w-0 flex-1">
                    <TripCover name={tr.name} className="w-24 shrink-0" />
                    <div className="min-w-0 flex-1 p-3">
                      <p className="truncate text-base font-bold">{tr.name}</p>
                      <p className="truncate text-xs text-slate-500">
                        {tr.member_name ? `${t('home.youAre', { name: tr.member_name })} · ` : ''}
                        {t('myTrips.code', { code: tr.invite_code })}
                      </p>
                      <p className="mt-2 text-sm">
                        {tr.status === 'pending' ? (
                          <span className="text-amber-700">{t('myTrips.pending')}</span>
                        ) : b?.net === undefined ? (
                          <span className="text-slate-400">{t('myTrips.open')}</span>
                        ) : b.net > 0 ? (
                          <span className="font-semibold text-emerald-600">
                            {t('myTrips.getsBack', { amount: formatMoney(b.net, b.currency) })} <Approx minor={b.net} currency={b.currency} className="inline text-xs font-normal text-slate-400" />
                          </span>
                        ) : b.net < 0 ? (
                          <span className="font-semibold text-red-600">
                            {t('myTrips.owes', { amount: formatMoney(-b.net, b.currency) })} <Approx minor={-b.net} currency={b.currency} className="inline text-xs font-normal text-slate-400" />
                          </span>
                        ) : (
                          <span className="font-semibold text-slate-500">{t('myTrips.settled')}</span>
                        )}
                      </p>
                    </div>
                  </Link>
                </li>
              )
            })}
          </ul>
        )}
        {trips.length > 0 && (
          <Link
            to="/new"
            className="flex items-center justify-center gap-1 rounded-2xl border-2 border-dashed border-brand-300 py-4 font-medium text-brand-600"
          >
            <Plus className="size-5" /> {t('myTrips.create')}
          </Link>
        )}
      </main>
    </AppFrame>
  )
}
