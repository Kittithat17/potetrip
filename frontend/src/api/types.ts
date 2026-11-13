export type Trip = {
  id: string
  name: string
  base_currency: string
  invite_code: string
  owner_member_id: string | null
  created_at: string
}

/** What anyone with the invite code can see about a member. */
export type PublicMember = { id: string; name: string }

export type JoinStatus = 'pending' | 'approved' | 'rejected'

export type JoinRequest = {
  id: string
  status: JoinStatus
  requested_name: string
  requested_member_id: string | null // set when claiming an existing name
  created_at: string
}

export type Me = { status: JoinStatus; request: JoinRequest; member: Member | null; is_owner: boolean }

export type User = { id: string; email: string; name: string; picture: string | null }

/** A trip the signed-in user is in (or waiting to join). */
export type MyTrip = {
  id: string
  name: string
  invite_code: string
  base_currency: string
  status: JoinStatus
  member_name: string | null
  is_owner: boolean
}

export type Member = {
  id: string
  trip_id: string
  name: string
  promptpay: string | null
  created_at: string
}

export type Category = 'food' | 'transport' | 'lodging' | 'shopping' | 'activity' | 'other'
export type SplitType = 'equal' | 'exact' | 'items'

export type Share = { member_id: string; amount_base: number }

/** A receipt line for split_type=items; amount is in the expense currency. */
export type ExpenseItem = { name: string; amount: number; member_ids: string[] }

export type Expense = {
  id: string
  trip_id: string
  title: string
  category: Category
  paid_by: string
  amount: number
  currency: string
  fx_rate: string
  amount_base: number
  split_type: SplitType
  spent_at: string
  slip_id: string | null
  created_by: string
  created_at: string
  shares: Share[]
  items: ExpenseItem[]
}

export type ExpenseInput = {
  title: string
  category: Category
  paid_by: string
  amount: number
  currency?: string
  split_type: SplitType
  spent_at?: string
  member_ids?: string[]
  shares?: { member_id: string; amount: number }[]
  items?: ExpenseItem[]
  slip_id?: string
  fx_rate?: string // own rate (base per 1 currency); omitted = reference rate for the day
}

export type Summary = {
  currency: string
  total: number
  count: number
  by_category: { category: Category; amount_base: number; count: number }[]
  by_currency: { currency: string; amount: number; amount_base: number; count: number }[]
  by_member: { member_id: string; paid: number; consumed: number }[]
}

export type Balance = {
  member_id: string
  paid: number
  owed: number
  sent: number
  received: number
  net: number
}

export type Transfer = { from: string; to: string; amount_base: number }

export type SlipResult = {
  kind: 'transfer_slip' | 'receipt' | 'unknown'
  amount: string
  currency: string
  datetime: string
  from_name: string
  to_name: string
  merchant: string
  reference: string
  category: string
  items: { name: string; amount: string }[]
  confidence: number
}

export type SlipSuggestion = {
  title: string // merchant or transfer receiver; see title_kind
  title_kind: 'merchant' | 'transfer_to' | 'none'
  amount: number | null
  currency: string
  spent_at: string | null
  category: Category
  low_confidence: boolean
  items: { name: string; amount: number }[]
}

export type SlipOutcome = {
  filename: string
  slip_id: string | null
  status: 'parsed' | 'failed'
  error_code?: string
  error?: string
  result?: SlipResult
  suggestion?: SlipSuggestion
}
