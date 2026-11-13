import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, type ReactNode } from 'react'
import { useNavigate } from 'react-router'
import { ApiError, SESSION_EXPIRED_EVENT } from '../api/client'
import { claimDevices, getCurrentUser, logout as apiLogout } from '../api/endpoints'
import { keys, retryUnlessClientError } from '../api/hooks'
import { Button, ErrorText, Spinner } from '../components/ui'
import { useI18n } from '../i18n/context'
import { clearLegacy, legacyTokens } from '../lib/legacy'
import { AuthContext, type AuthCtx } from './context'
import { LoginPage } from './LoginPage'
import { LoginBackdrop } from './LoginBackdrop'

const AUTH_ME = ['auth', 'me'] as const

/**
 * Renders the app only for a signed-in user; everyone else gets the sign-in
 * screen, whatever URL they opened (it brings them back there afterwards).
 * The server enforces the same rule on every API call.
 */
export function AuthGate({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const meQ = useQuery({ queryKey: AUTH_ME, queryFn: getCurrentUser, retry: retryUnlessClientError, staleTime: Infinity })
  const user = meQ.data

  // Drop every cached trip and ask the server who we are again.
  const signedOut = useMemo(
    () => async () => {
      qc.removeQueries({ predicate: (q) => q.queryKey[0] !== 'auth' })
      await qc.resetQueries({ queryKey: AUTH_ME })
    },
    [qc],
  )

  // Any request that finds the session gone (expired, signed out elsewhere).
  useEffect(() => {
    const on = () => void signedOut()
    window.addEventListener(SESSION_EXPIRED_EVENT, on)
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, on)
  }, [signedOut])

  // First sign-in on a browser that used the app before accounts existed:
  // move its trips onto the account, then wipe the old local tokens.
  const claimed = useRef(false)
  useEffect(() => {
    if (!user || claimed.current) return
    claimed.current = true
    const tokens = legacyTokens()
    if (tokens.length === 0) {
      clearLegacy()
      return
    }
    claimDevices(tokens)
      .then(() => {
        clearLegacy()
        return qc.invalidateQueries({ queryKey: keys.myTrips })
      })
      .catch(() => {
        claimed.current = false // keep the tokens; try again next time
      })
  }, [user, qc])

  const ctx = useMemo<AuthCtx | null>(
    () =>
      user
        ? {
            user,
            logout: async () => {
              await apiLogout()
              navigate('/', { replace: true })
              await signedOut()
            },
          }
        : null,
    [user, navigate, signedOut],
  )

  if (meQ.isPending) {
    return (
      <LoginBackdrop>
        <Spinner />
      </LoginBackdrop>
    )
  }
  if (meQ.error instanceof ApiError && meQ.error.status === 401) {
    return <LoginPage />
  }
  if (meQ.error || !ctx) {
    return (
      <LoginBackdrop>
        <div className="w-full max-w-sm space-y-3 rounded-3xl bg-white p-6 text-center shadow-lg ring-1 ring-slate-200/70">
          <ErrorText error={meQ.error} />
          <Button className="w-full" onClick={() => meQ.refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </LoginBackdrop>
    )
  }
  return <AuthContext.Provider value={ctx}>{children}</AuthContext.Provider>
}
