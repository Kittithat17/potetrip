import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { I18nProvider } from '../../i18n/I18nProvider'
import { render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { HealthBadge } from './HealthBadge'

afterEach(() => vi.restoreAllMocks())

function renderBadge() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider initialLang="th">
      <QueryClientProvider client={qc}>
      <HealthBadge />
    </QueryClientProvider>
    </I18nProvider>,
  )
}

it('shows ready when /healthz is ok', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{"status":"ok"}', { status: 200 }))
  renderBadge()
  expect(await screen.findByText('เซิร์ฟเวอร์พร้อมใช้งาน')).toBeInTheDocument()
})

it('shows not ready when /healthz fails', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response('{"error":{"code":"db_unavailable","message":"x"}}', { status: 503 }),
  )
  renderBadge()
  expect(await screen.findByText('เซิร์ฟเวอร์ไม่พร้อม')).toBeInTheDocument()
})
