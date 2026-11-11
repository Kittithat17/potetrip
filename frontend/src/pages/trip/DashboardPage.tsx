import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useBalances, useExpenses } from '../../api/hooks'
import { ExpenseRow } from '../../components/ExpenseRow'
import { Button, Card, ErrorText, Page, PageHeader, Spinner } from '../../components/ui'
import { CATEGORIES } from '../../lib/categories'
import { inviteLink } from '../../lib/invite'
import { formatMoney } from '../../lib/money'
import type { Category } from '../../api/types'
import { useTrip } from './context'

export function DashboardPage() {
  const { trip, me, memberName, base, switchUser } = useTrip()
  const cur = trip.base_currency
  const expensesQ = useExpenses(trip.id, me.id)
  const balancesQ = useBalances(trip.id, me.id)
  const [category, setCategory] = useState<Category | null>(null)
  const [onlyMine, setOnlyMine] = useState(false)
  const [copied, setCopied] = useState(false)

  const myBalance = balancesQ.data?.balances.find((b) => b.member_id === me.id)
  const expenses = expensesQ.data ?? []
  const total = expenses.reduce((s, e) => s + e.amount_base, 0)
  const visible = expenses.filter(
    (e) =>
      (!category || e.category === category) &&
      (!onlyMine || e.paid_by === me.id || e.shares.some((s) => s.member_id === me.id)),
  )

  const share = async () => {
    const url = inviteLink(trip.invite_code)
    try {
      if (navigator.share) {
        await navigator.share({ title: trip.name, text: `มาหารค่าใช้จ่ายทริป "${trip.name}" กัน`, url })
        return
      }
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      /* user cancelled share sheet */
    }
  }

  return (
    <Page>
      <PageHeader
        title={trip.name}
        back="/"
        right={
          <Button variant="ghost" className="min-h-9 px-3 text-sm" onClick={share}>
            {copied ? 'คัดลอกแล้ว ✓' : 'ชวนเพื่อน'}
          </Button>
        }
      />

      <div className="space-y-4">
        <Card className="bg-gradient-to-br from-teal-700 to-teal-900 text-white ring-0">
          <div className="flex items-start justify-between">
            <p className="text-sm text-teal-100">สวัสดี {me.name}</p>
            <button onClick={switchUser} className="text-xs text-teal-200 underline">
              ไม่ใช่ฉัน
            </button>
          </div>
          {!myBalance ? (
            <p className="mt-2 text-2xl font-bold">…</p>
          ) : myBalance.net > 0 ? (
            <>
              <p className="mt-2 text-sm text-teal-100">คุณจะได้คืน</p>
              <p className="text-3xl font-bold tabular-nums">{formatMoney(myBalance.net, cur)}</p>
            </>
          ) : myBalance.net < 0 ? (
            <>
              <p className="mt-2 text-sm text-teal-100">คุณต้องจ่ายเพื่อน</p>
              <p className="text-3xl font-bold tabular-nums text-amber-200">{formatMoney(-myBalance.net, cur)}</p>
            </>
          ) : (
            <p className="mt-2 text-2xl font-bold">เคลียร์ครบแล้ว 🎉</p>
          )}
          <div className="mt-3 flex items-center justify-between text-sm text-teal-100">
            <span>ทั้งทริป {formatMoney(total, cur)}</span>
            <Link to={`${base}/settle`} className="rounded-lg bg-white/15 px-3 py-1 font-medium text-white hover:bg-white/25">
              ดูว่าใครโอนให้ใคร ›
            </Link>
          </div>
        </Card>

        <div className="grid grid-cols-2 gap-3">
          <Link
            to={`${base}/expenses/new`}
            className="flex min-h-14 items-center justify-center rounded-2xl bg-teal-700 font-medium text-white shadow-sm hover:bg-teal-800"
          >
            + เพิ่มรายจ่าย
          </Link>
          <Link
            to={`${base}/scan`}
            className="flex min-h-14 items-center justify-center rounded-2xl bg-white font-medium text-teal-800 shadow-sm ring-1 ring-teal-700/30 hover:bg-teal-50"
          >
            📷 สแกนสลิป
          </Link>
        </div>

        <Card className="px-2">
          <div className="flex items-center justify-between px-2">
            <h2 className="font-bold">รายจ่ายล่าสุด</h2>
            <label className="flex items-center gap-1 text-sm text-slate-600">
              <input type="checkbox" checked={onlyMine} onChange={(e) => setOnlyMine(e.target.checked)} className="accent-teal-700" />
              เฉพาะที่เกี่ยวกับฉัน
            </label>
          </div>
          <div className="mt-2 flex gap-2 overflow-x-auto px-2 pb-1">
            <Chip active={!category} onClick={() => setCategory(null)}>
              ทั้งหมด
            </Chip>
            {CATEGORIES.map((c) => (
              <Chip key={c.value} active={category === c.value} onClick={() => setCategory(c.value)}>
                {c.icon} {c.label}
              </Chip>
            ))}
          </div>

          {expensesQ.isPending ? (
            <Spinner />
          ) : expensesQ.error ? (
            <div className="p-2">
              <ErrorText error={expensesQ.error} />
            </div>
          ) : visible.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500">
              {expenses.length === 0 ? 'ยังไม่มีรายจ่าย เริ่มเพิ่มรายการแรกได้เลย' : 'ไม่มีรายการที่ตรงกับตัวกรอง'}
            </p>
          ) : (
            <ul className="mt-1 divide-y divide-slate-100">
              {visible.map((e) => (
                <ExpenseRow
                  key={e.id}
                  expense={e}
                  to={`${base}/expenses/${e.id}`}
                  payerName={memberName(e.paid_by)}
                  myShare={e.shares.find((s) => s.member_id === me.id)?.amount_base}
                  currency={cur}
                />
              ))}
            </ul>
          )}
        </Card>
      </div>
    </Page>
  )
}

function Chip({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`shrink-0 rounded-full px-3 py-1 text-sm ${active ? 'bg-teal-700 text-white' : 'bg-slate-100 text-slate-700'}`}
    >
      {children}
    </button>
  )
}
