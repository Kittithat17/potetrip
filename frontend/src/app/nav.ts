import { ArrowLeftRight, House, Settings, type LucideIcon } from 'lucide-react'
import type { MsgKey } from '../i18n/context'

export type NavItem = { to: string; labelKey: MsgKey; icon: LucideIcon; end?: boolean }

export const settingsTab: NavItem = { to: '/settings', labelKey: 'nav.settings', icon: Settings }

/** Bottom tabs outside a trip. */
export const mainNav: NavItem[] = [
  { to: '/', labelKey: 'nav.home', icon: House, end: true },
  { to: '/convert', labelKey: 'nav.convert', icon: ArrowLeftRight },
  settingsTab,
]
