const BASE_URL = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

type Options = Omit<RequestInit, 'body'> & { body?: unknown; memberId?: string | null }

export async function apiFetch<T>(path: string, opts: Options = {}): Promise<T> {
  const { body, memberId, headers, ...rest } = opts
  const h = new Headers(headers)
  if (body !== undefined && !(body instanceof FormData)) h.set('Content-Type', 'application/json')
  if (memberId) h.set('X-Member-Id', memberId)

  let res: Response
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      ...rest,
      headers: h,
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'เชื่อมต่อเซิร์ฟเวอร์ไม่ได้')
  }

  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new ApiError(
      res.status,
      data?.error?.code ?? 'http_error',
      data?.error?.message ?? `HTTP ${res.status}`,
    )
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export type Health = { status: string }

export const getHealth = () => apiFetch<Health>('/healthz')
