import type { ReactNode } from 'react'
import { useSummary } from '../../api/hooks'
import { Avatar, Card, ErrorText, Page, PageHeader, Spinner } from '../../components/ui'
import { Approx } from '../../components/Approx'
import { useI18n } from '../../i18n/context'
import { categoryInfo } from '../../lib/categories'
import { currencyName } from '../../lib/currencyInfo'
import { formatMoney } from '../../lib/money'
import { useTrip } from './context'

// Two series on one scale: what someone paid for the group vs. their own share.
// Validated pair (dataviz validator, light surface): CVD ΔE 38, normal ΔE 46.
const PAID = '#2f4cf5' // brand-600
const CONSUMED = '#e8930c' // contrast < 3:1 on white, so every bar carries a printed value

export function SummaryPage() {
  const { t, locale } = useI18n()
  const { trip, members, memberName, base, nav } = useTrip()
  const q = useSummary(trip.id)
  const cur = trip.base_currency

  return (
    <Page nav={nav}>
      <PageHeader title={t('summary.title')} back={base} />
      {q.isPending ? (
        <Spinner />
      ) : q.error ? (
        <ErrorText error={q.error} />
      ) : q.data.count === 0 ? (
        <Card>
          <p className="py-8 text-center text-slate-600">{t('summary.empty')}</p>
        </Card>
      ) : (
        <div className="space-y-4">
          <Card>
            <p className="text-sm text-slate-500">{t('summary.total')}</p>
            <p className="text-4xl font-bold tabular-nums">{formatMoney(q.data.total, cur)}</p>
            <Approx minor={q.data.total} currency={cur} className="text-sm text-slate-500" />
            <div className="mt-3 grid grid-cols-2 gap-3 border-t border-slate-100 pt-3 text-sm">
              <div>
                <p className="text-slate-500">{t('summary.count')}</p>
                <p className="text-lg font-semibold tabular-nums">{q.data.count}</p>
              </div>
              <div>
                <p className="text-slate-500">{t('summary.perPerson')}</p>
                <p className="text-lg font-semibold tabular-nums">{formatMoney(Math.round(q.data.total / Math.max(members.length, 1)), cur)}</p>
              </div>
            </div>
          </Card>

          <Card>
            <h2 className="mb-3 font-bold">{t('summary.byCategory')}</h2>
            <BarList
              label={t('summary.byCategory')}
              rows={q.data.by_category.map((c) => {
                const info = categoryInfo(c.category)
                return {
                  key: c.category,
                  label: (
                    <span className="flex items-center gap-2">
                      <span className={`flex size-7 items-center justify-center rounded-lg ${info.tint}`}>
                        <info.icon className="size-4" />
                      </span>
                      {t(info.labelKey)}
                      <span className="text-xs text-slate-400">×{c.count}</span>
                    </span>
                  ),
                  value: c.amount_base,
                  text: `${formatMoney(c.amount_base, cur)} · ${Math.round((c.amount_base / q.data.total) * 100)}%`,
                }
              })}
            />
          </Card>

          <Card>
            <h2 className="font-bold">{t('summary.byMember')}</h2>
            <div className="mt-1 mb-3 flex gap-4 text-xs text-slate-600" aria-hidden>
              <span className="flex items-center gap-1.5">
                <span className="size-2.5 rounded-sm" style={{ background: PAID }} /> {t('summary.paid')}
              </span>
              <span className="flex items-center gap-1.5">
                <span className="size-2.5 rounded-sm" style={{ background: CONSUMED }} /> {t('summary.consumed')}
              </span>
            </div>
            <MemberBars rows={q.data.by_member} cur={cur} memberName={memberName} />
            <p className="mt-3 text-xs text-slate-500">{t('summary.memberHint')}</p>
          </Card>

          {q.data.by_currency.length > 1 && (
            <Card>
              <h2 className="mb-2 font-bold">{t('summary.byCurrency')}</h2>
              <ul className="divide-y divide-slate-100 text-sm">
                {q.data.by_currency.map((c) => (
                  <li key={c.currency} className="flex items-center justify-between gap-3 py-2">
                    <span>
                      <span className="font-medium">{c.currency}</span>{' '}
                      <span className="text-slate-500">{currencyName(c.currency, locale)}</span>
                      <span className="block text-xs text-slate-400">{c.count === 1 ? t('summary.itemsOne') : t('summary.items', { n: c.count })}</span>
                    </span>
                    <span className="text-right tabular-nums">
                      <span className="block font-semibold">{formatMoney(c.amount, c.currency)}</span>
                      {c.currency !== cur && <span className="block text-xs text-slate-500">= {formatMoney(c.amount_base, cur)}</span>}
                    </span>
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </div>
      )}
    </Page>
  )
}

type BarRow = { key: string; label: ReactNode; value: number; text: string }

/** Horizontal bars on one baseline; each carries its value, so the list doubles as the table. */
function BarList({ rows, label }: { rows: BarRow[]; label: string }) {
  const max = Math.max(...rows.map((r) => r.value), 1)
  return (
    <ul className="space-y-3" aria-label={label}>
      {rows.map((r) => (
        <li key={r.key}>
          <div className="mb-1 flex items-center justify-between gap-2 text-sm">
            {r.label}
            <span className="shrink-0 text-slate-700 tabular-nums">{r.text}</span>
          </div>
          <div className="h-3 w-full">
            <div className="h-full rounded-r-[4px] bg-brand-600" style={{ width: `${Math.max((r.value / max) * 100, 1)}%` }} />
          </div>
        </li>
      ))}
    </ul>
  )
}

function MemberBars({
  rows,
  cur,
  memberName,
}: {
  rows: { member_id: string; paid: number; consumed: number }[]
  cur: string
  memberName: (id: string) => string
}) {
  const { t } = useI18n()
  const max = Math.max(...rows.flatMap((r) => [r.paid, r.consumed]), 1)
  const bar = (v: number, color: string, name: string) => (
    <div className="flex items-center gap-2" title={`${name}: ${formatMoney(v, cur)}`}>
      <div className="h-2.5 min-w-0 flex-1">
        {v > 0 && <div className="h-full rounded-r-[4px]" style={{ width: `${Math.max((v / max) * 100, 1)}%`, background: color }} />}
      </div>
      <span className="w-24 shrink-0 text-right text-xs text-slate-700 tabular-nums">{formatMoney(v, cur)}</span>
    </div>
  )
  return (
    <ul className="space-y-3">
      {rows.map((r) => (
        <li key={r.member_id} className="flex items-center gap-3">
          <Avatar name={memberName(r.member_id)} size="sm" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{memberName(r.member_id)}</p>
            <div className="mt-1 space-y-0.5">
              {bar(r.paid, PAID, t('summary.paid'))}
              {bar(r.consumed, CONSUMED, t('summary.consumed'))}
            </div>
          </div>
        </li>
      ))}
    </ul>
  )
}
