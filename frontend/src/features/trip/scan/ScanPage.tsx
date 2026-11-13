import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { ApiError } from '../../../api/client'
import { uploadSlip } from '../../../api/endpoints'
import type { SlipOutcome } from '../../../api/types'
import { Button, Card, Page, PageHeader } from '../../../components/ui/ui'
import { MAX_UPLOAD_BYTES, mapLimit, resizeImage } from './image'
import { formatMoney } from '../../../lib/money'
import { useTrip } from '../layout/context'
import { useI18n, type I18n } from '../../../i18n/context'
import { ExpenseForm, type ExpenseDraft } from '../expenses/ExpenseFormPage'

type Status = 'waiting' | 'resizing' | 'reading' | 'parsed' | 'failed'
type Item = { key: string; file: File; preview: string; status: Status; outcome?: SlipOutcome; error?: string }
type Decision = 'saved' | 'skipped'

const UPLOAD_CONCURRENCY = 3

export function ScanPage() {
  const i18n = useI18n()
  const { t } = i18n
  const { trip, base, nav } = useTrip()
  const navigate = useNavigate()
  const [items, setItems] = useState<Item[]>([])
  const [stage, setStage] = useState<'pick' | 'review' | 'done'>('pick')
  const [index, setIndex] = useState(0)
  const [decisions, setDecisions] = useState<Record<string, Decision>>({})
  const inputRef = useRef<HTMLInputElement>(null)

  // Object URLs hold the image in memory; release them when leaving the page.
  const previews = useRef<string[]>([])
  useEffect(() => () => previews.current.forEach((u) => URL.revokeObjectURL(u)), [])

  const update = (key: string, patch: Partial<Item>) =>
    setItems((prev) => prev.map((it) => (it.key === key ? { ...it, ...patch } : it)))

  const onPick = async (e: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (!files.length) return
    const added: Item[] = files.map((file, i) => {
      const preview = URL.createObjectURL(file)
      previews.current.push(preview)
      return { key: `${Date.now()}-${i}-${file.name}`, file, preview, status: 'waiting' }
    })
    setItems((prev) => [...prev, ...added])

    await mapLimit(added, UPLOAD_CONCURRENCY, async (it) => {
      update(it.key, { status: 'resizing' })
      const blob = await resizeImage(it.file)
      if (blob.size > MAX_UPLOAD_BYTES) {
        update(it.key, { status: 'failed', error: t('scan.tooBig') })
        return
      }
      update(it.key, { status: 'reading' })
      try {
        const name = blob === it.file ? it.file.name : it.file.name.replace(/\.\w+$/, '') + '.jpg'
        const outcome = await uploadSlip(trip.id, blob, name)
        update(it.key, { status: outcome.status, outcome, error: slipError(outcome, i18n) })
      } catch (err) {
        update(it.key, {
          status: 'failed',
          error: err instanceof ApiError ? (i18n.tryT(`errors.${err.code}`) ?? err.message) : t('scan.uploadFailed'),
        })
      }
    })
  }

  const busy = items.some((it) => it.status === 'waiting' || it.status === 'resizing' || it.status === 'reading')
  const reviewable = items.filter((it) => it.status === 'parsed' || it.status === 'failed')

  const decide = (key: string, d: Decision) => {
    setDecisions((prev) => ({ ...prev, [key]: d }))
    if (index + 1 < reviewable.length) setIndex(index + 1)
    else setStage('done')
    window.scrollTo(0, 0)
  }

  if (stage === 'review') {
    const it = reviewable[index]
    return (
      <ExpenseForm
        key={it.key}
        heading={t('scan.reviewTitle', { i: index + 1, n: reviewable.length })}
        draft={draftFrom(it.outcome, trip.base_currency, i18n)}
        slipId={it.outcome?.slip_id ?? undefined}
        top={<SlipSummary item={it} baseCurrency={trip.base_currency} />}
        onSaved={() => decide(it.key, 'saved')}
        extraActions={
          <Button type="button" variant="ghost" className="w-full" onClick={() => decide(it.key, 'skipped')}>
            {t('scan.skip')}
          </Button>
        }
      />
    )
  }

  if (stage === 'done') {
    const saved = Object.values(decisions).filter((d) => d === 'saved').length
    return (
      <Page>
        <PageHeader title={t('scan.title')} back={base} />
        <Card className="space-y-4 text-center">
          <p className="text-4xl">✅</p>
          <p className="text-lg font-medium">{t('scan.done', { saved, n: reviewable.length })}</p>
          <Button className="w-full" onClick={() => navigate(base)}>
            {t('scan.backToTrip')}
          </Button>
        </Card>
      </Page>
    )
  }

  return (
    <Page nav={nav}>
      <PageHeader title={t('scan.title')} back={base} />
      <div className="space-y-4">
        <Card className="space-y-3 text-center">
          <p className="text-slate-600">{t('scan.intro')}</p>
          <input ref={inputRef} type="file" accept="image/*" multiple onChange={onPick} className="hidden" />
          <Button className="w-full" onClick={() => inputRef.current?.click()}>
            {t('scan.pick')}
          </Button>
          <p className="text-xs text-slate-500">{t('scan.privacy')}</p>
        </Card>

        {items.length > 0 && (
          <Card className="px-2">
            <ul className="divide-y divide-slate-100">
              {items.map((it) => (
                <li key={it.key} className="flex items-center gap-3 px-2 py-2">
                  <img src={it.preview} alt="" className="size-14 shrink-0 rounded-lg bg-slate-100 object-cover" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm">{(it.outcome && suggestedTitle(it.outcome, i18n)) || it.file.name}</p>
                    <StatusLine item={it} baseCurrency={trip.base_currency} />
                  </div>
                </li>
              ))}
            </ul>
          </Card>
        )}

        {items.length > 0 && (
          <Button className="w-full" disabled={busy || reviewable.length === 0} onClick={() => (setIndex(0), setStage('review'))}>
            {busy ? t('scan.reading') : t('scan.review', { n: reviewable.length })}
          </Button>
        )}
        <p className="text-center text-sm">
          <Link to={`${base}/expenses/new`} className="text-brand-700 underline">
            {t('scan.manual')}
          </Link>
        </p>
      </div>
    </Page>
  )
}

function StatusLine({ item, baseCurrency }: { item: Item; baseCurrency: string }) {
  const { t } = useI18n()
  const sug = item.outcome?.suggestion
  switch (item.status) {
    case 'waiting':
      return <p className="text-xs text-slate-500">{t('scan.queued')}</p>
    case 'resizing':
      return <p className="text-xs text-slate-500">{t('scan.resizing')}</p>
    case 'reading':
      return (
        <p className="flex items-center gap-1 text-xs text-brand-700">
          <span className="size-3 animate-spin rounded-full border-2 border-brand-600 border-t-transparent" /> {t('scan.aiReading')}
        </p>
      )
    case 'failed':
      return <p className="text-xs text-red-700">{t('scan.failedHint', { error: item.error ?? t('scan.readFailed') })}</p>
    case 'parsed':
      return (
        <p className="text-xs text-emerald-700">
          {t('scan.parsed')}{' '}
          {sug?.amount != null && sug.currency ? formatMoney(sug.amount, sug.currency) : t('scan.noAmount')}
          {sug?.currency && sug.currency !== baseCurrency && <span className="text-amber-700">{t('scan.currencyTag', { cur: sug.currency })}</span>}
          {sug?.low_confidence && <span className="text-amber-700">{t('scan.unsure')}</span>}
        </p>
      )
  }
}

function SlipSummary({ item, baseCurrency }: { item: Item; baseCurrency: string }) {
  const { t } = useI18n()
  const r = item.outcome?.result
  const sug = item.outcome?.suggestion
  const foreign = sug?.currency && sug.currency !== baseCurrency
  return (
    <div className="mb-4 space-y-2">
      <div className="flex gap-3 rounded-2xl bg-white p-3 ring-1 ring-slate-200">
        <a href={item.preview} target="_blank" rel="noreferrer" className="shrink-0">
          <img src={item.preview} alt={t('scan.slipAlt')} className="h-28 w-20 rounded-lg bg-slate-100 object-cover" />
        </a>
        <dl className="min-w-0 flex-1 space-y-0.5 text-xs text-slate-600">
          {r ? (
            <>
              <Row
                k={t('scan.kind')}
                v={{ transfer_slip: t('scan.kindTransfer'), receipt: t('scan.kindReceipt'), unknown: t('scan.kindUnknown') }[r.kind]}
              />
              <Row k={t('scan.amount')} v={r.amount && `${r.amount} ${r.currency}`} />
              <Row k={t('scan.shop')} v={r.merchant} />
              <Row k={t('scan.from')} v={r.from_name} />
              <Row k={t('scan.to')} v={r.to_name} />
              <Row k={t('scan.reference')} v={r.reference} />
              <Row k={t('scan.confidence')} v={`${Math.round(r.confidence * 100)}%`} />
            </>
          ) : (
            <p className="text-red-700">{item.error ?? t('scan.aiFailed')}</p>
          )}
        </dl>
      </div>
      {foreign && (
        <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
          {t('scan.foreign', { cur: sug.currency, base: baseCurrency })}
        </p>
      )}
      {sug?.low_confidence && (
        <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">{t('scan.lowConfidence')}</p>
      )}
    </div>
  )
}

function Row({ k, v }: { k: string; v?: string }) {
  if (!v) return null
  return (
    <div className="flex gap-2">
      <dt className="shrink-0 text-slate-400">{k}</dt>
      <dd className="truncate">{v}</dd>
    </div>
  )
}

/** Title for the expense form, worded in the UI language. */
function suggestedTitle(o: SlipOutcome, i18n: I18n): string {
  const s = o.suggestion
  if (!s) return ''
  if (s.title_kind === 'transfer_to' && s.title) return i18n.t('scan.transferTo', { name: s.title })
  return s.title || i18n.t('scan.defaultTitle')
}

function slipError(o: SlipOutcome, i18n: I18n): string | undefined {
  if (!o.error && !o.error_code) return undefined
  return (o.error_code && i18n.tryT(`slipErrors.${o.error_code}`)) || o.error
}

function draftFrom(o: SlipOutcome | undefined, baseCurrency: string, i18n: I18n): ExpenseDraft {
  const s = o?.suggestion
  if (!s || !o) return {}
  return {
    title: suggestedTitle(o, i18n),
    amount: s.amount,
    currency: s.currency || baseCurrency,
    items: s.items,
    spentAt: s.spent_at,
    category: s.category,
  }
}
