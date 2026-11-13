import { useEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

const DURATION = 220 // ms; keep in sync with the duration-[220ms] classes below

type Props = {
  open: boolean
  onClose: () => void
  labelledBy?: string
  describedBy?: string
  role?: 'dialog' | 'alertdialog'
  children: ReactNode
}

/**
 * Animated modal: a bottom sheet on phones, a centred card on wider screens.
 * Only transform/opacity animate, so it stays smooth. It stays mounted until
 * the exit animation finishes, closes on Escape or backdrop tap, locks page
 * scroll, and gives focus back to whatever opened it.
 */
export function Dialog({ open, onClose, labelledBy, describedBy, role = 'dialog', children }: Props) {
  // `exiting` keeps the dialog on screen while its close animation runs.
  const [exiting, setExiting] = useState(false)
  const [entered, setEntered] = useState(false)
  const [prevOpen, setPrevOpen] = useState(open)
  if (open !== prevOpen) {
    setPrevOpen(open)
    setExiting(!open)
  }
  const mounted = open || exiting
  const shown = open && entered
  const panelRef = useRef<HTMLDivElement>(null)
  const returnFocus = useRef<HTMLElement | null>(null)

  // Enter: wait two frames so the hidden state paints first and the transition runs.
  useEffect(() => {
    if (!open) return
    returnFocus.current = document.activeElement as HTMLElement | null
    let id = requestAnimationFrame(() => {
      id = requestAnimationFrame(() => setEntered(true))
    })
    return () => cancelAnimationFrame(id)
  }, [open])

  // Exit: unmount once the animation has finished, then restore focus.
  useEffect(() => {
    if (!exiting) return
    const id = setTimeout(() => {
      setExiting(false)
      setEntered(false)
      returnFocus.current?.focus?.()
    }, DURATION)
    return () => clearTimeout(id)
  }, [exiting])

  useEffect(() => {
    if (!mounted) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.body.style.overflow = prev
      document.removeEventListener('keydown', onKey)
    }
  }, [mounted, onClose])

  // Focus the preferred button (or the panel) once visible.
  useEffect(() => {
    if (!shown) return
    const target = panelRef.current?.querySelector<HTMLElement>('[data-autofocus]') ?? panelRef.current
    target?.focus()
  }, [shown])

  if (!mounted) return null
  return createPortal(
    // While closing, let taps pass through to the page underneath.
    <div className={`fixed inset-0 z-50 flex items-end justify-center sm:items-center ${open ? '' : 'pointer-events-none'}`}>
      <div
        aria-hidden
        onClick={onClose}
        className={`absolute inset-0 bg-slate-900/45 backdrop-blur-[2px] transition-opacity duration-[220ms] ease-out motion-reduce:transition-none ${
          shown ? 'opacity-100' : 'opacity-0'
        }`}
      />
      <div
        ref={panelRef}
        role={role}
        aria-modal="true"
        aria-labelledby={labelledBy}
        aria-describedby={describedBy}
        tabIndex={-1}
        className={`relative w-full max-w-md rounded-t-3xl bg-white p-5 pb-[calc(env(safe-area-inset-bottom)+1.25rem)] shadow-2xl outline-none will-change-transform sm:mx-4 sm:rounded-3xl sm:pb-5
          transition-[transform,opacity] duration-[220ms] ease-[cubic-bezier(0.2,0.9,0.3,1)] motion-reduce:transition-none
          ${shown ? 'translate-y-0 opacity-100 sm:scale-100' : 'translate-y-full opacity-0 sm:translate-y-0 sm:scale-95'}`}
      >
        <span aria-hidden className="mx-auto mb-4 block h-1.5 w-10 rounded-full bg-slate-200 sm:hidden" />
        {children}
      </div>
    </div>,
    document.body,
  )
}
