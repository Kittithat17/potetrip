import { HealthBadge } from '../components/HealthBadge'

export function HomePage() {
  return (
    <main className="mx-auto flex min-h-dvh max-w-md flex-col gap-6 px-4 py-10">
      <header>
        <h1 className="text-2xl font-bold text-teal-800">Travel Split</h1>
        <p className="mt-1 text-slate-600">หารค่าใช้จ่ายทริปกับเพื่อน</p>
      </header>
      <HealthBadge />
    </main>
  )
}
