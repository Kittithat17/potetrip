import { Plus, X } from 'lucide-react'
import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router'
import { useDeleteExpense, useExpense, useExpenseRate, useSaveExpense } from '../../../api/hooks'
import type { Category, Expense, ExpenseInput, Member } from '../../../api/types'
import { Approx } from '../../../components/money/Approx'
import { Avatar, Button, ErrorText, Field, inputBase, inputClass, Page, PageHeader, Spinner, TextInput } from '../../../components/ui/ui'
import { useI18n } from '../../../i18n/context'
import { CATEGORIES } from './categories'
import { useConfirm } from '../../../components/feedback/confirm'
import { currencyName } from '../../../lib/currencyInfo'
import { formatShortDate, fromLocalInput, toLocalInput } from '../../../lib/datetime'
import { CURRENCIES, formatAmount, formatMoney, parseAmount, splitEqual } from '../../../lib/money'
import { convertMinor, parseRate, splitItems } from './split'
import { useTrip } from '../layout/context'

export function ExpenseFormPage() {
  const { t } = useI18n()
  const { id } = useParams()
  const { trip, base } = useTrip()
  const q = useExpense(trip.id, id)

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

type SplitMode = 'equal' | 'exact' | 'items'

/** Prefilled values for a new expense, e.g. from a scanned slip. */
export type ExpenseDraft = {
  title?: string
  amount?: number | null
  currency?: string
  spentAt?: string | null
  category?: Category
  items?: { name: string; amount: number }[]
}

type ItemRow = { key: number; name: string; amount: string; members: Set<string> }

type FormProps = {
  existing?: Expense
  draft?: ExpenseDraft
  slipId?: string
  heading?: string
  top?: ReactNode // shown above the fields
  extraActions?: ReactNode // shown under the save button
  onSaved?: () => void // defaults to returning to the dashboard
}

let nextKey = 1

export function ExpenseForm({ existing, draft, slipId, heading, top, extraActions, onSaved }: FormProps) {
  const { t, locale } = useI18n()
  const confirm = useConfirm()
  const { trip, me, members, base } = useTrip()
  const navigate = useNavigate()
  const baseCur = trip.base_currency
  // Same rule as the server: who logged it, who paid it, or the trip owner.
  const readOnly = !!existing && me.id !== existing.created_by
  const save = useSaveExpense(trip.id, existing?.id)
  const del = useDeleteExpense(trip.id)

  const initialCur = existing?.currency ?? (draft?.currency && CURRENCIES.includes(draft.currency) ? draft.currency : baseCur)
  const [cur, setCur] = useState(initialCur)
  const [title, setTitle] = useState(existing?.title ?? draft?.title ?? '')
  const [amountText, setAmountText] = useState(() => {
    const v = existing?.amount ?? draft?.amount
    return v ? formatAmount(v, initialCur, false) : ''
  })
  const [paidBy, setPaidBy] = useState(existing?.paid_by ?? me.id)
  const [category, setCategory] = useState<Category>(existing?.category ?? draft?.category ?? 'food')
  const [spentAt, setSpentAt] = useState(() => toLocalInput(existing?.spent_at ?? draft?.spentAt ?? new Date()))
  // Editing keeps the rate the expense was saved with unless the user switches back.
  const [ownRate, setOwnRate] = useState(existing && existing.currency !== baseCur ? existing.fx_rate : '')
  const [useOwnRate, setUseOwnRate] = useState(!!existing && existing.currency !== baseCur)

  const [mode, setMode] = useState<SplitMode>(existing?.split_type ?? (draft?.items?.length ? 'items' : 'equal'))
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(existing ? existing.shares.map((s) => s.member_id) : members.map((m) => m.id)),
  )
  // Exact shares are entered in the expense currency; saved ones are in the base
  // currency, so they only prefill when the two match.
  const [exact, setExact] = useState<Record<string, string>>(() =>
    existing?.split_type === 'exact' && existing.currency === baseCur
      ? Object.fromEntries(existing.shares.map((s) => [s.member_id, formatAmount(s.amount_base, baseCur, false)]))
      : {},
  )
  const [items, setItems] = useState<ItemRow[]>(() => {
    const everyone = () => new Set(members.map((m) => m.id))
    if (existing?.items.length) {
      return existing.items.map((it) => ({
        key: nextKey++,
        name: it.name,
        amount: formatAmount(it.amount, existing.currency, false),
        members: new Set(it.member_ids),
      }))
    }
    if (draft?.items?.length) {
      return draft.items.map((it) => ({ key: nextKey++, name: it.name, amount: formatAmount(it.amount, initialCur, false), members: everyone() }))
    }
    return [{ key: nextKey++, name: '', amount: '', members: everyone() }]
  })
  const [formError, setFormError] = useState('')

  const foreign = cur !== baseCur
  const amount = parseAmount(amountText, cur)

  // The reference rate is for the expense's day (UTC, like the server); future dates use the latest.
  const spentIso = spentAt ? fromLocalInput(spentAt) : undefined
  const [today] = useState(() => new Date().toISOString().slice(0, 10))
  const rateDay = spentIso && spentIso.slice(0, 10) < today ? spentIso.slice(0, 10) : undefined
  const refRate = useExpenseRate(cur, baseCur, foreign ? rateDay : undefined)
  const noRefRate = foreign && !!refRate.error
  const manual = foreign && (useOwnRate || noRefRate)
  const rate = !foreign ? 1 : manual ? parseRate(ownRate) : refRate.data ? Number(refRate.data.rate) : null
  const amountBase = amount && rate ? convertMinor(amount, cur, baseCur, rate) : null

  const selectedOrdered = members.filter((m) => selected.has(m.id))
  const equalParts = amount ? splitEqual(amount, selectedOrdered.length) : []
  const exactParsed = members.map((m) => ({ id: m.id, v: exact[m.id]?.trim() ? parseAmount(exact[m.id], cur) : 0 }))
  const exactSum = exactParsed.reduce((s, x) => s + (x.v ?? 0), 0)
  const exactRemaining = (amount ?? 0) - exactSum

  const itemsParsed = items.map((it) => ({ ...it, value: parseAmount(it.amount, cur) }))
  const itemsTotal = itemsParsed.reduce((s, it) => s + (it.value ?? 0), 0)
  const itemsValid = itemsParsed.every((it) => it.name.trim() && it.value && it.members.size > 0)
  const itemShares =
    amount && itemsValid
      ? splitItems(
          amount,
          itemsParsed.map((it) => ({ amount: it.value!, memberIds: [...it.members] })),
          members.map((m) => m.id),
        )
      : null

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const updateItem = (key: number, patch: Partial<ItemRow>) => setItems((rows) => rows.map((r) => (r.key === key ? { ...r, ...patch } : r)))
  const toggleItemMember = (key: number, id: string) =>
    setItems((rows) =>
      rows.map((r) => {
        if (r.key !== key) return r
        const next = new Set(r.members)
        if (next.has(id)) next.delete(id)
        else next.add(id)
        return { ...r, members: next }
      }),
    )

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFormError('')
    if (!amount) return setFormError(t('expense.errAmount'))
    if (manual && !parseRate(ownRate)) return setFormError(t('expense.errRate'))

    const body: ExpenseInput = {
      title,
      category,
      paid_by: paidBy,
      amount,
      currency: cur,
      split_type: mode,
      spent_at: spentIso,
      slip_id: existing ? undefined : slipId,
      fx_rate: manual ? ownRate.trim() : undefined,
    }
    if (mode === 'equal') {
      if (selected.size === 0) return setFormError(t('expense.errNoOne'))
      body.member_ids = selectedOrdered.map((m) => m.id)
    } else if (mode === 'exact') {
      if (exactParsed.some((x) => x.v === null)) return setFormError(t('expense.errBadShare'))
      if (exactRemaining !== 0) return setFormError(t('expense.errSum'))
      body.shares = exactParsed.filter((x) => (x.v ?? 0) > 0).map((x) => ({ member_id: x.id, amount: x.v! }))
    } else {
      if (!itemsValid) return setFormError(t('expense.errItems'))
      body.items = itemsParsed.map((it) => ({
        name: it.name.trim(),
        amount: it.value!,
        member_ids: members.filter((m) => it.members.has(m.id)).map((m) => m.id),
      }))
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
        {readOnly && <p className="rounded-xl bg-slate-100 px-3 py-2 text-sm text-slate-600">{t('expense.readOnly')}</p>}
        <fieldset disabled={readOnly} className="min-w-0 space-y-4">
        <Field label={t('expense.title')}>
          <TextInput value={title} onChange={(e) => setTitle(e.target.value)} placeholder={t('expense.titlePlaceholder')} required maxLength={200} />
        </Field>

        <Field
          label={t('expense.amount')}
          hint={!foreign && amount ? <Approx minor={amount} currency={cur} className="text-xs text-slate-500" /> : undefined}
        >
          <div className="flex gap-2">
            <TextInput
              value={amountText}
              onChange={(e) => setAmountText(e.target.value)}
              inputMode="decimal"
              placeholder="0.00"
              required
              className="text-lg font-semibold tabular-nums"
            />
            <select
              value={cur}
              onChange={(e) => setCur(e.target.value)}
              aria-label={t('expense.currency')}
              className={`${inputBase} w-24 shrink-0 bg-slate-100 font-medium`}
            >
              {[baseCur, ...CURRENCIES.filter((c) => c !== baseCur)].map((c) => (
                // The closed select is narrow, so options show the code; the name is in the tooltip.
                <option key={c} value={c} title={currencyName(c, locale)}>
                  {c}
                </option>
              ))}
            </select>
          </div>
        </Field>

        {foreign && (
          <div className="space-y-2 rounded-2xl bg-brand-50 p-3 text-sm ring-1 ring-brand-100">
            {manual ? (
              <>
                <label className="block text-slate-700">
                  {t('expense.rateLabel', { from: cur, to: baseCur })}
                  <input
                    value={ownRate}
                    onChange={(e) => setOwnRate(e.target.value)}
                    inputMode="decimal"
                    placeholder={refRate.data?.rate ?? '0.0000'}
                    className={`${inputClass} mt-1 tabular-nums`}
                  />
                </label>
                {noRefRate ? (
                  <p className="text-amber-800">{t('expense.noRate', { cur })}</p>
                ) : (
                  <button type="button" onClick={() => setUseOwnRate(false)} className="text-brand-700 underline">
                    {t('expense.useRefRate')}
                  </button>
                )}
              </>
            ) : refRate.data ? (
              <>
                <p className="text-slate-700 tabular-nums">
                  1 {cur} = {refRate.data.rate} {baseCur}
                  <span className="block text-xs text-slate-500">
                    {t('expense.rateRef', { date: formatShortDate(refRate.data.rate_date, locale) })}
                    {refRate.data.stale && t('expense.rateStale')}
                  </span>
                </p>
                <button
                  type="button"
                  onClick={() => {
                    setOwnRate(refRate.data.rate)
                    setUseOwnRate(true)
                  }}
                  className="text-brand-700 underline"
                >
                  {t('expense.ownRate')}
                </button>
              </>
            ) : (
              <p className="text-slate-500">{t('expense.rateLoading')}</p>
            )}
            {amountBase !== null && (
              <p className="border-t border-brand-100 pt-2 font-semibold tabular-nums">
                {t('expense.inBase', { amount: formatMoney(amountBase, baseCur) })}
              </p>
            )}
          </div>
        )}

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
                ['items', t('expense.items')],
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
          </div>

          {mode === 'equal' && (
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
          )}

          {mode === 'exact' && (
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

          {mode === 'items' && (
            <ItemsEditor
              items={itemsParsed}
              members={members}
              cur={cur}
              amount={amount}
              itemsTotal={itemsTotal}
              shares={itemShares}
              onChange={updateItem}
              onToggle={toggleItemMember}
              onRemove={(key) => setItems((rows) => (rows.length > 1 ? rows.filter((r) => r.key !== key) : rows))}
              onAdd={() => setItems((rows) => [...rows, { key: nextKey++, name: '', amount: '', members: new Set(members.map((m) => m.id)) }])}
              onUseTotal={() => setAmountText(formatAmount(itemsTotal, cur, false))}
            />
          )}
        </fieldset>

        </fieldset>

        <ErrorText error={formError || save.error || del.error} />

        <div className="space-y-2 pt-2">
          {!readOnly && (
          <Button type="submit" className="w-full" disabled={save.isPending}>
            {save.isPending ? t('common.saving') : t('common.save')}
          </Button>
          )}
          {existing && !readOnly && (
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

type ItemsEditorProps = {
  items: (ItemRow & { value: number | null })[]
  members: Member[]
  cur: string
  amount: number | null
  itemsTotal: number
  shares: Map<string, number> | null
  onChange: (key: number, patch: Partial<ItemRow>) => void
  onToggle: (key: number, memberId: string) => void
  onRemove: (key: number) => void
  onAdd: () => void
  onUseTotal: () => void
}

function ItemsEditor({ items, members, cur, amount, itemsTotal, shares, onChange, onToggle, onRemove, onAdd, onUseTotal }: ItemsEditorProps) {
  const { t } = useI18n()
  const diff = amount !== null ? amount - itemsTotal : 0
  return (
    <div className="space-y-3">
      <ul className="space-y-2">
        {items.map((it, i) => (
          <li key={it.key} className="space-y-2 rounded-2xl bg-white p-3 ring-1 ring-slate-200">
            <div className="flex gap-2">
              <input
                value={it.name}
                onChange={(e) => onChange(it.key, { name: e.target.value })}
                placeholder={t('expense.itemName', { n: i + 1 })}
                aria-label={t('expense.itemName', { n: i + 1 })}
                maxLength={100}
                className={`${inputBase} min-w-0 flex-1`}
              />
              <input
                value={it.amount}
                onChange={(e) => onChange(it.key, { amount: e.target.value })}
                inputMode="decimal"
                placeholder="0.00"
                aria-label={t('expense.itemPrice', { n: i + 1 })}
                className={`${inputBase} w-24 shrink-0 text-right tabular-nums ${it.amount && it.value === null ? 'ring-red-400' : ''}`}
              />
              <button
                type="button"
                onClick={() => onRemove(it.key)}
                disabled={items.length === 1}
                aria-label={t('expense.removeItem')}
                className="shrink-0 rounded-lg px-1 text-slate-400 hover:text-red-600 disabled:opacity-30"
              >
                <X className="size-5" />
              </button>
            </div>
            <div className="flex flex-wrap gap-1.5" role="group" aria-label={t('expense.whoHad')}>
              {members.map((m) => {
                const on = it.members.has(m.id)
                return (
                  <button
                    type="button"
                    key={m.id}
                    onClick={() => onToggle(it.key, m.id)}
                    aria-pressed={on}
                    className={`rounded-full px-3 py-1 text-sm ring-1 ${on ? 'bg-brand-600 text-white ring-brand-600' : 'bg-white text-slate-500 ring-slate-300'}`}
                  >
                    {m.name}
                  </button>
                )
              })}
            </div>
          </li>
        ))}
      </ul>
      <button type="button" onClick={onAdd} className="flex items-center gap-1 text-sm font-medium text-brand-700">
        <Plus className="size-4" /> {t('expense.addItem')}
      </button>

      <div className="space-y-1 rounded-xl bg-slate-50 p-3 text-sm">
        <p className="flex justify-between tabular-nums">
          <span className="text-slate-600">{t('expense.itemsTotal')}</span>
          <span>{formatMoney(itemsTotal, cur)}</span>
        </p>
        {amount !== null && diff !== 0 && (
          <>
            <p className="text-xs text-slate-500">
              {t(diff > 0 ? 'expense.itemsExtra' : 'expense.itemsDiscount', { amount: formatMoney(Math.abs(diff), cur) })}
            </p>
            <button type="button" onClick={onUseTotal} className="text-xs text-brand-700 underline">
              {t('expense.useItemsTotal')}
            </button>
          </>
        )}
        {amount === null && itemsTotal > 0 && (
          <button type="button" onClick={onUseTotal} className="text-xs text-brand-700 underline">
            {t('expense.useItemsTotal')}
          </button>
        )}
      </div>

      {shares && (
        <ul className="divide-y divide-slate-100 rounded-2xl bg-white ring-1 ring-slate-200">
          {members
            .filter((m) => (shares.get(m.id) ?? 0) > 0)
            .map((m) => (
              <li key={m.id} className="flex items-center gap-3 px-3 py-2">
                <Avatar name={m.name} size="sm" />
                <span className="flex-1">{m.name}</span>
                <span className="text-sm font-medium tabular-nums">{formatMoney(shares.get(m.id)!, cur)}</span>
              </li>
            ))}
        </ul>
      )}
    </div>
  )
}
