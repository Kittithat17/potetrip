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

/** `token` is the device's secret for the trip, sent as X-Device-Token. */
type Options = Omit<RequestInit, 'body'> & { body?: unknown; token?: string | null }

export async function apiFetch<T>(path: string, opts: Options = {}): Promise<T> {
  const { body, token, headers, ...rest } = opts
  const h = new Headers(headers)
  if (body !== undefined && !(body instanceof FormData)) h.set('Content-Type', 'application/json')
  if (token) h.set('X-Device-Token', token)

  let res: Response
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      ...rest,
      headers: h,
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'network_error', "Can't reach the server")
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
