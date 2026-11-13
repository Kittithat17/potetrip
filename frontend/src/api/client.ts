export const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')

/**
 * Fired when the server says the session is gone (expired, logged out on
 * another tab), so the app can drop back to the sign-in screen.
 */
export const SESSION_EXPIRED_EVENT = 'travel-split:session-expired'

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

type Options = Omit<RequestInit, 'body'> & { body?: unknown }

/**
 * Calls the API. The session lives in an HttpOnly cookie the browser attaches
 * by itself; this code never sees it. X-Requested-With is the server's
 * anti-CSRF check: other sites can't add it to requests aimed at us.
 */
export async function apiFetch<T>(path: string, opts: Options = {}): Promise<T> {
  const { body, headers, ...rest } = opts
  const h = new Headers(headers)
  if (body !== undefined && !(body instanceof FormData)) h.set('Content-Type', 'application/json')
  h.set('X-Requested-With', 'travel-split')

  let res: Response
  try {
    res = await fetch(`${API_BASE_URL}${path}`, {
      ...rest,
      credentials: 'same-origin',
      headers: h,
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'network_error', "Can't reach the server")
  }

  if (!res.ok) {
    const data = await res.json().catch(() => null)
    if (res.status === 401 && data?.error?.code === 'login_required' && path !== '/auth/me') {
      window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT))
    }
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
