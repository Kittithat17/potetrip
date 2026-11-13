import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import { ConfirmProvider } from '../components/feedback/ConfirmProvider'
import { I18nProvider } from '../i18n/I18nProvider'
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
