import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from 'react'
import { Link } from 'react-router'
import { ApiError } from '../api/client'

type Variant = 'primary' | 'secondary' | 'danger' | 'ghost'

const variants: Record<Variant, string> = {
  primary: 'bg-teal-700 text-white hover:bg-teal-800 disabled:bg-teal-700/50',
  secondary: 'bg-white text-teal-800 ring-1 ring-teal-700/30 hover:bg-teal-50 disabled:opacity-50',
  danger: 'bg-white text-red-700 ring-1 ring-red-300 hover:bg-red-50 disabled:opacity-50',
  ghost: 'text-slate-600 hover:bg-slate-100 disabled:opacity-50',
}

export function Button({
  variant = 'primary',
  className = '',
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return (
    <button
      className={`inline-flex min-h-11 items-center justify-center gap-2 rounded-xl px-4 font-medium transition-colors disabled:cursor-not-allowed ${variants[variant]} ${className}`}
      {...props}
    />
  )
}

export function Card({ className = '', children }: { className?: string; children: ReactNode }) {
  return <section className={`rounded-2xl bg-white p-4 shadow-sm ring-1 ring-slate-200 ${className}`}>{children}</section>
}

export function Field({ label, hint, children }: { label: string; hint?: ReactNode; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-slate-700">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-slate-500">{hint}</span>}
    </label>
  )
}

export const inputBase =
  'block min-h-11 rounded-xl border-0 bg-white px-3 text-base ring-1 ring-slate-300 placeholder:text-slate-400 focus:ring-2 focus:ring-teal-600 focus:outline-none'
export const inputClass = `${inputBase} w-full`

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={`${inputClass} ${props.className ?? ''}`} />
}

export function ErrorText({ error }: { error: unknown }) {
  if (!error) return null
  const msg = error instanceof ApiError ? error.message : error instanceof Error ? error.message : String(error)
  return (
    <p role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
      {msg}
    </p>
  )
}

export function Spinner({ label = 'กำลังโหลด…' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-10 text-slate-500" role="status">
      <span className="size-4 animate-spin rounded-full border-2 border-teal-600 border-t-transparent" />
      {label}
    </div>
  )
}

export function PageHeader({ title, back, right }: { title: string; back?: string; right?: ReactNode }) {
  return (
    <header className="sticky top-0 z-10 -mx-4 mb-4 flex items-center gap-2 bg-slate-50/90 px-4 py-3 backdrop-blur">
      {back && (
        <Link to={back} className="-ml-2 rounded-lg px-2 py-1 text-xl text-slate-600 hover:bg-slate-200" aria-label="ย้อนกลับ">
          ‹
        </Link>
      )}
      <h1 className="min-w-0 flex-1 truncate text-lg font-bold">{title}</h1>
      {right}
    </header>
  )
}

export function Page({ children }: { children: ReactNode }) {
  return <main className="mx-auto min-h-dvh max-w-md px-4 pb-[calc(env(safe-area-inset-bottom)+2rem)]">{children}</main>
}

export function Avatar({ name, size = 'md' }: { name: string; size?: 'sm' | 'md' }) {
  const hue = [...name].reduce((h, c) => (h * 31 + c.codePointAt(0)!) % 360, 7)
  const cls = size === 'sm' ? 'size-7 text-xs' : 'size-10 text-base'
  return (
    <span
      aria-hidden
      className={`inline-flex shrink-0 items-center justify-center rounded-full font-bold text-white ${cls}`}
      style={{ backgroundColor: `hsl(${hue} 45% 45%)` }}
    >
      {initial(name)}
    </span>
  )
}

// Thai leading vowels (เ แ โ ใ ไ) come before the consonant they belong to;
// skip them so "แบงก์" shows "บ" rather than "แ".
function initial(name: string): string {
  const chars = [...name.trim()]
  return (chars.find((c) => !'เแโใไ'.includes(c)) ?? chars[0] ?? '?').toUpperCase()
}
