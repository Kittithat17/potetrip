import { createContext, useContext, type ReactNode } from 'react'

export type ConfirmOptions = {
  title: string
  message?: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  tone?: 'danger' | 'primary'
}

export type ConfirmFn = (opts: ConfirmOptions) => Promise<boolean>

export const ConfirmContext = createContext<ConfirmFn | null>(null)

/** `await confirm({ title })` -> true if the user confirmed. Needs <ConfirmProvider>. */
export function useConfirm(): ConfirmFn {
  const ctx = useContext(ConfirmContext)
  if (!ctx) throw new Error('useConfirm must be used inside <ConfirmProvider>')
  return ctx
}
