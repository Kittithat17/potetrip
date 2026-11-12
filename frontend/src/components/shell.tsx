import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link, NavLink } from 'react-router'
import { useI18n } from '../i18n/context'

/** Blue gradient top area. Content placed after it can overlap via -mt-*. */
export function BlueHeader({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <header
      className={`bg-gradient-to-b from-brand-700 via-brand-600 to-brand-500 px-4 pt-[calc(env(safe-area-inset-top)+0.75rem)] text-white ${className}`}
    >
      <div className="mx-auto max-w-md">{children}</div>
    </header>
  )
}

/** White rounded sheet that slides up over the BlueHeader. */
export function Sheet({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <section className={`rounded-3xl bg-white p-4 shadow-sm ring-1 ring-slate-200/70 ${className}`}>{children}</section>
}

type ActionProps = { icon: LucideIcon; label: string; to?: string; onClick?: () => void; disabled?: boolean }

/** Round blue icon with a label underneath, for the quick-action grid. */
export function IconAction({ icon: Icon, label, to, onClick, disabled }: ActionProps) {
  const body = (
    <>
      <span className="flex size-14 items-center justify-center rounded-full bg-gradient-to-b from-brand-500 to-brand-700 text-white shadow-md shadow-brand-600/30">
        <Icon className="size-7" strokeWidth={2} />
      </span>
      <span className="text-center text-xs leading-tight font-medium text-slate-700">{label}</span>
    </>
  )
  const cls = `flex flex-col items-center gap-1.5 ${disabled ? 'opacity-40 pointer-events-none' : 'active:scale-95 transition-transform'}`
  return to ? (
    <Link to={to} className={cls}>
      {body}
    </Link>
  ) : (
    <button type="button" onClick={onClick} disabled={disabled} className={cls}>
      {body}
    </button>
  )
}

import type { NavItem } from '../lib/nav'
export type { NavItem }

export function BottomNav({ items }: { items: NavItem[] }) {
  const { t } = useI18n()
  return (
    <nav className="fixed inset-x-0 bottom-0 z-20 border-t border-slate-200 bg-white/95 pb-[env(safe-area-inset-bottom)] backdrop-blur">
      <ul className="mx-auto flex max-w-md">
        {items.map(({ to, labelKey, icon: Icon, end }) => (
          <li key={to} className="flex-1">
            <NavLink
              to={to}
              end={end}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 py-2 text-[11px] ${isActive ? 'text-brand-600 font-semibold' : 'text-slate-500'}`
              }
            >
              <Icon className="size-6" />
              {t(labelKey)}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}

/** Page frame with room for the bottom nav. */
export function AppFrame({ children, nav }: { children: ReactNode; nav?: NavItem[] }) {
  return (
    <div className={`min-h-dvh bg-slate-100 ${nav ? 'pb-24' : 'pb-8'}`}>
      {children}
      {nav && <BottomNav items={nav} />}
    </div>
  )
}

/** A colourful cover tile derived from the trip name, used where a photo would go. */
export function TripCover({ name, className = '' }: { name: string; className?: string }) {
  const hue = [...name].reduce((h, c) => (h * 31 + c.codePointAt(0)!) % 360, 11)
  const first = [...name.trim()].find((c) => !'เแโใไ'.includes(c)) ?? '✈'
  return (
    <div
      aria-hidden
      className={`relative flex items-center justify-center overflow-hidden text-white ${className}`}
      style={{ background: `linear-gradient(135deg, hsl(${hue} 80% 55%), hsl(${(hue + 40) % 360} 85% 45%))` }}
    >
      <span className="text-3xl font-bold drop-shadow">{first.toUpperCase()}</span>
      <span className="absolute -right-3 -bottom-3 text-5xl opacity-20">✈</span>
    </div>
  )
}
