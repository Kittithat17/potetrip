import type { ReactNode } from 'react'

/** Soft tinted backdrop with a faint dot grid, behind the sign-in card. */
export function LoginBackdrop({ children }: { children: ReactNode }) {
  return (
    <div className="relative flex min-h-dvh flex-col items-center justify-center overflow-hidden bg-gradient-to-br from-sky-100 via-indigo-50 to-emerald-50 px-4 py-[calc(env(safe-area-inset-top)+2rem)]">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-60"
        style={{ backgroundImage: 'radial-gradient(rgb(148 163 184 / 0.35) 1px, transparent 1px)', backgroundSize: '22px 22px' }}
      />
      <div className="relative flex w-full flex-col items-center">{children}</div>
    </div>
  )
}
