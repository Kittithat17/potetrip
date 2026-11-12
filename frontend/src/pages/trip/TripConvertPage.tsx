import { ConvertPage } from '../ConvertPage'
import { useTrip } from './context'

/** The converter inside a trip: keeps the trip's tabs and converts into its base currency. */
export function TripConvertPage() {
  const { trip, nav } = useTrip()
  return <ConvertPage nav={nav} defaultTo={trip.base_currency} />
}
