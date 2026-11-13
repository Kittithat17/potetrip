import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { getAuthConfig, googleSignInUrl, mockLogin } from '../../api/endpoints'
import { Button, ErrorText, TextInput } from '../../components/ui/ui'
import { useI18n } from '../../i18n/context'
import { LoginBackdrop } from './LoginBackdrop'
import { isPasskeyCancel, passkeysSupported, signInWithPasskey, signUpWithPasskey } from './passkey'

const ERRORS = ['cancelled', 'expired', 'google', 'server'] as const

/** Reads (and removes) the ?login_error=… the sign-in callback sends us back with. */
function takeLoginError(): string | null {
  const url = new URL(window.location.href)
  const err = url.searchParams.get('login_error')
  if (!err) return null
  url.searchParams.delete('login_error')
  window.history.replaceState(window.history.state, '', url.pathname + url.search + url.hash)
  return (ERRORS as readonly string[]).includes(err) ? err : 'server'
}

export function LoginPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [loginError] = useState(takeLoginError)
  const [leaving, setLeaving] = useState(false)
  const cfg = useQuery({ queryKey: ['auth', 'config'], queryFn: getAuthConfig, staleTime: Infinity })
  // Come back to the page that was opened (e.g. an invite link) after signing in.
  const next = window.location.pathname + window.location.search

  const signInWithGoogle = () => {
    setLeaving(true)
    window.location.assign(googleSignInUrl(next))
  }

  return (
    <LoginBackdrop>
      <div className="mb-8 flex items-center gap-3">
        <img src="/favicon.svg" alt="" className="size-11 rounded-xl shadow-sm" />
        <span className="text-2xl font-bold tracking-tight text-brand-700">Travel Split</span>
      </div>

      <main className="w-full max-w-sm rounded-3xl bg-white px-6 py-8 text-center shadow-lg shadow-slate-300/40 ring-1 ring-slate-200/70">
        <h1 className="text-2xl font-bold text-slate-900">{t('login.title')}</h1>
        <p className="mt-2 text-slate-500">{t('login.subtitle')}</p>

        {loginError && (
          <p role="alert" className="mt-5 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
            {t(`login.error.${loginError as (typeof ERRORS)[number]}`)}
          </p>
        )}

        <div className="mt-6">
          {cfg.isPending ? (
            <div className="h-12 animate-pulse rounded-xl bg-slate-100" />
          ) : cfg.error ? (
            <ErrorText error={cfg.error} />
          ) : cfg.data.provider === 'mock' ? (
            <MockLogin onDone={() => qc.resetQueries({ queryKey: ['auth', 'me'] })} />
          ) : (
            <button
              type="button"
              onClick={signInWithGoogle}
              disabled={leaving}
              className="flex h-12 w-full items-center justify-center gap-3 rounded-xl bg-white font-medium text-slate-800 ring-1 ring-slate-300 transition-colors hover:bg-slate-50 disabled:opacity-60"
            >
              <GoogleG />
              {t('login.google')}
            </button>
          )}
        </div>

        {cfg.data?.passkeys && passkeysSupported() && <PasskeyLogin onDone={() => qc.resetQueries({ queryKey: ['auth', 'me'] })} />}

        <p className="mt-6 text-xs text-slate-400">{t('login.privacy')}</p>
      </main>
    </LoginBackdrop>
  )
}

/**
 * Sign in with a passkey, or create an account with one. Works without
 * Google, e.g. in mainland China.
 */
function PasskeyLogin({ onDone }: { onDone: () => void }) {
  const { t } = useI18n()
  const [signingUp, setSigningUp] = useState(false)
  const [name, setName] = useState('')
  const [cancelled, setCancelled] = useState(false)
  const run = useMutation({
    mutationFn: (action: () => Promise<unknown>) => action(),
    onMutate: () => setCancelled(false),
    onSuccess: onDone,
    onError: (err) => setCancelled(isPasskeyCancel(err)),
  })
  const signUp = (e: FormEvent) => {
    e.preventDefault()
    run.mutate(() => signUpWithPasskey(name))
  }

  return (
    <div className="mt-4 space-y-3">
      <div className="flex items-center gap-3 text-xs text-slate-400" aria-hidden>
        <span className="h-px flex-1 bg-slate-200" />
        {t('login.or')}
        <span className="h-px flex-1 bg-slate-200" />
      </div>
      {signingUp ? (
        <form onSubmit={signUp} className="space-y-3 text-left">
          <p className="text-sm text-slate-600">{t('login.signupHint')}</p>
          <TextInput
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('login.signupName')}
            aria-label={t('login.signupName')}
            required
            maxLength={50}
            autoFocus
          />
          <Button type="submit" className="w-full" disabled={run.isPending}>
            <KeyRound className="size-4" /> {t('login.signupSubmit')}
          </Button>
          <button type="button" onClick={() => setSigningUp(false)} className="block w-full text-center text-sm text-slate-500">
            {t('common.back')}
          </button>
        </form>
      ) : (
        <>
          <button
            type="button"
            onClick={() => run.mutate(signInWithPasskey)}
            disabled={run.isPending}
            className="flex h-12 w-full items-center justify-center gap-3 rounded-xl bg-white font-medium text-slate-800 ring-1 ring-slate-300 transition-colors hover:bg-slate-50 disabled:opacity-60"
          >
            <KeyRound className="size-5 text-brand-600" />
            {t('login.passkey')}
          </button>
          <p className="text-xs text-slate-500">{t('login.passkeyHint')}</p>
          <button type="button" onClick={() => setSigningUp(true)} className="text-sm font-medium text-brand-700">
            {t('login.signup')}
          </button>
        </>
      )}
      {cancelled ? (
        <p className="text-sm text-slate-500">{t('login.passkeyCancelled')}</p>
      ) : (
        <ErrorText error={run.error} />
      )}
    </div>
  )
}

/** Dev-only stand-in for Google (AUTH_PROVIDER=mock): sign in by typing a name. */
function MockLogin({ onDone }: { onDone: () => void }) {
  const { t } = useI18n()
  const [name, setName] = useState('')
  const login = useMutation({ mutationFn: () => mockLogin(name), onSuccess: onDone })
  const submit = (e: FormEvent) => {
    e.preventDefault()
    login.mutate()
  }
  return (
    <form onSubmit={submit} className="space-y-3 text-left">
      <p className="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">{t('login.mockHint')}</p>
      <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder={t('login.mockName')} aria-label={t('login.mockName')} required maxLength={50} />
      <ErrorText error={login.error} />
      <Button type="submit" className="w-full" disabled={login.isPending}>
        {t('login.mockSubmit')}
      </Button>
    </form>
  )
}

/** Google's "G" mark, as its sign-in branding guidelines ask for. */
function GoogleG() {
  return (
    <svg viewBox="0 0 48 48" className="size-5" aria-hidden>
      <path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z" />
      <path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z" />
      <path fill="#FBBC05" d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z" />
      <path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z" />
    </svg>
  )
}
