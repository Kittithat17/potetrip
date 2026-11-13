import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router'
import { retryUnlessClientError } from '../api/hooks'
import { AuthGate } from '../features/auth/AuthGate'
import { ConvertPage } from '../features/convert/ConvertPage'
import { HomePage } from '../features/home/HomePage'
import { MyTripsPage } from '../features/home/MyTripsPage'
import { NewTripPage } from '../features/home/NewTripPage'
import { SettingsPage } from '../features/settings/SettingsPage'
import { DashboardPage } from '../features/trip/dashboard/DashboardPage'
import { ExpenseFormPage } from '../features/trip/expenses/ExpenseFormPage'
import { ScanPage } from '../features/trip/scan/ScanPage'
import { SummaryPage } from '../features/trip/summary/SummaryPage'
import { SettlePage } from '../features/trip/settle/SettlePage'
import { TripConvertPage } from '../features/trip/convert/TripConvertPage'
import { TripSettingsPage } from '../features/trip/settings/TripSettingsPage'
import { mainNav } from './nav'
import { TripLayout } from '../features/trip/layout/TripLayout'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 5_000, retry: retryUnlessClientError } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthGate>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/new" element={<NewTripPage />} />
          <Route path="/trips" element={<MyTripsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/convert" element={<ConvertPage nav={mainNav} />} />
          <Route path="/t/:code" element={<TripLayout />}>
            <Route index element={<DashboardPage />} />
            <Route path="expenses/new" element={<ExpenseFormPage />} />
            <Route path="expenses/:id" element={<ExpenseFormPage />} />
            <Route path="scan" element={<ScanPage />} />
            <Route path="settle" element={<SettlePage />} />
            <Route path="summary" element={<SummaryPage />} />
            <Route path="convert" element={<TripConvertPage />} />
            <Route path="edit" element={<TripSettingsPage />} />
          </Route>
        </Routes>
        </AuthGate>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
