import { ArrowLeftRight, HandCoins, House, ScanLine } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { Link, Outlet, useParams } from 'react-router'
import { ApiError } from '../../api/client'
import { useMe, useMembers, useTripByInvite } from '../../api/hooks'
import { ErrorText, Page, Spinner } from '../../components/ui'
import { clearIdentity, getIdentity, setIdentity } from '../../lib/identity'
import { settingsTab } from '../../lib/nav'
import { useI18n } from '../../i18n/context'
import { rememberTrip } from '../../lib/recentTrips'
import type { TripCtx } from './context'
import { JoinPage, WaitingPage } from './JoinPage'

export function TripLayout() {
  const { t } = useI18n()
  const code = (useParams().code ?? '').toUpperCase()
  const inviteQ = useTripByInvite(code)
  const trip = inviteQ.data?.trip
  const [identity, setIdentityState] = useState(() => (trip ? getIdentity(trip.id) : null))

  // Load the stored credential once the trip id is known.
  useEffect(() => {
    if (trip) setIdentityState(getIdentity(trip.id))
  }, [trip])

  const token = identity?.token
  const meQ = useMe(trip?.id, token)
  const approved = meQ.data?.status === 'approved' && meQ.data.member
  const membersQ = useMembers(approved ? trip?.id : undefined, token)
  const me = meQ.data?.member ?? undefined

  const forget = () => {
    if (!trip) return
    clearIdentity(trip.id)
    setIdentityState(null)
  }

  // The token no longer works (e.g. the database was reset): start over.
  const tokenRejected = meQ.error instanceof ApiError && (meQ.error.status === 401 || meQ.error.code === 'not_a_member')
  useEffect(() => {
    if (tokenRejected) forget()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tokenRejected])

  // Remember the trip on this device, and cache "me" for list pages.
  useEffect(() => {
    if (!trip) return
    rememberTrip({ id: trip.id, code: trip.invite_code, name: trip.name, myName: me?.name })
    if (me && token && identity?.memberId !== me.id) {
      const next = { token, memberId: me.id }
      setIdentity(trip.id, next)
      setIdentityState(next)
    }
  }, [trip, me, token, identity?.memberId])

  const ctx = useMemo<TripCtx | null>(() => {
    if (!trip || !me || !token || !membersQ.data) return null
    const names = new Map(membersQ.data.map((m) => [m.id, m.name]))
    const base = `/t/${trip.invite_code}`
    return {
      trip,
      members: membersQ.data,
      me,
      token,
      isOwner: !!meQ.data?.is_owner,
      memberName: (id) => names.get(id) ?? '?',
      base,
      leave: forget,
      nav: [
        { to: base, labelKey: 'nav.overview', icon: House, end: true },
        { to: `${base}/scan`, labelKey: 'nav.scan', icon: ScanLine },
        { to: `${base}/convert`, labelKey: 'nav.convert', icon: ArrowLeftRight },
        { to: `${base}/settle`, labelKey: 'nav.settle', icon: HandCoins },
        settingsTab,
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trip, me, token, membersQ.data, meQ.data?.is_owner])

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

  if (!token) {
    return (
      <JoinPage
        trip={trip}
        members={inviteQ.data.members}
        onRequested={(t) => {
          const next = { token: t }
          setIdentity(trip.id, next)
          setIdentityState(next)
        }}
      />
    )
  }
  if (meQ.isPending || tokenRejected) {
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
    return <WaitingPage trip={trip} me={meQ.data} onCancel={forget} />
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
