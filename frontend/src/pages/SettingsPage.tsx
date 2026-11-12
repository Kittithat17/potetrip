import { Check, ChevronRight, Info, Languages, Luggage, Trash2, UserRound } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { HealthBadge } from '../components/HealthBadge'
import { AppFrame, BlueHeader } from '../components/shell'
import { clearIdentity } from '../lib/identity'
import { LANGS, useI18n } from '../i18n/context'
import { mainNav } from '../lib/nav'
import { useConfirm } from '../lib/confirm'
import { forgetAllTrips, getRecentTrips } from '../lib/recentTrips'

export function SettingsPage() {
  const { t, lang, setLang } = useI18n()
  const confirm = useConfirm()
  const [tripCount, setTripCount] = useState(() => getRecentTrips().length)

  const clearDevice = async () => {
    const ok = await confirm({ title: t('settings.clearDevice'), message: t('settings.clearConfirm'), confirmLabel: t('settings.clearAction') })
    if (!ok) return
    for (const tr of getRecentTrips()) clearIdentity(tr.id)
    forgetAllTrips()
    setTripCount(0)
  }

  return (
    <AppFrame nav={mainNav}>
      <BlueHeader className="pb-6">
        <h1 className="text-xl font-bold">{t('settings.title')}</h1>
      </BlueHeader>

      <main className="mx-auto max-w-md space-y-4 px-4 pt-4">
        <Group title={t('settings.account')}>
          <Row icon={<UserRound className="size-5" />} label={t('settings.profile')} detail={t('settings.comingSoon')} disabled />
        </Group>

        <Group title={t('settings.language')}>
          {LANGS.map((l) => (
            <Row
              key={l.value}
              icon={<Languages className="size-5" />}
              label={l.label}
              onClick={() => setLang(l.value)}
              right={lang === l.value ? <Check className="size-5 text-brand-600" aria-label="✓" /> : undefined}
              selected={lang === l.value}
            />
          ))}
        </Group>

        <Group title={t('settings.trips')}>
          <Row
            icon={<Luggage className="size-5" />}
            label={t('settings.myTrips')}
            detail={tripCount === 1 ? t('settings.tripCountOne') : t('settings.tripCount', { n: tripCount })}
            to="/trips"
          />
          <Row
            icon={<Trash2 className="size-5" />}
            label={t('settings.clearDevice')}
            detail={t('settings.clearDeviceDetail')}
            onClick={clearDevice}
            danger
            disabled={tripCount === 0}
          />
        </Group>

        <Group title={t('settings.about')}>
          <Row icon={<Info className="size-5" />} label={t('settings.serverStatus')} right={<HealthBadge />} />
          <Row icon={<Info className="size-5" />} label={t('settings.version')} detail="0.3 (M3)" />
        </Group>
      </main>
    </AppFrame>
  )
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="mb-1.5 px-2 text-xs font-semibold tracking-wide text-slate-500 uppercase">{title}</h2>
      <ul className="divide-y divide-slate-100 overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200/70">{children}</ul>
    </section>
  )
}

type RowProps = {
  icon: ReactNode
  label: string
  detail?: string
  right?: ReactNode
  to?: string
  onClick?: () => void
  danger?: boolean
  disabled?: boolean
  selected?: boolean // radio-style rows (language)
}

function Row({ icon, label, detail, right, to, onClick, danger, disabled, selected }: RowProps) {
  const body = (
    <>
      <span className={`flex size-9 items-center justify-center rounded-xl ${danger ? 'bg-red-50 text-red-600' : 'bg-brand-50 text-brand-600'}`}>
        {icon}
      </span>
      <span className="min-w-0 flex-1">
        <span className={`block font-medium ${danger ? 'text-red-600' : ''}`}>{label}</span>
        {detail && <span className="block truncate text-xs text-slate-500">{detail}</span>}
      </span>
      {right}
      {(to || onClick) && !disabled && !danger && selected === undefined && <ChevronRight className="size-5 text-slate-400" />}
    </>
  )
  const cls = `flex w-full items-center gap-3 px-4 py-3 text-left ${disabled ? 'opacity-50' : to || onClick ? 'hover:bg-slate-50' : ''}`
  return (
    <li>
      {to && !disabled ? (
        <Link to={to} className={cls}>
          {body}
        </Link>
      ) : onClick ? (
        <button type="button" onClick={onClick} disabled={disabled} className={cls} aria-pressed={selected}>
          {body}
        </button>
      ) : (
        <div className={cls}>{body}</div>
      )}
    </li>
  )
}
