import { Link } from 'react-router'
import type { Expense } from '../api/types'
import { categoryInfo } from '../lib/categories'
import { formatShortDateTime } from '../lib/datetime'
import { formatMoney } from '../lib/money'

type Props = { expense: Expense; to: string; payerName: string; myShare?: number; currency: string }

export function ExpenseRow({ expense: e, to, payerName, myShare, currency }: Props) {
  const cat = categoryInfo(e.category)
  return (
    <li>
      <Link to={to} className="flex items-center gap-3 rounded-xl px-2 py-3 hover:bg-slate-50">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-teal-50 text-xl" aria-label={cat.label}>
          {cat.icon}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-medium">{e.title}</span>
          <span className="block truncate text-xs text-slate-500">
            {payerName} จ่าย · {formatShortDateTime(e.spent_at)}
          </span>
        </span>
        <span className="text-right">
          <span className="block font-semibold tabular-nums">{formatMoney(e.amount, e.currency)}</span>
          {myShare !== undefined && myShare > 0 && (
            <span className="block text-xs text-slate-500 tabular-nums">ส่วนฉัน {formatMoney(myShare, currency)}</span>
          )}
        </span>
      </Link>
    </li>
  )
}
