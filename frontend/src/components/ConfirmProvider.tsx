import { AlertTriangle } from 'lucide-react'
import { useCallback, useId, useRef, useState, type ReactNode } from 'react'
import { useI18n } from '../i18n/context'
import { ConfirmContext, type ConfirmOptions } from '../lib/confirm'
import { Dialog } from './Dialog'
import { Button } from './ui'

/** Renders one confirmation dialog at a time for useConfirm() callers. */
export function ConfirmProvider({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  const [opts, setOpts] = useState<ConfirmOptions | null>(null)
  const [open, setOpen] = useState(false)
  const resolver = useRef<((ok: boolean) => void) | null>(null)
  const titleId = useId()
  const descId = useId()

  const confirm = useCallback((o: ConfirmOptions) => {
    resolver.current?.(false) // a newer request replaces an unanswered one
    setOpts(o)
    setOpen(true)
    return new Promise<boolean>((resolve) => {
      resolver.current = resolve
    })
  }, [])

  const settle = useCallback((ok: boolean) => {
    resolver.current?.(ok)
    resolver.current = null
    setOpen(false)
  }, [])

  const danger = opts?.tone !== 'primary'
  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      <Dialog open={open} onClose={() => settle(false)} role="alertdialog" labelledBy={titleId} describedBy={descId}>
        {opts && (
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              {danger && (
                <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-red-50 text-red-600">
                  <AlertTriangle className="size-5" />
                </span>
              )}
              <div className="min-w-0 pt-1">
                <h2 id={titleId} className="text-lg leading-snug font-bold">
                  {opts.title}
                </h2>
                {opts.message && (
                  <div id={descId} className="mt-1 text-sm text-slate-600">
                    {opts.message}
                  </div>
                )}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Button variant="secondary" onClick={() => settle(false)} data-autofocus>
                {opts.cancelLabel ?? t('common.cancel')}
              </Button>
              <Button variant={danger ? 'dangerSolid' : 'primary'} onClick={() => settle(true)}>
                {opts.confirmLabel ?? t('common.confirm')}
              </Button>
            </div>
          </div>
        )}
      </Dialog>
    </ConfirmContext.Provider>
  )
}
