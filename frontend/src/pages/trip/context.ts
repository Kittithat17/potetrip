import { useOutletContext } from 'react-router'
import type { Member, Trip } from '../../api/types'
import type { NavItem } from '../../components/shell'

export type TripCtx = {
  trip: Trip
  members: Member[]
  me: Member
  token: string // this device's secret for the trip; send with every request
  isOwner: boolean
  memberName: (id: string) => string
  base: string // route prefix, e.g. /t/ABCDEFGH23
  leave: () => void // forget this device's access (rejoining needs approval)
  nav: NavItem[] // bottom tabs for pages inside this trip
}

export const useTrip = () => useOutletContext<TripCtx>()
