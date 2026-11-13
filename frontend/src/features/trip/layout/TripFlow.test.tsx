import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfirmProvider } from '../../../components/feedback/ConfirmProvider'
import { I18nProvider } from '../../../i18n/I18nProvider'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { AuthContext } from '../../auth/context'
import { DashboardPage } from '../dashboard/DashboardPage'
import { TripLayout } from './TripLayout'

const trip = { id: 't1', name: 'โตเกียว', base_currency: 'THB', invite_code: 'ABCDEFGH23', owner_member_id: 'm2', created_at: '' }
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
const request = { id: 'r1', status: 'pending', requested_name: 'แบงก์', requested_member_id: null, created_at: '' }

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status })

// Server state the mock serves; tests tweak it.
let meStatus: 'none' | 'pending' | 'approved' = 'approved'
let meMember = members[0]
let calls: { url: string; init?: RequestInit }[] = []

beforeEach(() => {
  localStorage.clear()
  meStatus = 'approved'
  meMember = members[0]
  calls = []
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
    const url = String(input)
    calls.push({ url, init })
    if (url.endsWith('/trips/ABCDEFGH23')) return json({ trip, members: members.map(({ id, name }) => ({ id, name })) })
    if (url.endsWith('/join')) {
      meStatus = 'pending'
      return json({ request }, 201)
    }
    if (url.endsWith('/me') && meStatus === 'none') return json({ error: { code: 'not_a_member', message: 'x' } }, 403)
    if (url.endsWith('/me'))
      return json({ status: meStatus, request, member: meStatus === 'approved' ? meMember : null, is_owner: meMember.id === 'm2' })
    if (url.endsWith('/members')) return json(members)
    if (url.endsWith('/expenses')) return json(expenses)
    if (url.endsWith('/balances')) return json(balances)
    if (url.endsWith('/join-requests')) return json([request])
    if (url.endsWith('/approve')) return json(members[0])
    return json({ error: { code: 'trip_not_found', message: 'ไม่พบทริปนี้' } }, 404)
  })
})
afterEach(() => vi.restoreAllMocks())

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider initialLang="th">
      <ConfirmProvider>
      <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <AuthContext.Provider value={{ user: { id: 'u1', name: 'แบงก์ Google', email: 'b@example.com', picture: null, has_google: true }, logout: async () => {} }}>
        <Routes>
          <Route path="/t/:code" element={<TripLayout />}>
            <Route index element={<DashboardPage />} />
          </Route>
        </Routes>
        </AuthContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>
      </ConfirmProvider>
    </I18nProvider>,
  )
}

it('a newcomer can only request to join and sees nothing until approved', async () => {
  meStatus = 'none'
  renderAt('/t/abcdefgh23')

  expect(await screen.findByText('ขอเข้าทริปในชื่อ')).toBeInTheDocument()
  // The name starts as the account name.
  expect(screen.getByPlaceholderText('ชื่อเล่น')).toHaveValue('แบงก์ Google')
  fireEvent.change(screen.getByPlaceholderText('ชื่อเล่น'), { target: { value: 'แบงก์' } })
  fireEvent.click(screen.getByRole('button', { name: 'ส่งคำขอเข้าทริป' }))

  expect(await screen.findByText('รอเจ้าของทริปอนุมัติ')).toBeInTheDocument()
  expect(calls.some((c) => c.url.endsWith('/expenses'))).toBe(false)
  // Nothing secret is kept in the browser any more.
  expect(Object.keys(localStorage).filter((k) => k.includes('device'))).toEqual([])
  const join = calls.find((c) => c.url.endsWith('/join'))!
  expect(new Headers(join.init?.headers).get('X-Requested-With')).toBe('travel-split')
})

it('an approved member sees the dashboard', async () => {
  renderAt('/t/ABCDEFGH23')

  expect(await screen.findByText(/สวัสดี แบงก์/)).toBeInTheDocument()
  expect(await screen.findByText('คุณต้องจ่ายเพื่อน')).toBeInTheDocument()
  expect(screen.getByText('ราเมง')).toBeInTheDocument()
  // Non-owners never ask for join requests.
  expect(calls.some((c) => c.url.endsWith('/join-requests'))).toBe(false)
})

it('the owner sees pending requests and can approve them', async () => {
  meMember = members[1]
  renderAt('/t/ABCDEFGH23')

  expect(await screen.findByText('คำขอเข้าทริป (1)')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'อนุมัติ แบงก์' }))
  await screen.findByText(/สวัสดี ฝน/)
  await vi.waitFor(() => expect(calls.some((c) => c.url.endsWith('/join-requests/r1/approve'))).toBe(true))
})

it('shows not found for an unknown invite code', async () => {
  renderAt('/t/ZZZZZZZZZZ')
  expect(await screen.findByText(/ไม่พบทริปนี้/)).toBeInTheDocument()
})
