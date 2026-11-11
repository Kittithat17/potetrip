import { useEffect, useMemo, useState } from 'react'
import { Link, Outlet, useParams } from 'react-router'
import { useMembers, useTripByInvite } from '../../api/hooks'
import { ApiError } from '../../api/client'
import { ErrorText, Page, Spinner } from '../../components/ui'
import { clearMyMemberId, getMyMemberId, setMyMemberId } from '../../lib/identity'
import { rememberTrip } from '../../lib/recentTrips'
import type { TripCtx } from './context'
import { PickMemberPage } from './PickMemberPage'

export function TripLayout() {
  const code = (useParams().code ?? '').toUpperCase()
  const tripQ = useTripByInvite(code)
  const trip = tripQ.data
  const membersQ = useMembers(trip?.id)
  const [myId, setMyId] = useState<string | null>(null)

  const storedId = trip ? (myId ?? getMyMemberId(trip.id)) : null
  const members = membersQ.data
  const me = members?.find((m) => m.id === storedId)

  // Remember every trip opened here so it shows up under "ทริปของฉัน" on the home page.
  useEffect(() => {
    if (trip) rememberTrip({ id: trip.id, code: trip.invite_code, name: trip.name, myName: me?.name })
  }, [trip, me?.name])

  const ctx = useMemo<TripCtx | null>(() => {
    if (!trip || !members || !me) return null
    const names = new Map(members.map((m) => [m.id, m.name]))
    return {
      trip,
      members,
      me,
      memberName: (id) => names.get(id) ?? '?',
      base: `/t/${trip.invite_code}`,
      switchUser: () => {
        clearMyMemberId(trip.id)
        setMyId(null)
      },
    }
  }, [trip, members, me])

  if (tripQ.isPending || (trip && membersQ.isPending)) {
    return (
      <Page>
        <Spinner />
      </Page>
    )
  }
  if (tripQ.error || membersQ.error || !trip || !members) {
    const notFound = tripQ.error instanceof ApiError && tripQ.error.status === 404
    return (
      <Page>
        <div className="space-y-4 pt-16 text-center">
          <p className="text-4xl">🧳</p>
          {notFound ? <p className="text-lg font-medium">ไม่พบทริปนี้ ลองเช็กลิงก์เชิญอีกครั้ง</p> : <ErrorText error={tripQ.error ?? membersQ.error} />}
          <Link to="/" className="inline-block text-teal-700 underline">
            กลับหน้าแรก
          </Link>
        </div>
      </Page>
    )
  }

  if (!ctx) {
    return (
      <PickMemberPage
        trip={trip}
        members={members}
        onPicked={(id) => {
          setMyMemberId(trip.id, id)
          setMyId(id)
        }}
      />
    )
  }

  return <Outlet context={ctx} />
}
