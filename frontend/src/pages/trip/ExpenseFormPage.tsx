import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router'
import { useDeleteExpense, useExpense, useSaveExpense } from '../../api/hooks'
import type { Category, Expense, ExpenseInput } from '../../api/types'
import { Avatar, Button, ErrorText, Field, inputBase, inputClass, Page, PageHeader, Spinner, TextInput } from '../../components/ui'
import { CATEGORIES } from '../../lib/categories'
import { fromLocalInput, toLocalInput } from '../../lib/datetime'
import { formatAmount, formatMoney, parseAmount, splitEqual } from '../../lib/money'
import { useTrip } from './context'
import { useI18n } from '../../i18n/context'
import { useConfirm } from '../../lib/confirm'
import { Approx } from '../../components/Approx'

export function ExpenseFormPage() {
  const { t } = useI18n()
  const { id } = useParams()
  const { trip, token, base } = useTrip()
  const q = useExpense(trip.id, token, id)

  if (id && q.isPending) {
    return (
      <Page>
        <Spinner />
      </Page>
    )
  }
  if (id && q.error) {
    return (
      <Page>
        <PageHeader title={t('expense.edit')} back={base} />
        <ErrorText error={q.error} />
      </Page>
    )
  }
  // Re-mount the form once the expense loads so initial state is correct.
  return <ExpenseForm key={q.data?.id ?? 'new'} existing={q.data} />
}

type SplitMode = 'equal' | 'exact'

/** Prefilled values for a new expense, e.g. from a scanned slip. */
export type ExpenseDraft = { title?: string; amount?: number | null; spentAt?: string | null; category?: Category }

type FormProps = {
  existing?: Expense
  draft?: ExpenseDraft
  slipId?: string
  heading?: string
  top?: ReactNode // shown above the fields
  extraActions?: ReactNode // shown under the save button
  onSaved?: () => void // defaults to returning to the dashboard
}

export function ExpenseForm({ existing, draft, slipId, heading, top, extraActions, onSaved }: FormProps) {
  const { t } = useI18n()
  const confirm = useConfirm()
  const { trip, me, token, members, base } = useTrip()
  const navigate = useNavigate()
  const cur = trip.base_currency
  const save = useSaveExpense(trip.id, token, existing?.id)
  const del = useDeleteExpense(trip.id, token)

  const [title, setTitle] = useState(existing?.title ?? draft?.title ?? '')
  const [amountText, setAmountText] = useState(
    existing
      ? formatAmount(existing.amount, existing.currency, false)
      : draft?.amount
        ? formatAmount(draft.amount, cur, false)
        : '',
  )
  const [paidBy, setPaidBy] = useState(existing?.paid_by ?? me.id)
  const [category, setCategory] = useState<Category>(existing?.category ?? draft?.category ?? 'food')
  const [spentAt, setSpentAt] = useState(() => toLocalInput(existing?.spent_at ?? draft?.spentAt ?? new Date()))
  const [mode, setMode] = useState<SplitMode>(existing?.split_type === 'exact' ? 'exact' : 'equal')
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(existing ? existing.shares.map((s) => s.member_id) : members.map((m) => m.id)),
  )
  const [exact, setExact] = useState<Record<string, string>>(() =>
    Object.fromEntries(
      (existing?.split_type === 'exact' ? existing.shares : []).map((s) => [s.member_id, formatAmount(s.amount_base, cur, false)]),
    ),
  )
  const [formError, setFormError] = useState('')

  const amount = parseAmount(amountText, cur)
  const selectedOrdered = members.filter((m) => selected.has(m.id))
  const equalParts = amount ? splitEqual(amount, selectedOrdered.length) : []
  const exactParsed = members.map((m) => ({ id: m.id, v: exact[m.id]?.trim() ? parseAmount(exact[m.id], cur) : 0 }))
  const exactSum = exactParsed.reduce((s, x) => s + (x.v ?? 0), 0)
  const exactRemaining = (amount ?? 0) - exactSum

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    if (!amount) return setFormError(t('expense.errAmount'))

    const body: ExpenseInput = {
      title,
      category,
      paid_by: paidBy,
      amount,
      currency: cur,
      split_type: mode,
      spent_at: fromLocalInput(spentAt),
      slip_id: existing ? undefined : slipId,
    }
    if (mode === 'equal') {
      if (selected.size === 0) return setFormError(t('expense.errNoOne'))
      body.member_ids = selectedOrdered.map((m) => m.id)
    } else {
      if (exactParsed.some((x) => x.v === null)) return setFormError(t('expense.errBadShare'))
      if (exactRemaining !== 0) return setFormError(t('expense.errSum'))
      body.shares = exactParsed.filter((x) => (x.v ?? 0) > 0).map((x) => ({ member_id: x.id, amount: x.v! }))
    }
    save.mutate(body, { onSuccess: () => (onSaved ? onSaved() : navigate(base)) })
  }

  const onDelete = async () => {
    if (!existing) return
    const ok = await confirm({
      title: t('expense.deleteConfirm', { title: existing.title }),
      message: t('expense.deleteHint'),
      confirmLabel: t('expense.deleteAction'),
    })
    if (!ok) return
    del.mutate(existing.id, { onSuccess: () => navigate(base) })
  }

  return (
    <Page>
      <PageHeader title={heading ?? (existing ? t('expense.edit') : t('expense.add'))} back={base} />
      {top}
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label={t('expense.title')}>
          <TextInput value={title} onChange={(e) => setTitle(e.target.value)} placeholder={t('expense.titlePlaceholder')} required maxLength={200} />
        </Field>

        <Field label={t('expense.amount')} hint={amount ? <Approx minor={amount} currency={cur} className="text-xs text-slate-500" /> : undefined}>
          <div className="flex gap-2">
            <TextInput
              value={amountText}
              onChange={(e) => setAmountText(e.target.value)}
              inputMode="decimal"
              placeholder="0.00"
              required
              className="text-lg font-semibold tabular-nums"
            />
            <span className="flex min-h-11 w-20 shrink-0 items-center justify-center rounded-xl bg-slate-200 font-medium text-slate-600">{cur}</span>
          </div>
        </Field>

        <Field label={t('expense.paidBy')}>
          <select value={paidBy} onChange={(e) => setPaidBy(e.target.value)} className={inputClass}>
            {members.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
                {m.id === me.id ? t('common.meSuffix') : ''}
              </option>
            ))}
          </select>
        </Field>

        <fieldset>
          <legend className="mb-1 text-sm font-medium text-slate-700">{t('expense.category')}</legend>
          <div className="grid grid-cols-3 gap-2">
            {CATEGORIES.map((c) => (
              <button
                type="button"
                key={c.value}
                onClick={() => setCategory(c.value)}
                aria-pressed={category === c.value}
                className={`rounded-xl px-2 py-2 text-sm ring-1 ${
                  category === c.value ? 'bg-brand-700 text-white ring-brand-700' : 'bg-white ring-slate-300'
                }`}
              >
                <c.icon className="mx-auto mb-0.5 size-5" />
                {t(c.labelKey)}
              </button>
            ))}
          </div>
        </fieldset>

        <Field label={t('expense.when')}>
          <TextInput type="datetime-local" value={spentAt} onChange={(e) => setSpentAt(e.target.value)} required />
        </Field>

        <fieldset className="space-y-3">
          <legend className="mb-1 text-sm font-medium text-slate-700">{t('expense.split')}</legend>
          <div className="grid grid-cols-3 rounded-xl bg-slate-200 p-1 text-sm">
            {(
              [
                ['equal', t('expense.equal')],
                ['exact', t('expense.exact')],
              ] as const
            ).map(([v, label]) => (
              <button
                type="button"
                key={v}
                onClick={() => setMode(v)}
                aria-pressed={mode === v}
                className={`rounded-lg py-2 ${mode === v ? 'bg-white font-medium shadow-sm' : 'text-slate-600'}`}
              >
                {label}
              </button>
            ))}
            <button type="button" disabled className="rounded-lg py-2 text-slate-400" title={t('expense.comingSoon')}>
              {t('expense.items')}
            </button>
          </div>

          {mode === 'equal' ? (
            <ul className="divide-y divide-slate-100 rounded-2xl bg-white ring-1 ring-slate-200">
              {members.map((m) => {
                const idx = selectedOrdered.findIndex((x) => x.id === m.id)
                return (
                  <li key={m.id}>
                    <label className="flex items-center gap-3 px-3 py-2">
                      <input type="checkbox" checked={selected.has(m.id)} onChange={() => toggle(m.id)} className="size-5 accent-brand-700" />
                      <Avatar name={m.name} size="sm" />
                      <span className="flex-1">{m.name}</span>
                      <span className="text-sm text-slate-500 tabular-nums">{idx >= 0 && amount ? formatMoney(equalParts[idx], cur) : ''}</span>
                    </label>
                  </li>
                )
              })}
            </ul>
          ) : (
            <>
              <ul className="divide-y divide-slate-100 rounded-2xl bg-white ring-1 ring-slate-200">
                {members.map((m) => (
                  <li key={m.id} className="flex items-center gap-3 px-3 py-2">
                    <Avatar name={m.name} size="sm" />
                    <span className="flex-1">{m.name}</span>
                    <input
                      value={exact[m.id] ?? ''}
                      onChange={(e) => setExact((p) => ({ ...p, [m.id]: e.target.value }))}
                      inputMode="decimal"
                      placeholder="0.00"
                      aria-label={t('expense.amountFor', { name: m.name })}
                      className={`${inputBase} w-28 shrink-0 text-right tabular-nums`}
                    />
                  </li>
                ))}
              </ul>
              <p className={`text-right text-sm tabular-nums ${exactRemaining === 0 ? 'text-emerald-700' : 'text-amber-700'}`}>
                {exactRemaining === 0
                  ? t('expense.adds')
                  : exactRemaining > 0
                    ? t('expense.remaining', { amount: formatMoney(exactRemaining, cur) })
                    : t('expense.over', { amount: formatMoney(-exactRemaining, cur) })}
              </p>
            </>
          )}
        </fieldset>

        <ErrorText error={formError || save.error || del.error} />

        <div className="space-y-2 pt-2">
          <Button type="submit" className="w-full" disabled={save.isPending}>
            {save.isPending ? t('common.saving') : t('common.save')}
          </Button>
          {existing && (
            <Button type="button" variant="danger" className="w-full" onClick={onDelete} disabled={del.isPending}>
              {t('expense.delete')}
            </Button>
          )}
          {extraActions}
        </div>
      </form>
    </Page>
  )
}
