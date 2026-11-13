import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import { ConfirmProvider } from '../components/feedback/ConfirmProvider'
import { I18nProvider } from '../i18n/I18nProvider'
// Served from our own origin: Google Fonts is blocked in mainland China and
// would stall page loads there.
import '@fontsource/noto-sans-thai/400.css'
import '@fontsource/noto-sans-thai/500.css'
import '@fontsource/noto-sans-thai/700.css'
import '../index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <I18nProvider>
      <ConfirmProvider>
        <App />
      </ConfirmProvider>
    </I18nProvider>
  </StrictMode>,
)
