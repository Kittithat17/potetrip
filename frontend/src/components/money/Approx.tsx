import { useHomeApprox } from '../../lib/homeApprox'

/** Small "≈ ฿…" line for an amount in a foreign currency; renders nothing for THB. */
export function Approx({ minor, currency, className = 'text-xs text-slate-500' }: { minor: number; currency: string; className?: string }) {
  const text = useHomeApprox(currency).approx(minor)
  return text ? <span className={`tabular-nums ${className.includes('inline') ? '' : 'block '}${className}`}>{text}</span> : null
}
