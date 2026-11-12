import { useQuery } from '@tanstack/react-query'
import { ArrowUpDown, Info } from 'lucide-react'
import { useEffect, useState } from 'react'
import { getFx, getFxCurrencies } from '../api/endpoints'
import { AppFrame, BlueHeader, Sheet } from '../components/shell'
import { ErrorText, Spinner } from '../components/ui'
import { currencyFlag, currencyName } from '../lib/currencyInfo'
import { exponent } from '../lib/money'
import type { NavItem } from '../lib/nav'
import { useI18n } from '../i18n/context'
import { formatShortDate } from '../lib/datetime'

const PAIR_KEY = 'travel-split:convert-pair'
const QUICK = [10, 50, 100, 500, 1000]
const TABLE = [1, 10, 50, 100, 500, 1000, 5000]

function loadPair(defaultTo?: string): [string, string] {
  try {
    const p = JSON.parse(localStorage.getItem(PAIR_KEY) ?? 'null')
    if (Array.isArray(p) && p.length === 2) return defaultTo && p[0] !== defaultTo ? [p[0], defaultTo] : [p[0], p[1]]
  } catch {
    /* ignore */
  }
  return ['CNY', defaultTo ?? 'THB']
}

/** Rough display formatting; this page is for estimating, not bookkeeping. */
function fmt(v: number, cur: string, locale: string): string {
  const exp = exponent(cur)
  return v.toLocaleString(locale, { minimumFractionDigits: exp, maximumFractionDigits: exp })
}

type Props = { nav: NavItem[]; defaultTo?: string }

export function ConvertPage({ nav, defaultTo }: Props) {
  const { t, locale } = useI18n()
  const [[from, to], setPair] = useState<[string, string]>(() => loadPair(defaultTo))
  const [amountText, setAmountText] = useState('100')

  useEffect(() => {
    try {
      localStorage.setItem(PAIR_KEY, JSON.stringify([from, to]))
    } catch {
      /* ignore */
    }
  }, [from, to])

  const currenciesQ = useQuery({ queryKey: ['fx', 'currencies'], queryFn: getFxCurrencies, staleTime: 24 * 3600_000 })
  const rateQ = useQuery({
    queryKey: ['fx', from, to],
    queryFn: () => getFx(from, to),
    staleTime: 10 * 60_000,
    refetchInterval: 10 * 60_000,
  })

  const rate = rateQ.data ? Number(rateQ.data.rate) : null
  const amount = Number(amountText.replace(/,/g, ''))
  const valid = amountText.trim() !== '' && Number.isFinite(amount) && amount >= 0
  const result = rate !== null && valid ? amount * rate : null
  const currencies = currenciesQ.data ?? [from, to]

  const select = (value: string, onChange: (v: string) => void, label: string) => (
    <label className="relative flex shrink-0 items-center gap-2 rounded-xl bg-slate-100 px-3 py-2">
      <span className="text-2xl leading-none">{currencyFlag(value)}</span>
      <span className="font-bold">{value}</span>
      <select
        aria-label={label}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="absolute inset-0 cursor-pointer opacity-0"
      >
        {currencies.map((c) => (
          <option key={c} value={c}>
            {c} · {currencyName(c, locale)}
          </option>
        ))}
      </select>
    </label>
  )

  return (
    <AppFrame nav={nav}>
      <BlueHeader className="pb-16">
        <h1 className="text-xl font-bold">{t('convert.title')}</h1>
        <p className="mt-0.5 text-sm text-brand-100">{t('convert.subtitle')}</p>
      </BlueHeader>

      <main className="mx-auto -mt-10 max-w-md space-y-4 px-4">
        <Sheet className="space-y-3">
          <div className="flex items-center gap-3 rounded-2xl ring-1 ring-slate-200 p-3">
            {select(from, (v) => setPair([v, to]), t('convert.from'))}
            <input
              value={amountText}
              onChange={(e) => setAmountText(e.target.value)}
              inputMode="decimal"
              aria-label={t('convert.amount')}
              className="min-w-0 flex-1 bg-transparent text-right text-3xl font-bold tabular-nums outline-none"
            />
          </div>
          <p className="-mt-1 pl-1 text-xs text-slate-500">{currencyName(from, locale)}</p>

          <div className="relative flex justify-center">
            <span className="absolute inset-x-0 top-1/2 border-t border-slate-200" />
            <button
              onClick={() => setPair([to, from])}
              aria-label={t('convert.swap')}
              className="relative flex size-11 items-center justify-center rounded-full bg-gradient-to-b from-brand-500 to-brand-700 text-white shadow-md active:scale-95"
            >
              <ArrowUpDown className="size-5" />
            </button>
          </div>

          <div className="flex items-center gap-3 rounded-2xl bg-brand-50 p-3">
            {select(to, (v) => setPair([from, v]), t('convert.to'))}
            <p className="min-w-0 flex-1 truncate text-right text-3xl font-bold text-brand-700 tabular-nums" aria-live="polite">
              {rateQ.isPending ? '…' : result !== null ? fmt(result, to, locale) : '-'}
            </p>
          </div>
          <p className="-mt-1 pl-1 text-xs text-slate-500">{currencyName(to, locale)}</p>

          <div className="no-scrollbar flex gap-2 overflow-x-auto">
            {QUICK.map((q) => (
              <button
                key={q}
                onClick={() => setAmountText(String(q))}
                className={`shrink-0 rounded-full px-3 py-1.5 text-sm ${amountText === String(q) ? 'bg-brand-600 text-white' : 'bg-slate-100 text-slate-700'}`}
              >
                {q.toLocaleString(locale)} {from}
              </button>
            ))}
          </div>

          {rateQ.error ? (
            <ErrorText error={rateQ.error} />
          ) : rateQ.data ? (
            <p className="flex items-start gap-1.5 rounded-xl bg-slate-50 p-3 text-xs text-slate-600">
              <Info className="mt-0.5 size-4 shrink-0 text-slate-400" />
              <span>
                {t('convert.rateNote', {
                  from,
                  to,
                  rate: Number(rateQ.data.rate).toLocaleString(locale, { maximumFractionDigits: 6 }),
                  date: formatShortDate(rateQ.data.rate_date, locale),
                })}
                {rateQ.data.stale && t('convert.stale')}
              </span>
            </p>
          ) : null}
        </Sheet>

        <section className="rounded-3xl bg-white p-4 shadow-sm ring-1 ring-slate-200/70">
          <h2 className="mb-2 font-bold">{t('convert.quick')}</h2>
          {rate === null ? (
            rateQ.isPending ? <Spinner /> : null
          ) : (
            <table className="w-full text-sm tabular-nums">
              <tbody className="divide-y divide-slate-100">
                {TABLE.map((n) => (
                  <tr key={n}>
                    <td className="py-2 text-slate-600">
                      {n.toLocaleString(locale)} {from}
                    </td>
                    <td className="py-2 text-right font-semibold">
                      {fmt(n * rate, to, locale)} {to}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </main>
    </AppFrame>
  )
}
