import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfirmProvider } from '../../components/ConfirmProvider'
import { I18nProvider } from '../../i18n/I18nProvider'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { ScanPage } from './ScanPage'
import { TripLayout } from './TripLayout'

const trip = { id: 't1', name: 'โตเกียว', base_currency: 'THB', invite_code: 'ABCDEFGH23', owner_member_id: 'm1', created_at: '' }
const members = [
  { id: 'm1', trip_id: 't1', name: 'แบงก์', promptpay: null, created_at: '' },
  { id: 'm2', trip_id: 't1', name: 'ฝน', promptpay: null, created_at: '' },
]
const parsed = {
  filename: 'a.jpg', slip_id: 's1', status: 'parsed',
  result: { kind: 'receipt', amount: '1250.50', currency: 'THB', datetime: '2026-11-10T19:30:00+07:00', from_name: '', to_name: '',
    merchant: 'Ichiran', reference: '', category: 'food', items: [], confidence: 0.95 },
  suggestion: { title: 'Ichiran', amount: 125050, currency: 'THB', spent_at: '2026-11-10T12:30:00Z', category: 'food', low_confidence: false },
}
const failed = { filename: 'b.jpg', slip_id: 's2', status: 'failed', error: 'อ่านสลิปไม่สำเร็จ ลองใหม่หรือกรอกเอง' }

const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status })
let posted: Record<string, unknown>[] = []

beforeEach(() => {
  localStorage.clear()
  posted = []
  let uploads = 0
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
    const url = String(input)
    if (url.endsWith('/trips/ABCDEFGH23')) return json({ trip, members })
    if (url.endsWith('/me')) return json({ status: 'approved', member: members[0], is_owner: true, request: {} })
    if (url.endsWith('/members')) return json(members)
    if (url.endsWith('/join-requests')) return json([])
    if (url.endsWith('/slips')) return json({ results: [uploads++ === 0 ? parsed : failed] })
    if (url.endsWith('/expenses') && init?.method === 'POST') {
      posted.push(JSON.parse(String(init.body)))
      return json({ id: 'e1' }, 201)
    }
    return json([])
  })
})
afterEach(() => vi.restoreAllMocks())

it('reads slips, then requires a decision on each before saving', async () => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const { container } = render(
    <I18nProvider initialLang="th">
      <ConfirmProvider>
      <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/t/ABCDEFGH23/scan']}>
        <Routes>
          <Route path="/t/:code" element={<TripLayout />}>
            <Route path="scan" element={<ScanPage />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
      </ConfirmProvider>
    </I18nProvider>,
  )

  await screen.findByRole('button', { name: /เลือกรูป/ })
  const input = container.querySelector('input[type=file]') as HTMLInputElement
  const files = [new File(['a'], 'a.jpg', { type: 'image/jpeg' }), new File(['b'], 'b.jpg', { type: 'image/jpeg' })]
  fireEvent.change(input, { target: { files } })

  expect(await screen.findByText(/อ่านได้ ✓/)).toBeInTheDocument()
  expect(await screen.findByText(/อ่านสลิปไม่สำเร็จ/)).toBeInTheDocument()
  expect(posted).toHaveLength(0) // nothing is saved automatically

  fireEvent.click(await screen.findByRole('button', { name: 'ตรวจสอบ 2 รายการ' }))

  expect(await screen.findByText('ตรวจสอบสลิป 1/2')).toBeInTheDocument()
  expect(screen.getByDisplayValue('Ichiran')).toBeInTheDocument()
  expect(screen.getByDisplayValue('1250.50')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'บันทึก' }))

  expect(await screen.findByText('ตรวจสอบสลิป 2/2')).toBeInTheDocument()
  expect(posted).toHaveLength(1)
  expect(posted[0]).toMatchObject({ title: 'Ichiran', amount: 125050, category: 'food', slip_id: 's1', paid_by: 'm1' })

  fireEvent.click(screen.getByRole('button', { name: /ข้าม/ }))
  expect(await screen.findByText('บันทึกแล้ว 1 จาก 2 รายการ')).toBeInTheDocument()
  expect(posted).toHaveLength(1)
})
