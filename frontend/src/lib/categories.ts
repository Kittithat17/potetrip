import type { Category } from '../api/types'

export const CATEGORIES: { value: Category; label: string; icon: string }[] = [
  { value: 'food', label: 'อาหาร', icon: '🍜' },
  { value: 'transport', label: 'เดินทาง', icon: '🚕' },
  { value: 'lodging', label: 'ที่พัก', icon: '🏨' },
  { value: 'shopping', label: 'ช้อปปิ้ง', icon: '🛍️' },
  { value: 'activity', label: 'กิจกรรม', icon: '🎡' },
  { value: 'other', label: 'อื่นๆ', icon: '📌' },
]

export const categoryInfo = (c: Category) => CATEGORIES.find((x) => x.value === c) ?? CATEGORIES[5]
