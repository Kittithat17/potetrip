import { useBalances, useMarkPaid, useSettlement } from '../../api/hooks'
import type { Transfer } from '../../api/types'
import { Avatar, Button, Card, ErrorText, Page, PageHeader, Spinner } from '../../components/ui'
import { formatMoney } from '../../lib/money'
import { useTrip } from './context'

export function SettlePage() {
  const { trip, me, members, memberName, base } = useTrip()
  const cur = trip.base_currency
  const planQ = useSettlement(trip.id, me.id)
  const balQ = useBalances(trip.id, me.id)
  const paid = useMarkPaid(trip.id, me.id)

  const onPaid = (t: Transfer) => {
    if (!confirm(`ยืนยันว่า ${memberName(t.from)} โอนให้ ${memberName(t.to)} ${formatMoney(t.amount_base, cur)} แล้ว?`)) return
    paid.mutate(t)
  }

  const transfers = planQ.data?.transfers ?? []
  const mine = (t: Transfer) => t.from === me.id || t.to === me.id

  return (
    <Page>
      <PageHeader title="ใครต้องโอนให้ใคร" back={base} />
      <div className="space-y-4">
        <Card>
          {planQ.isPending ? (
            <Spinner />
          ) : planQ.error ? (
            <ErrorText error={planQ.error} />
          ) : transfers.length === 0 ? (
            <p className="py-6 text-center text-slate-600">ไม่มีใครติดเงินใครแล้ว 🎉</p>
          ) : (
            <ul className="space-y-3">
              {[...transfers]
                .sort((a, b) => Number(mine(b)) - Number(mine(a)))
                .map((t) => (
                  <li
                    key={`${t.from}-${t.to}`}
                    className={`rounded-xl p-3 ring-1 ${mine(t) ? 'bg-amber-50 ring-amber-200' : 'ring-slate-200'}`}
                  >
                    <div className="flex items-center gap-2">
                      <Avatar name={memberName(t.from)} size="sm" />
                      <span className="font-medium">{t.from === me.id ? 'คุณ' : memberName(t.from)}</span>
                      <span className="text-slate-400">→</span>
                      <Avatar name={memberName(t.to)} size="sm" />
                      <span className="font-medium">{t.to === me.id ? 'คุณ' : memberName(t.to)}</span>
                    </div>
                    <div className="mt-2 flex items-center justify-between">
                      <span className="text-xl font-bold tabular-nums">{formatMoney(t.amount_base, cur)}</span>
                      <Button variant="secondary" className="min-h-9 text-sm" onClick={() => onPaid(t)} disabled={paid.isPending}>
                        โอนแล้ว
                      </Button>
                    </div>
                  </li>
                ))}
            </ul>
          )}
          <ErrorText error={paid.error} />
        </Card>

        <Card>
          <h2 className="mb-2 font-bold">ยอดสุทธิของแต่ละคน</h2>
          {balQ.data && (
            <ul className="divide-y divide-slate-100">
              {balQ.data.balances.map((b) => (
                <li key={b.member_id} className="flex items-center gap-3 py-2">
                  <Avatar name={memberName(b.member_id)} size="sm" />
                  <span className="flex-1">
                    {memberName(b.member_id)}
                    <span className="block text-xs text-slate-500">
                      จ่ายไป {formatMoney(b.paid, cur)} · ส่วนตัวเอง {formatMoney(b.owed, cur)}
                    </span>
                  </span>
                  <span
                    className={`font-semibold tabular-nums ${b.net > 0 ? 'text-emerald-700' : b.net < 0 ? 'text-red-700' : 'text-slate-500'}`}
                  >
                    {b.net > 0 ? '+' : ''}
                    {formatMoney(b.net, cur)}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <p className="mt-2 text-xs text-slate-500">+ ได้คืน / − ต้องจ่าย · สมาชิก {members.length} คน</p>
        </Card>
      </div>
    </Page>
  )
}
