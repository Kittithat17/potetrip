import { BedDouble, CarTaxiFront, Pin, ShoppingBag, TicketCheck, Utensils, type LucideIcon } from 'lucide-react'
import type { Category } from '../../../api/types'
import type { MsgKey } from '../../../i18n/context'

export const CATEGORIES: { value: Category; labelKey: MsgKey; icon: LucideIcon; tint: string }[] = [
  { value: 'food', labelKey: 'category.food', icon: Utensils, tint: 'bg-orange-100 text-orange-600' },
  { value: 'transport', labelKey: 'category.transport', icon: CarTaxiFront, tint: 'bg-sky-100 text-sky-600' },
  { value: 'lodging', labelKey: 'category.lodging', icon: BedDouble, tint: 'bg-violet-100 text-violet-600' },
  { value: 'shopping', labelKey: 'category.shopping', icon: ShoppingBag, tint: 'bg-pink-100 text-pink-600' },
  { value: 'activity', labelKey: 'category.activity', icon: TicketCheck, tint: 'bg-emerald-100 text-emerald-600' },
  { value: 'other', labelKey: 'category.other', icon: Pin, tint: 'bg-slate-200 text-slate-600' },
]

export const categoryInfo = (c: Category) => CATEGORIES.find((x) => x.value === c) ?? CATEGORIES[5]
