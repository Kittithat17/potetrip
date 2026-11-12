import { Link } from 'react-router'
import type { Expense } from '../api/types'
import { categoryInfo } from '../lib/categories'
import { formatShortDateTime } from '../lib/datetime'
import { formatMoney } from '../lib/money'
import { Approx } from './Approx'
import { useI18n } from '../i18n/context'

type Props = { expense: Expense; to: string; payerName: string; myShare?: number; currency: string }

export function ExpenseRow({ expense: e, to, payerName, myShare, currency }: Props) {
  const { t, locale } = useI18n()
  const cat = categoryInfo(e.category)
  return (
    <li>
      <Link to={to} className="flex items-center gap-3 rounded-xl px-2 py-3 hover:bg-slate-50">
        <span className={`flex size-11 shrink-0 items-center justify-center rounded-2xl ${cat.tint}`} aria-label={t(cat.labelKey)}>
          <cat.icon className="size-5" />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-medium">{e.title}</span>
          <span className="block truncate text-xs text-slate-500">
            {t('trip.paidBy', { name: payerName })} · {formatShortDateTime(e.spent_at, locale)}
          </span>
        </span>
        <span className="text-right">
          <span className="block font-semibold tabular-nums">{formatMoney(e.amount, e.currency)}</span>
          <Approx minor={e.amount} currency={e.currency} className="text-[11px] text-slate-400" />
          {myShare !== undefined && myShare > 0 && (
            <span className="block text-xs text-slate-500 tabular-nums">{t('trip.myShare', { amount: formatMoney(myShare, currency) })}</span>
          )}
        </span>
      </Link>
    </li>
  )
}
