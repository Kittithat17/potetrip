import { ChevronLeft, ChevronRight, HandCoins, Plus, ScanLine, Settings2, Share2, UserPlus } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useBalances, useExpenses } from '../../api/hooks'
import type { Category } from '../../api/types'
import { Approx } from '../../components/Approx'
import { ExpenseRow } from '../../components/ExpenseRow'
import { JoinRequestsCard } from './JoinRequestsCard'
import { AppFrame, BlueHeader, IconAction, Sheet } from '../../components/shell'
import { ErrorText, Spinner } from '../../components/ui'
import { CATEGORIES } from '../../lib/categories'
import { inviteLink } from '../../lib/invite'
import { formatMoney } from '../../lib/money'
import { useTrip } from './context'
import { useI18n } from '../../i18n/context'
import { useConfirm } from '../../lib/confirm'
import { InviteDialog } from './InviteDialog'

export function DashboardPage() {
  const { t } = useI18n()
  const { trip, me, token, isOwner, members, memberName, base, leave, nav } = useTrip()
  const cur = trip.base_currency
  const expensesQ = useExpenses(trip.id, token)
  const balancesQ = useBalances(trip.id, token)
  const [category, setCategory] = useState<Category | null>(null)
  const [onlyMine, setOnlyMine] = useState(false)
  const [copied, setCopied] = useState(false)
  const [inviteOpen, setInviteOpen] = useState(false)
  const confirm = useConfirm()

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
    if (navigator.share) {
      try {
        await navigator.share({ title: trip.name, text: t('trip.shareText', { name: trip.name }), url })
      } catch {
        /* user closed the share sheet */
      }
      return
    }
    // No share sheet (desktop, or plain http): show our own invite dialog.
    setInviteOpen(true)
  }

  const onLeave = async () => {
    const ok = await confirm({ title: t('trip.leaveTitle'), message: t('trip.leaveConfirm'), confirmLabel: t('trip.leaveAction') })
    if (ok) leave()
  }

  return (
    <AppFrame nav={nav}>
      <InviteDialog open={inviteOpen} onClose={() => setInviteOpen(false)} onCopied={() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }} />
      <BlueHeader className="pb-20">
        <div className="flex items-center gap-1">
          <Link to="/trips" aria-label={t('trip.myTripsLabel')} className="-ml-2 rounded-full p-1 hover:bg-white/15">
            <ChevronLeft className="size-7" />
          </Link>
          <h1 className="min-w-0 flex-1 truncate text-xl font-bold">{trip.name}</h1>
          <button onClick={share} className="flex items-center gap-1 rounded-full bg-white/15 px-3 py-1.5 text-sm hover:bg-white/25">
            <Share2 className="size-4" />
            {copied ? t('trip.copied') : t('trip.invite')}
          </button>
          <Link to={`${base}/edit`} aria-label={t('trip.settingsLabel')} className="rounded-full bg-white/15 p-2 hover:bg-white/25">
            <Settings2 className="size-4" />
          </Link>
        </div>

        <div className="mt-4 flex items-end justify-between">
          <div>
            <p className="text-sm text-brand-100">
              {t('trip.hello', { name: me.name })}{' '}
              <button
                onClick={onLeave}
                className="ml-1 text-xs text-brand-200 underline"
              >
                {t('trip.notMe')}
              </button>
            </p>
            {!myBalance ? (
              <p className="mt-1 text-3xl font-bold">…</p>
            ) : myBalance.net > 0 ? (
              <>
                <p className="mt-1 text-sm text-brand-100">{t('trip.getsBack')}</p>
                <p className="text-4xl font-bold tabular-nums">{formatMoney(myBalance.net, cur)}</p>
                <Approx minor={myBalance.net} currency={cur} className="text-sm text-brand-100" />
              </>
            ) : myBalance.net < 0 ? (
              <>
                <p className="mt-1 text-sm text-brand-100">{t('trip.owes')}</p>
                <p className="text-4xl font-bold text-amber-200 tabular-nums">{formatMoney(-myBalance.net, cur)}</p>
                <Approx minor={-myBalance.net} currency={cur} className="text-sm text-brand-100" />
              </>
            ) : (
              <p className="mt-1 text-2xl font-bold">{t('trip.settled')}</p>
            )}
          </div>
          <div className="text-right text-xs text-brand-100">
            <p>{t('trip.total')}</p>
            <p className="text-base font-semibold text-white tabular-nums">{formatMoney(total, cur)}</p>
            <Approx minor={total} currency={cur} className="text-xs text-brand-100" />
            <p>{members.length === 1 ? t('trip.peopleOne') : t('trip.people', { n: members.length })}</p>
          </div>
        </div>
      </BlueHeader>

      <main className="mx-auto -mt-14 max-w-md space-y-4 px-4">
        <Sheet>
          <div className="grid grid-cols-4 gap-2">
            <IconAction icon={Plus} label={t('trip.addExpense')} to={`${base}/expenses/new`} />
            <IconAction icon={ScanLine} label={t('trip.scanSlip')} to={`${base}/scan`} />
            <IconAction icon={HandCoins} label={t('trip.settle')} to={`${base}/settle`} />
            <IconAction icon={UserPlus} label={t('trip.invite')} onClick={share} />
          </div>
        </Sheet>

        {isOwner && <JoinRequestsCard />}

        <section className="rounded-3xl bg-white px-2 pt-4 pb-2 shadow-sm ring-1 ring-slate-200/70">
          <div className="flex items-center justify-between px-2">
            <h2 className="text-lg font-bold">{t('trip.recent')}</h2>
            <label className="flex items-center gap-1 text-sm text-slate-600">
              <input type="checkbox" checked={onlyMine} onChange={(e) => setOnlyMine(e.target.checked)} className="accent-brand-600" />
              {t('trip.onlyMine')}
            </label>
          </div>
          <div className="no-scrollbar mt-3 flex gap-2 overflow-x-auto px-2 pb-1">
            <Chip active={!category} onClick={() => setCategory(null)}>
              {t('trip.all')}
            </Chip>
            {CATEGORIES.map((c) => (
              <Chip key={c.value} active={category === c.value} onClick={() => setCategory(c.value)}>
                <c.icon className="size-4" /> {t(c.labelKey)}
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
            <div className="py-10 text-center text-sm text-slate-500">
              {expenses.length === 0 ? (
                <>
                  <p>{t('trip.noExpenses')}</p>
                  <Link to={`${base}/expenses/new`} className="mt-2 inline-flex items-center text-brand-600">
                    {t('trip.addFirst')} <ChevronRight className="size-4" />
                  </Link>
                </>
              ) : (
                t('trip.noMatch')
              )}
            </div>
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
        </section>
      </main>
    </AppFrame>
  )
}

function Chip({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`flex shrink-0 items-center gap-1 rounded-full px-3 py-1.5 text-sm ${
        active ? 'bg-brand-600 text-white' : 'bg-slate-100 text-slate-700'
      }`}
    >
      {children}
    </button>
  )
}
