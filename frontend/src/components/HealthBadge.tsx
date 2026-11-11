import { useQuery } from '@tanstack/react-query'
import { getHealth } from '../api/client'

export function HealthBadge() {
  const { data, isPending, isError } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 10_000,
    retry: false,
  })

  const [label, cls] = isPending
    ? ['กำลังตรวจสอบเซิร์ฟเวอร์…', 'bg-slate-200 text-slate-600']
    : isError || data?.status !== 'ok'
      ? ['เซิร์ฟเวอร์ไม่พร้อม', 'bg-red-100 text-red-700']
      : ['เซิร์ฟเวอร์พร้อมใช้งาน', 'bg-emerald-100 text-emerald-700']

  return (
    <span role="status" className={`inline-flex items-center rounded-full px-3 py-1 text-sm ${cls}`}>
      {label}
    </span>
  )
}
