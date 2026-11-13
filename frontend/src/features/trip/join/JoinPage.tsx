import { useMutation } from '@tanstack/react-query'
import { Clock, ShieldCheck, UserRoundX } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { requestJoin } from '../../../api/endpoints'
import type { Me, PublicMember, Trip } from '../../../api/types'
import { AppFrame, BlueHeader, Sheet } from '../../../components/ui/shell'
import { Avatar, Button, ErrorText, Field, TextInput } from '../../../components/ui/ui'
import { useI18n } from '../../../i18n/context'
import { useAuth } from '../../auth/context'

type JoinProps = { trip: Trip; members: PublicMember[]; onRequested: () => void }

/** Someone opened an invite link: they may only ask the owner to let them in. */
export function JoinPage({ trip, members, onRequested }: JoinProps) {
  const { t } = useI18n()
  const { user } = useAuth()
  const [name, setName] = useState(user.name)
  const [promptpay, setPromptpay] = useState('')
  const join = useMutation({
    mutationFn: (body: { name?: string; member_id?: string; promptpay?: string | null }) => requestJoin(trip.id, body),
    onSuccess: () => onRequested(),
  })

  const onNew = (e: FormEvent) => {
    e.preventDefault()
    join.mutate({ name, promptpay: promptpay || null })
  }

  return (
    <AppFrame>
      <BlueHeader className="pb-16">
        <p className="text-sm text-brand-100">{t('join.invited')}</p>
        <h1 className="text-2xl font-bold">{trip.name}</h1>
      </BlueHeader>
      <main className="mx-auto -mt-10 max-w-md space-y-4 px-4">
        <Sheet className="space-y-3">
          <p className="flex items-start gap-2 rounded-xl bg-brand-50 p-3 text-sm text-brand-800">
            <ShieldCheck className="mt-0.5 size-4 shrink-0" />
            {t('join.ownerMustApprove')}
          </p>
          <h2 className="font-bold">{t('join.joinAs')}</h2>
          <form onSubmit={onNew} className="space-y-3">
            <Field label={t('join.yourName')}>
              <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder={t('join.nickname')} required maxLength={50} />
            </Field>
            <Field label={t('join.promptpay')} hint={t('join.promptpayHint')}>
              <TextInput value={promptpay} onChange={(e) => setPromptpay(e.target.value)} inputMode="numeric" placeholder="08x-xxx-xxxx" />
            </Field>
            <ErrorText error={join.error} />
            <Button type="submit" className="w-full" disabled={join.isPending}>
              {join.isPending ? t('join.sending') : t('join.send')}
            </Button>
          </form>
        </Sheet>

        {members.some((m) => m.claimable) && (
          <Sheet>
            <h2 className="font-bold">{t('join.alreadyIn')}</h2>
            <p className="mb-2 text-sm text-slate-500">{t('join.alreadyInHint')}</p>
            <ul className="divide-y divide-slate-100">
              {members.filter((m) => m.claimable).map((m) => (
                <li key={m.id}>
                  <button
                    onClick={() => join.mutate({ member_id: m.id })}
                    disabled={join.isPending}
                    className="flex w-full items-center gap-3 py-2.5 text-left hover:bg-slate-50"
                  >
                    <Avatar name={m.name} size="sm" />
                    <span className="flex-1">{m.name}</span>
                    <span className="text-sm text-brand-600">{t('join.thisIsMe')}</span>
                  </button>
                </li>
              ))}
            </ul>
          </Sheet>
        )}
      </main>
    </AppFrame>
  )
}

type WaitProps = { trip: Trip; me: Me; onCancel: () => void; cancelling?: boolean }

export function WaitingPage({ trip, me, onCancel, cancelling }: WaitProps) {
  const { t } = useI18n()
  const rejected = me.status === 'rejected'
  return (
    <AppFrame>
      <BlueHeader className="pb-16">
        <p className="text-sm text-brand-100">{t('join.trip')}</p>
        <h1 className="text-2xl font-bold">{trip.name}</h1>
      </BlueHeader>
      <main className="mx-auto -mt-10 max-w-md px-4">
        <Sheet className="space-y-4 py-8 text-center">
          {rejected ? (
            <>
              <UserRoundX className="mx-auto size-14 text-red-500" />
              <p className="text-lg font-bold">{t('join.rejected')}</p>
              <p className="text-sm text-slate-600">{t('join.rejectedHint')}</p>
              <Button className="w-full" onClick={onCancel} disabled={cancelling}>
                {t('join.requestAgain')}
              </Button>
            </>
          ) : (
            <>
              <Clock className="mx-auto size-14 animate-pulse text-brand-500" />
              <p className="text-lg font-bold">{t('join.waiting')}</p>
              <p className="text-sm text-slate-600">
                {t('join.waitingAs')} <b>{me.request.requested_name}</b>
                <br />
                {t('join.waitingHint')}
              </p>
              <Button variant="ghost" className="w-full" onClick={onCancel} disabled={cancelling}>
                {t('join.cancel')}
              </Button>
            </>
          )}
          <Link to="/" className="block text-sm text-brand-600">
            {t('common.backHome')}
          </Link>
        </Sheet>
      </main>
    </AppFrame>
  )
}
