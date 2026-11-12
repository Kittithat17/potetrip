import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { ConfirmProvider } from '../components/ConfirmProvider'
import { I18nProvider } from '../i18n/I18nProvider'
import { SettingsPage } from './SettingsPage'

beforeEach(() => {
  localStorage.clear()
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{"status":"ok"}'))
})
afterEach(() => vi.restoreAllMocks())

function renderPage() {
  return render(
    <I18nProvider>
      <ConfirmProvider>
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <SettingsPage />
        </MemoryRouter>
      </QueryClientProvider>
      </ConfirmProvider>
    </I18nProvider>,
  )
}

it('defaults to English and switches to Thai, remembering the choice', async () => {
  renderPage()
  expect(screen.getByRole('heading', { name: 'Settings' })).toBeInTheDocument()
  expect(screen.getByText('Home')).toBeInTheDocument()

  fireEvent.click(screen.getByRole('button', { name: /ไทย/ }))
  expect(await screen.findByRole('heading', { name: 'ตั้งค่า' })).toBeInTheDocument()
  expect(screen.getByText('หน้าแรก')).toBeInTheDocument()
  expect(localStorage.getItem('travel-split:lang')).toBe('th')
  expect(document.documentElement.lang).toBe('th')
})
