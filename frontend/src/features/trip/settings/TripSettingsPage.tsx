import { Crown, Lock } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useActivity, useBalances, useExpenses, useUpdateMe, useUpdateTrip } from '../../../api/hooks'
import { formatShortDateTime } from '../../../lib/datetime'
import { formatMoney } from '../../../lib/money'
import { Avatar, Button, Card, ErrorText, Field, inputClass, Page, PageHeader, TextInput } from '../../../components/ui/ui'
import { currencyName } from '../../../lib/currencyInfo'
import { CURRENCIES } from '../../../lib/money'
import { useTrip } from '../layout/context'
import { useI18n } from '../../../i18n/context'

export function TripSettingsPage() {
  const { t, locale } = useI18n()
  const { trip, me, isOwner, members, base, nav } = useTrip()
  const expensesQ = useExpenses(trip.id)
  const balancesQ = useBalances(trip.id)
  const update = useUpdateTrip(trip.id, trip.invite_code)

  const [name, setName] = useState(trip.name)
  const [currency, setCurrency] = useState(trip.base_currency)
  const [saved, setSaved] = useState(false)
  const updateMe = useUpdateMe(trip.id)
  const [promptpay, setPromptpay] = useState(me.promptpay ?? '')
  const [ppSaved, setPpSaved] = useState(false)
  const onSavePromptPay = (e: FormEvent) => {
    e.preventDefault()
    setPpSaved(false)
    updateMe.mutate(promptpay.trim() || null, { onSuccess: () => setPpSaved(true) })
  }

  // Mirrors the server rule: the currency locks once money has been recorded.
  const hasActivity =
    (expensesQ.data?.length ?? 0) > 0 || (balancesQ.data?.balances.some((b) => b.sent > 0 || b.received > 0) ?? false)
  const currencyLocked = hasActivity || !expensesQ.data
  const dirty = name.trim() !== trip.name || currency !== trip.base_currency

  const onSave = (e: FormEvent) => {
    e.preventDefault()
    setSaved(false)
    update.mutate(
      { name, base_currency: currency },
      { onSuccess: () => setSaved(true) },
    )
  }

  return (
    <Page nav={nav}>
      <PageHeader title={t('tripSettings.title')} back={base} />
      <div className="space-y-4">
        <Card>
          {isOwner ? (
            <form onSubmit={onSave} className="space-y-4">
              <Field label={t('tripSettings.name')}>
                <TextInput value={name} onChange={(e) => setName(e.target.value)} required maxLength={100} />
              </Field>
              <Field
                label={t('tripSettings.currency')}
                hint={
                  currencyLocked && expensesQ.data ? (
                    <span className="flex items-center gap-1 text-amber-700">
                      <Lock className="size-3" /> {t('tripSettings.locked')}
                    </span>
                  ) : (
                    t('tripSettings.changeable')
                  )
                }
              >
                <select value={currency} onChange={(e) => setCurrency(e.target.value)} disabled={currencyLocked} className={`${inputClass} disabled:bg-slate-100 disabled:text-slate-500`}>
                  {CURRENCIES.map((c) => (
                    <option key={c} value={c}>
                      {c} · {currencyName(c, locale)}
                    </option>
                  ))}
                </select>
              </Field>
              <ErrorText error={update.error} />
              {saved && !dirty && <p className="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700">{t('tripSettings.saved')}</p>}
              <Button type="submit" className="w-full" disabled={!dirty || update.isPending}>
                {update.isPending ? t('common.saving') : t('common.save')}
              </Button>
            </form>
          ) : (
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between">
                <dt className="text-slate-500">{t('tripSettings.name')}</dt>
                <dd className="font-medium">{trip.name}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-500">{t('tripSettings.currency')}</dt>
                <dd className="font-medium">
                  {trip.base_currency} · {currencyName(trip.base_currency, locale)}
                </dd>
              </div>
              <p className="pt-2 text-xs text-slate-500">{t('tripSettings.ownerOnly')}</p>
            </dl>
          )}
        </Card>

        <Card>
          <form onSubmit={onSavePromptPay} className="space-y-3">
            <Field label={t('tripSettings.myPromptPay')} hint={t('tripSettings.myPromptPayHint')}>
              <TextInput value={promptpay} onChange={(e) => setPromptpay(e.target.value)} inputMode="numeric" placeholder="08x-xxx-xxxx" maxLength={20} />
            </Field>
            <ErrorText error={updateMe.error} />
            {ppSaved && promptpay.replace(/\D/g, '') === (me.promptpay ?? '') && (
              <p className="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700">{t('tripSettings.saved')}</p>
            )}
            <Button type="submit" variant="secondary" className="w-full" disabled={updateMe.isPending || promptpay.replace(/\D/g, '') === (me.promptpay ?? '')}>
              {updateMe.isPending ? t('common.saving') : t('common.save')}
            </Button>
          </form>
        </Card>

        <Card>
          <h2 className="mb-1 font-bold">{t('tripSettings.inviteCode')}</h2>
          <p className="font-mono text-2xl tracking-widest text-brand-700">{trip.invite_code}</p>
          <p className="mt-1 text-xs text-slate-500">{t('tripSettings.inviteHint')}</p>
        </Card>

        <Card>
          <h2 className="mb-2 font-bold">{t('tripSettings.members', { n: members.length })}</h2>
          <ul className="divide-y divide-slate-100">
            {members.map((m) => (
              <li key={m.id} className="flex items-center gap-3 py-2">
                <Avatar name={m.name} size="sm" />
                <span className="flex-1">{m.name}</span>
                {m.id === trip.owner_member_id && (
                  <span className="flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-800">
                    <Crown className="size-3" /> {t('tripSettings.owner')}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </Card>

        <ActivityCard />
      </div>
    </Page>
  )
}

/** Who added, edited or deleted what, so changes to the money are visible. */
function ActivityCard() {
  const { t, locale } = useI18n()
  const { trip } = useTrip()
  const q = useActivity(trip.id)
  const rows = q.data ?? []
  return (
    <Card>
      <h2 className="mb-1 font-bold">{t('activity.title')}</h2>
      <p className="mb-2 text-xs text-slate-500">{t('activity.hint')}</p>
      {rows.length === 0 ? (
        <p className="text-sm text-slate-500">{t('activity.empty')}</p>
      ) : (
        <ul className="divide-y divide-slate-100">
          {rows.map((a) => (
            <li key={a.id} className="py-2 text-sm">
              <p>
                <b>{a.member_name ?? '?'}</b> {t(`activity.${a.action}`)} <span className="font-medium">“{a.title}”</span>{' '}
                <span className="tabular-nums text-slate-600">{formatMoney(a.amount_base, trip.base_currency)}</span>
              </p>
              <p className="text-xs text-slate-500">{formatShortDateTime(a.created_at, locale)}</p>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}
