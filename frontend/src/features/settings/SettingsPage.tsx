import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronRight, Info, KeyRound, Languages, LogOut, Luggage, Plus, Trash2 } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useMyTrips } from '../../api/hooks'
import { deletePasskey, listPasskeys } from '../../api/endpoints'
import { useAuth } from '../auth/context'
import { addPasskey, isPasskeyCancel, passkeysSupported } from '../auth/passkey'
import { formatShortDate } from '../../lib/datetime'
import { HealthBadge } from '../../components/feedback/HealthBadge'
import { AppFrame, BlueHeader } from '../../components/ui/shell'
import { Avatar, ErrorText } from '../../components/ui/ui'
import { LANGS, useI18n } from '../../i18n/context'
import { mainNav } from '../../app/nav'
import { useConfirm } from '../../components/feedback/confirm'

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
            detail={user.email ?? t('settings.passkeyAccount')}
          />
          <Row icon={<LogOut className="size-5" />} label={t('settings.logout')} onClick={onLogout} danger disabled={signingOut} />
        </Group>

        <PasskeysGroup />

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

/**
 * Passkeys on this account: sign in with Face ID / fingerprint instead of
 * Google, which matters where Google is blocked (mainland China).
 */
function PasskeysGroup() {
  const { t, locale } = useI18n()
  const confirm = useConfirm()
  const qc = useQueryClient()
  const list = useQuery({ queryKey: ['auth', 'passkeys'], queryFn: listPasskeys })
  const supported = passkeysSupported()
  const add = useMutation({
    mutationFn: addPasskey,
    onSuccess: (keys) => qc.setQueryData(['auth', 'passkeys'], keys),
  })
  const remove = useMutation({
    mutationFn: deletePasskey,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['auth', 'passkeys'] }),
  })
  const onRemove = async (id: string, name: string) => {
    const ok = await confirm({
      title: t('settings.passkeyDeleteTitle', { name }),
      message: t('settings.passkeyDeleteHint'),
      confirmLabel: t('settings.passkeyDeleteAction'),
    })
    if (ok) remove.mutate(id)
  }

  return (
    <Group title={t('settings.passkeys')} hint={t('settings.passkeyHint')}>
      {list.data?.map((k) => (
        <Row
          key={k.id}
          icon={<KeyRound className="size-5" />}
          label={k.name}
          detail={
            t('settings.passkeyAdded', { date: formatShortDate(k.created_at, locale) }) +
            (k.last_used_at ? ' · ' + t('settings.passkeyUsed', { date: formatShortDate(k.last_used_at, locale) }) : '')
          }
          right={
            <button
              type="button"
              onClick={() => onRemove(k.id, k.name)}
              aria-label={t('settings.passkeyDelete', { name: k.name })}
              className="rounded-full p-2 text-slate-400 hover:bg-red-50 hover:text-red-600"
            >
              <Trash2 className="size-4" />
            </button>
          }
        />
      ))}
      <Row
        icon={<Plus className="size-5" />}
        label={t('settings.passkeyAdd')}
        detail={supported ? undefined : t('settings.passkeyUnsupported')}
        onClick={() => add.mutate()}
        disabled={!supported || add.isPending}
      />
      {(add.error && !isPasskeyCancel(add.error)) || remove.error ? (
        <li className="px-4 py-2">
          <ErrorText error={add.error ?? remove.error} />
        </li>
      ) : null}
    </Group>
  )
}

function Group({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="mb-1.5 px-2 text-xs font-semibold tracking-wide text-slate-500 uppercase">{title}</h2>
      <ul className="divide-y divide-slate-100 overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200/70">{children}</ul>
      {hint && <p className="mt-1.5 px-2 text-xs text-slate-500">{hint}</p>}
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
