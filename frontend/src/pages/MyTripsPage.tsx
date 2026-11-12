import { useQueries } from '@tanstack/react-query'
import { ChevronLeft, Plus, X } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { getBalances } from '../api/endpoints'
import { AppFrame, BlueHeader, TripCover } from '../components/shell'
import { useI18n } from '../i18n/context'
import { mainNav } from '../lib/nav'
import { useConfirm } from '../lib/confirm'
import { getIdentity } from '../lib/identity'
import { formatMoney } from '../lib/money'
import { Approx } from '../components/Approx'
import { forgetTrip, getRecentTrips, type RecentTrip } from '../lib/recentTrips'

export function MyTripsPage() {
  const { t } = useI18n()
  const confirm = useConfirm()
  const [trips, setTrips] = useState<RecentTrip[]>(getRecentTrips)

  // My balance in each trip, for trips where this device knows who I am.
  const balances = useQueries({
    queries: trips.map((tr) => {
      const id = getIdentity(tr.id)
      const me = id?.memberId
      return {
        queryKey: ['trip', tr.id, 'balances'],
        queryFn: () => getBalances(tr.id, id!.token),
        enabled: !!id?.memberId,
        retry: false,
        staleTime: 30_000,
        select: (d: Awaited<ReturnType<typeof getBalances>>) => ({
          currency: d.currency,
          net: d.balances.find((b) => b.member_id === me)?.net,
        }),
      }
    }),
  })

  const hide = async (tr: RecentTrip) => {
    const ok = await confirm({ title: t('myTrips.hideTitle', { name: tr.name }), message: t('myTrips.hideHint'), confirmLabel: t('myTrips.hideAction') })
    if (!ok) return
    forgetTrip(tr.id)
    setTrips(getRecentTrips())
  }

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
        {trips.length === 0 ? (
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
                  <Link to={`/t/${tr.code}`} className="flex min-w-0 flex-1">
                    <TripCover name={tr.name} className="w-24 shrink-0" />
                    <div className="min-w-0 flex-1 p-3 pr-10">
                      <p className="truncate text-base font-bold">{tr.name}</p>
                      <p className="truncate text-xs text-slate-500">
                        {tr.myName ? `${t('home.youAre', { name: tr.myName })} · ` : ''}
                        {t('myTrips.code', { code: tr.code })}
                      </p>
                      <p className="mt-2 text-sm">
                        {b?.net === undefined ? (
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
                  <button
                    onClick={() => hide(tr)}
                    aria-label={t('myTrips.hide', { name: tr.name })}
                    className="absolute top-2 right-2 rounded-full p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
                  >
                    <X className="size-4" />
                  </button>
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
