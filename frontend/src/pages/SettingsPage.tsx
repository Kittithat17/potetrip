import { Check, ChevronRight, Info, Languages, LogOut, Luggage } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useMyTrips } from '../api/hooks'
import { useAuth } from '../auth/context'
import { HealthBadge } from '../components/HealthBadge'
import { AppFrame, BlueHeader } from '../components/shell'
import { Avatar } from '../components/ui'
import { LANGS, useI18n } from '../i18n/context'
import { mainNav } from '../lib/nav'
import { useConfirm } from '../lib/confirm'

export function SettingsPage() {
  const { t, lang, setLang } = useI18n()
  const confirm = useConfirm()
  const { user, logout } = useAuth()
  const tripCount = useMyTrips().data?.length
  const [signingOut, setSigningOut] = useState(false)

  const onLogout = async () => {
    const ok = await confirm({ title: t('settings.logout'), message: t('settings.logoutConfirm'), confirmLabel: t('settings.logout') })
    if (!ok) return
    setSigningOut(true)
    try {
      await logout()
    } finally {
      setSigningOut(false)
    }
  }

  return (
    <AppFrame nav={mainNav}>
      <BlueHeader className="pb-6">
        <h1 className="text-xl font-bold">{t('settings.title')}</h1>
      </BlueHeader>

      <main className="mx-auto max-w-md space-y-4 px-4 pt-4">
        <Group title={t('settings.account')}>
          <Row
            icon={
              user.picture ? (
                <img src={user.picture} alt="" referrerPolicy="no-referrer" className="size-9 rounded-xl object-cover" />
              ) : (
                <Avatar name={user.name} size="sm" />
              )
            }
            bare
            label={user.name}
            detail={user.email}
          />
          <Row icon={<LogOut className="size-5" />} label={t('settings.logout')} onClick={onLogout} danger disabled={signingOut} />
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
            detail={tripCount === undefined ? undefined : tripCount === 1 ? t('settings.tripCountOne') : t('settings.tripCount', { n: tripCount })}
            to="/trips"
          />
        </Group>

        <Group title={t('settings.about')}>
          <Row icon={<Info className="size-5" />} label={t('settings.serverStatus')} right={<HealthBadge />} />
          <Row icon={<Info className="size-5" />} label={t('settings.version')} detail="0.5" />
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
  bare?: boolean // icon is a picture, not a glyph on a tinted tile
}

function Row({ icon, label, detail, right, to, onClick, danger, disabled, selected, bare }: RowProps) {
  const body = (
    <>
      {bare ? (
        <span className="flex size-9 shrink-0 items-center justify-center">{icon}</span>
      ) : (
        <span className={`flex size-9 items-center justify-center rounded-xl ${danger ? 'bg-red-50 text-red-600' : 'bg-brand-50 text-brand-600'}`}>
          {icon}
        </span>
      )}
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
