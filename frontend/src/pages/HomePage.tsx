import { ChevronRight, CircleHelp, Luggage, Plus, Search, Ticket } from 'lucide-react'
import { useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { HealthBadge } from '../components/HealthBadge'
import { AppFrame, BlueHeader, IconAction, Sheet, TripCover } from '../components/shell'
import { useI18n } from '../i18n/context'
import { mainNav } from '../lib/nav'
import { parseInviteInput } from '../lib/invite'
import { getRecentTrips } from '../lib/recentTrips'

export function HomePage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [invite, setInvite] = useState('')
  const [inviteError, setInviteError] = useState('')
  const [showHelp, setShowHelp] = useState(false)
  const inviteRef = useRef<HTMLInputElement>(null)
  const recent = getRecentTrips()

  const onJoin = (e: FormEvent) => {
    e.preventDefault()
    const code = parseInviteInput(invite)
    if (!code) {
      setInviteError(t('home.invalidInvite'))
      return
    }
    navigate(`/t/${code}`)
  }

  return (
    <AppFrame nav={mainNav}>
      <BlueHeader className="pb-20">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold tracking-tight">Travel Split</h1>
          <HealthBadge compact />
        </div>
        <form onSubmit={onJoin} className="mt-3">
          <label className="flex items-center gap-2 rounded-xl bg-white px-3 py-2.5 text-slate-900 shadow-sm">
            <Search className="size-5 shrink-0 text-slate-400" />
            <input
              ref={inviteRef}
              value={invite}
              onChange={(e) => {
                setInvite(e.target.value)
                setInviteError('')
              }}
              placeholder={t('home.searchPlaceholder')}
              aria-label={t('home.searchLabel')}
              autoCapitalize="characters"
              className="min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-slate-400"
            />
            {invite && (
              <button type="submit" className="rounded-lg bg-brand-600 px-3 py-1 text-sm font-medium text-white">
                {t('home.join')}
              </button>
            )}
          </label>
          {inviteError && <p className="mt-2 text-sm text-amber-200">{inviteError}</p>}
        </form>
        {recent.length > 0 && (
          <div className="no-scrollbar -mx-4 mt-3 flex gap-2 overflow-x-auto px-4">
            {recent.slice(0, 6).map((tr) => (
              <Link key={tr.id} to={`/t/${tr.code}`} className="shrink-0 rounded-lg bg-white/15 px-3 py-1 text-sm backdrop-blur hover:bg-white/25">
                {tr.name}
              </Link>
            ))}
          </div>
        )}
      </BlueHeader>

      <main className="mx-auto -mt-14 max-w-md space-y-4 px-4">
        <Sheet>
          <div className="grid grid-cols-4 gap-2">
            <IconAction icon={Plus} label={t('home.newTrip')} to="/new" />
            <IconAction icon={Luggage} label={t('home.myTrips')} to="/trips" />
            <IconAction icon={Ticket} label={t('home.enterCode')} onClick={() => inviteRef.current?.focus()} />
            <IconAction icon={CircleHelp} label={t('home.howTo')} onClick={() => setShowHelp((v) => !v)} />
          </div>
          {showHelp && (
            <ol className="mt-4 list-decimal space-y-1 rounded-2xl bg-brand-50 p-4 pl-8 text-sm text-slate-700">
              <li>{t('home.help1')}</li>
              <li>{t('home.help2')}</li>
              <li>{t('home.help3')}</li>
            </ol>
          )}
        </Sheet>

        {recent.length > 0 && (
          <section>
            <div className="mb-2 flex items-center justify-between px-1">
              <h2 className="text-lg font-bold">{t('home.recent')}</h2>
              <Link to="/trips" className="flex items-center text-sm text-brand-600">
                {t('home.seeAll')} <ChevronRight className="size-4" />
              </Link>
            </div>
            <div className="no-scrollbar -mx-4 flex gap-3 overflow-x-auto px-4 pb-1">
              {recent.slice(0, 8).map((tr) => (
                <Link key={tr.id} to={`/t/${tr.code}`} className="w-40 shrink-0 overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200/70">
                  <TripCover name={tr.name} className="h-24" />
                  <div className="p-3">
                    <p className="truncate font-semibold">{tr.name}</p>
                    <p className="truncate text-xs text-slate-500">{tr.myName ? t('home.youAre', { name: tr.myName }) : tr.code}</p>
                  </div>
                </Link>
              ))}
            </div>
          </section>
        )}

        <Link
          to="/new"
          className="relative block overflow-hidden rounded-3xl bg-gradient-to-br from-brand-500 to-indigo-700 p-5 text-white shadow-md"
        >
          <p className="text-sm text-brand-100">{t('home.promoKicker')}</p>
          <p className="mt-1 text-2xl leading-tight font-bold">
            {t('home.promoTitle1')}
            <br />
            {t('home.promoTitle2')}
          </p>
          <span className="mt-4 inline-flex items-center gap-1 rounded-full bg-white px-4 py-1.5 text-sm font-semibold text-brand-700">
            {t('home.promoCta')} <ChevronRight className="size-4" />
          </span>
          <span aria-hidden className="absolute -right-4 -bottom-6 text-[7rem] opacity-20">
            ✈
          </span>
        </Link>
      </main>
    </AppFrame>
  )
}
