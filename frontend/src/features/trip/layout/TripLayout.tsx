import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeftRight, HandCoins, House, ScanLine } from 'lucide-react'
import { useMemo } from 'react'
import { Link, Outlet, useNavigate, useParams } from 'react-router'
import { ApiError } from '../../../api/client'
import { leaveTrip } from '../../../api/endpoints'
import { keys, useMe, useMembers, useTripByInvite } from '../../../api/hooks'
import { ErrorText, Page, Spinner } from '../../../components/ui/ui'
import { settingsTab } from '../../../app/nav'
import { useI18n } from '../../../i18n/context'
import type { TripCtx } from './context'
import { JoinPage, WaitingPage } from '../join/JoinPage'

export function TripLayout() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const code = (useParams().code ?? '').toUpperCase()
  const inviteQ = useTripByInvite(code)
  const trip = inviteQ.data?.trip

  // Where the signed-in user stands in this trip (member, pending, or not in it).
  const meQ = useMe(trip?.id)
  const notInTrip = meQ.error instanceof ApiError && meQ.error.code === 'not_a_member'
  const approved = meQ.data?.status === 'approved' && meQ.data.member
  const membersQ = useMembers(approved ? trip?.id : undefined)
  const me = meQ.data?.member ?? undefined

  const refreshMe = async () => {
    if (!trip) return
    await qc.invalidateQueries({ queryKey: keys.trip(trip.id) })
    await qc.invalidateQueries({ queryKey: keys.myTrips })
  }

  const leave = useMutation({
    mutationFn: () => leaveTrip(trip!.id),
    onSuccess: async () => {
      qc.removeQueries({ queryKey: keys.trip(trip!.id) })
      await qc.invalidateQueries({ queryKey: keys.myTrips })
    },
  })

  const ctx = useMemo<TripCtx | null>(() => {
    if (!trip || !me || !membersQ.data) return null
    const names = new Map(membersQ.data.map((m) => [m.id, m.name]))
    const base = `/t/${trip.invite_code}`
    return {
      trip,
      members: membersQ.data,
      me,
      isOwner: !!meQ.data?.is_owner,
      memberName: (id) => names.get(id) ?? '?',
      base,
      leave: async () => {
        await leave.mutateAsync()
        navigate('/trips', { replace: true })
      },
      nav: [
        { to: base, labelKey: 'nav.overview', icon: House, end: true },
        { to: `${base}/scan`, labelKey: 'nav.scan', icon: ScanLine },
        { to: `${base}/convert`, labelKey: 'nav.convert', icon: ArrowLeftRight },
        { to: `${base}/settle`, labelKey: 'nav.settle', icon: HandCoins },
        settingsTab,
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trip, me, membersQ.data, meQ.data?.is_owner])

  if (inviteQ.isPending) {
    return (
      <Page>
        <Spinner />
      </Page>
    )
  }
  if (inviteQ.error || !trip) {
    const notFound = inviteQ.error instanceof ApiError && inviteQ.error.status === 404
    return (
      <Page>
        <div className="space-y-4 pt-16 text-center">
          <p className="text-4xl">🧳</p>
          {notFound ? <p className="text-lg font-medium">{t('trip.notFound')}</p> : <ErrorText error={inviteQ.error} />}
          <Link to="/" className="inline-block text-brand-700 underline">
            {t('common.backHome')}
          </Link>
        </div>
      </Page>
    )
  }

  if (notInTrip) {
    return <JoinPage trip={trip} members={inviteQ.data.members} onRequested={refreshMe} />
  }
  if (meQ.isPending) {
    return (
      <Page>
        <Spinner />
      </Page>
    )
  }
  if (meQ.error) {
    return (
      <Page>
        <div className="pt-16">
          <ErrorText error={meQ.error} />
        </div>
      </Page>
    )
  }
  if (meQ.data.status !== 'approved') {
    // Cancelling (or, after a rejection, clearing it to ask again) drops the request.
    return <WaitingPage trip={trip} me={meQ.data} onCancel={() => leave.mutate(undefined, { onSuccess: refreshMe })} cancelling={leave.isPending} />
  }
  if (!ctx) {
    return membersQ.error ? (
      <Page>
        <ErrorText error={membersQ.error} />
      </Page>
    ) : (
      <Page>
        <Spinner />
      </Page>
    )
  }
  return <Outlet context={ctx} />
}
