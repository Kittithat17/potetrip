import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { I18nProvider } from '../i18n/I18nProvider'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { mainNav } from '../lib/nav'
import { ConvertPage } from './ConvertPage'

beforeEach(() => {
  localStorage.clear()
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input)
    if (url.includes('/fx/currencies')) return new Response(JSON.stringify({ currencies: ['CNY', 'JPY', 'THB'] }))
    const u = new URL(url, 'http://x')
    const from = u.searchParams.get('from')
    const to = u.searchParams.get('to')
    const rate = from === 'CNY' && to === 'THB' ? '5.0505' : from === 'THB' && to === 'CNY' ? '0.198' : '1'
    return new Response(JSON.stringify({ from, to, rate, rate_date: '2026-11-11', requested_date: '2026-11-12', source: 'x', stale: false }))
  })
})
afterEach(() => vi.restoreAllMocks())

function renderPage(defaultTo?: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider initialLang="th">
      <QueryClientProvider client={qc}>
      <MemoryRouter>
        <ConvertPage nav={mainNav} defaultTo={defaultTo} />
      </MemoryRouter>
    </QueryClientProvider>
    </I18nProvider>,
  )
}

it('converts as you type, swaps, and remembers the pair', async () => {
  renderPage()
  const input = screen.getByLabelText('จำนวนเงิน')
  fireEvent.change(input, { target: { value: '10' } })
  expect(await screen.findByText('50.51')).toBeInTheDocument() // 10 CNY ≈ 50.51 THB

  fireEvent.click(screen.getByRole('button', { name: 'สลับสกุลเงิน' }))
  expect(await screen.findByText('1.98')).toBeInTheDocument() // 10 THB ≈ 1.98 CNY
  expect(JSON.parse(localStorage.getItem('travel-split:convert-pair')!)).toEqual(['THB', 'CNY'])
})

it('uses the trip base currency as the target', async () => {
  renderPage('THB')
  expect(await screen.findByText('505.05')).toBeInTheDocument() // default 100 CNY
})
