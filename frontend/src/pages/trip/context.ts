import { useOutletContext } from 'react-router'
import type { Member, Trip } from '../../api/types'
import type { NavItem } from '../../components/shell'

export type TripCtx = {
  trip: Trip
  members: Member[]
  me: Member
  isOwner: boolean
  memberName: (id: string) => string
  base: string // route prefix, e.g. /t/ABCDEFGH23
  leave: () => Promise<void> // leave the trip (rejoining needs approval)
  nav: NavItem[] // bottom tabs for pages inside this trip
}

export const useTrip = () => useOutletContext<TripCtx>()
