export type Trip = {
  id: string
  name: string
  base_currency: string
  invite_code: string
  created_at: string
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
  slip_id?: string
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
  title: string
  amount: number | null
  currency: string
  spent_at: string | null
  category: Category
  low_confidence: boolean
}

export type SlipOutcome = {
  filename: string
  slip_id: string | null
  status: 'parsed' | 'failed'
  error?: string
  result?: SlipResult
  suggestion?: SlipSuggestion
}
