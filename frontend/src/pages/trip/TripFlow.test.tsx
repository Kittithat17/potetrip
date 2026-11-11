import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { getMyMemberId } from '../../lib/identity'
import { getRecentTrips } from '../../lib/recentTrips'
import { DashboardPage } from './DashboardPage'
import { TripLayout } from './TripLayout'

const trip = { id: 't1', name: 'โตเกียว', base_currency: 'THB', invite_code: 'ABCDEFGH23', created_at: '2026-11-01T00:00:00Z' }
const members = [
  { id: 'm1', trip_id: 't1', name: 'แบงก์', promptpay: null, created_at: '' },
  { id: 'm2', trip_id: 't1', name: 'ฝน', promptpay: null, created_at: '' },
]
const expenses = [
  {
    id: 'e1', trip_id: 't1', title: 'ราเมง', category: 'food', paid_by: 'm2', amount: 60000, currency: 'THB',
    fx_rate: '1', amount_base: 60000, split_type: 'equal', spent_at: '2026-11-10T12:00:00Z', slip_id: null,
    created_by: 'm2', created_at: '', shares: [{ member_id: 'm1', amount_base: 30000 }, { member_id: 'm2', amount_base: 30000 }],
  },
]
const balances = {
  currency: 'THB',
  balances: [
    { member_id: 'm1', paid: 0, owed: 30000, sent: 0, received: 0, net: -30000 },
    { member_id: 'm2', paid: 60000, owed: 30000, sent: 0, received: 0, net: 30000 },
  ],
}

const json = (v: unknown) => new Response(JSON.stringify(v), { status: 200 })

beforeEach(() => {
  localStorage.clear()
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input)
    if (url.endsWith('/trips/ABCDEFGH23')) return json(trip)
    if (url.endsWith('/trips/t1/members')) return json(members)
    if (url.endsWith('/trips/t1/expenses')) return json(expenses)
    if (url.endsWith('/trips/t1/balances')) return json(balances)
    return new Response('{}', { status: 404 })
  })
})
afterEach(() => vi.restoreAllMocks())

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/t/:code" element={<TripLayout />}>
            <Route index element={<DashboardPage />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

it('asks who I am, remembers it, then shows my balance', async () => {
  renderAt('/t/abcdefgh23')

  expect(await screen.findByText('คุณคือใครในทริปนี้?')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: /แบงก์/ }))

  expect(await screen.findByText('สวัสดี แบงก์')).toBeInTheDocument()
  expect(getMyMemberId('t1')).toBe('m1')
  expect(getRecentTrips()).toMatchObject([{ id: 't1', code: 'ABCDEFGH23', name: 'โตเกียว', myName: 'แบงก์' }])
  expect(await screen.findByText('คุณต้องจ่ายเพื่อน')).toBeInTheDocument()
  expect(screen.getByText('฿300.00')).toBeInTheDocument()
  expect(screen.getByText('ราเมง')).toBeInTheDocument()

  // Every request after picking carries the member header.
  const calls = vi.mocked(fetch).mock.calls.filter(([u]) => String(u).includes('/expenses'))
  const init = calls[0][1] as RequestInit
  expect(new Headers(init.headers).get('X-Member-Id')).toBe('m1')
})

it('shows not found for an unknown invite code', async () => {
  vi.mocked(fetch).mockResolvedValue(
    new Response('{"error":{"code":"trip_not_found","message":"ไม่พบทริปนี้"}}', { status: 404 }),
  )
  renderAt('/t/ZZZZZZZZZZ')
  expect(await screen.findByText(/ไม่พบทริปนี้/)).toBeInTheDocument()
})
