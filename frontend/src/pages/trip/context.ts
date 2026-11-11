import { useOutletContext } from 'react-router'
import type { Member, Trip } from '../../api/types'

export type TripCtx = {
  trip: Trip
  members: Member[]
  me: Member
  memberName: (id: string) => string
  base: string // route prefix, e.g. /t/ABCDEFGH23
  switchUser: () => void
}

export const useTrip = () => useOutletContext<TripCtx>()
