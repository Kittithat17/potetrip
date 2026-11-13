import { createContext, useContext } from 'react'
import type { User } from '../api/types'

export type AuthCtx = { user: User; logout: () => Promise<void> }

export const AuthContext = createContext<AuthCtx | null>(null)

/** The signed-in user. Only usable inside AuthGate, which never renders signed out. */
export function useAuth(): AuthCtx {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth outside AuthGate')
  return ctx
}
