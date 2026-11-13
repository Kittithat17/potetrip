import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { AuthContext } from '../auth/context'
import { ConfirmProvider } from '../../components/feedback/ConfirmProvider'
import { I18nProvider } from '../../i18n/I18nProvider'
import { SettingsPage } from './SettingsPage'

beforeEach(() => {
  localStorage.clear()
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('{"status":"ok"}'))
})
afterEach(() => vi.restoreAllMocks())
const logout = vi.fn(async () => {})

function renderPage() {
  return render(
    <I18nProvider>
      <ConfirmProvider>
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <AuthContext.Provider value={{ user: { id: 'u1', name: 'Bank', email: 'bank@example.com', picture: null }, logout }}>
            <SettingsPage />
          </AuthContext.Provider>
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

it('shows the signed-in account and signs out after confirming', async () => {
  renderPage()
  expect(screen.getByText('Bank')).toBeInTheDocument()
  expect(screen.getByText('bank@example.com')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: /Sign out/ }))
  const dialog = await screen.findByRole('alertdialog', { name: 'Sign out' })
  fireEvent.click(within(dialog).getByRole('button', { name: 'Sign out' }))
  await vi.waitFor(() => expect(logout).toHaveBeenCalledOnce())
})
