import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router'
import { retryUnlessClientError } from './api/hooks'
import { HomePage } from './pages/HomePage'
import { DashboardPage } from './pages/trip/DashboardPage'
import { ExpenseFormPage } from './pages/trip/ExpenseFormPage'
import { ScanPage } from './pages/trip/ScanPage'
import { SettlePage } from './pages/trip/SettlePage'
import { TripLayout } from './pages/trip/TripLayout'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 5_000, retry: retryUnlessClientError } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/t/:code" element={<TripLayout />}>
            <Route index element={<DashboardPage />} />
            <Route path="expenses/new" element={<ExpenseFormPage />} />
            <Route path="expenses/:id" element={<ExpenseFormPage />} />
            <Route path="scan" element={<ScanPage />} />
            <Route path="settle" element={<SettlePage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
