import { QrCode } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { useBalances, useMarkPaid, useSettlement } from '../../../api/hooks'
import type { Transfer } from '../../../api/types'
import { Approx } from '../../../components/money/Approx'
import { Avatar, Button, Card, ErrorText, Page, PageHeader, Spinner } from '../../../components/ui/ui'
import { useI18n } from '../../../i18n/context'
import { formatShortDate } from '../../../lib/datetime'
import { HOME_CURRENCY, useHomeApprox } from '../../../lib/homeApprox'
import { formatMoney } from '../../../lib/money'
import { useTrip } from '../layout/context'
import { useConfirm } from '../../../components/feedback/confirm'
import { PromptPayDialog } from './PromptPayDialog'

export function SettlePage() {
  const { t, locale } = useI18n()
  const { trip, me, members, memberName, base, nav } = useTrip()
  const cur = trip.base_currency
  const planQ = useSettlement(trip.id)
  const balQ = useBalances(trip.id)
  const paid = useMarkPaid(trip.id)
  const home = useHomeApprox(cur)
  const confirm = useConfirm()
  const [qrFor, setQrFor] = useState<Transfer | null>(null)
  const promptpayOf = (id: string) => members.find((m) => m.id === id)?.promptpay ?? null
  const qrReceiver = qrFor ? promptpayOf(qrFor.to) : null

  const onPaid = async (tr: Transfer) => {
    const title = t('settle.confirmPaid', { from: memberName(tr.from), to: memberName(tr.to), amount: formatMoney(tr.amount_base, cur) })
    if (!(await confirm({ title, tone: 'primary', confirmLabel: t('settle.paid') }))) return
    paid.mutate(tr)
  }

  const transfers = planQ.data?.transfers ?? []
  const mine = (tr: Transfer) => tr.from === me.id || tr.to === me.id
  const who = (id: string) => (id === me.id ? t('common.you') : memberName(id))

  return (
    <Page nav={nav}>
      <PageHeader title={t('settle.title')} back={base} />
      <div className="space-y-4">
        <Card>
          {planQ.isPending ? (
            <Spinner />
          ) : planQ.error ? (
            <ErrorText error={planQ.error} />
          ) : transfers.length === 0 ? (
            <p className="py-6 text-center text-slate-600">{t('settle.nothing')}</p>
          ) : (
            <ul className="space-y-3">
              {[...transfers]
                .sort((a, b) => Number(mine(b)) - Number(mine(a)))
                .map((tr) => (
                  <li
                    key={`${tr.from}-${tr.to}`}
                    className={`rounded-xl p-3 ring-1 ${mine(tr) ? 'bg-amber-50 ring-amber-200' : 'ring-slate-200'}`}
                  >
                    <div className="flex items-center gap-2">
                      <Avatar name={memberName(tr.from)} size="sm" />
                      <span className="font-medium">{who(tr.from)}</span>
                      <span className="text-slate-400">→</span>
                      <Avatar name={memberName(tr.to)} size="sm" />
                      <span className="font-medium">{who(tr.to)}</span>
                    </div>
                    <div className="mt-2 flex items-center justify-between">
                      <span>
                        <span className="block text-xl font-bold tabular-nums">{formatMoney(tr.amount_base, cur)}</span>
                        <Approx minor={tr.amount_base} currency={cur} />
                      </span>
                      <span className="flex gap-2">
                        {promptpayOf(tr.to) && (
                          <Button variant="secondary" className="min-h-9 px-3 text-sm" onClick={() => setQrFor(tr)} aria-label={t('settle.qr')}>
                            <QrCode className="size-4" /> {t('settle.qr')}
                          </Button>
                        )}
                        <Button variant="secondary" className="min-h-9 text-sm" onClick={() => onPaid(tr)} disabled={paid.isPending}>
                          {t('settle.paid')}
                        </Button>
                      </span>
                    </div>
                    {tr.to === me.id && !promptpayOf(tr.to) && (
                      <p className="mt-2 text-xs text-slate-500">
                        {t('settle.addPromptPay')}{' '}
                        <Link to={`${base}/edit`} className="text-brand-700 underline">
                          {t('settle.addPromptPayLink')}
                        </Link>
                      </p>
                    )}
                  </li>
                ))}
            </ul>
          )}
          <ErrorText error={paid.error} />
          <PromptPayDialog
            transfer={qrFor}
            receiverName={qrFor ? memberName(qrFor.to) : ''}
            promptpay={qrReceiver ?? ''}
            currency={cur}
            onClose={() => setQrFor(null)}
          />
        </Card>

        <Card>
          <h2 className="mb-2 font-bold">{t('settle.net')}</h2>
          {balQ.data && (
            <ul className="divide-y divide-slate-100">
              {balQ.data.balances.map((b) => (
                <li key={b.member_id} className="flex items-center gap-3 py-2">
                  <Avatar name={memberName(b.member_id)} size="sm" />
                  <span className="flex-1">
                    {memberName(b.member_id)}
                    <span className="block text-xs text-slate-500">
                      {t('settle.summary', { paid: formatMoney(b.paid, cur), owed: formatMoney(b.owed, cur) })}
                    </span>
                  </span>
                  <span
                    className={`font-semibold tabular-nums ${b.net > 0 ? 'text-emerald-700' : b.net < 0 ? 'text-red-700' : 'text-slate-500'}`}
                  >
                    {b.net > 0 ? '+' : ''}
                    {formatMoney(b.net, cur)}
                    <Approx minor={b.net} currency={cur} className="text-right text-[11px] font-normal text-slate-400" />
                  </span>
                </li>
              ))}
            </ul>
          )}
          <p className="mt-2 text-xs text-slate-500">{t('settle.legend', { n: members.length })}</p>
          {home.rateDate && (
            <p className="mt-1 text-xs text-slate-400">
              {t('settle.approxNote', { home: HOME_CURRENCY, date: formatShortDate(home.rateDate, locale), cur })}
            </p>
          )}
        </Card>
      </div>
    </Page>
  )
}
