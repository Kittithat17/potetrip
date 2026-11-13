import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, SESSION_EXPIRED_EVENT, apiFetch } from './client'

afterEach(() => vi.restoreAllMocks())

describe('apiFetch', () => {
  it('sends JSON body, the anti-CSRF header and cookies (never a token header)', async () => {
    const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    )
    const out = await apiFetch<{ ok: boolean }>('/x', { method: 'POST', body: { a: 1 } })

    expect(out).toEqual({ ok: true })
    const [url, init] = spy.mock.calls[0]
    expect(String(url)).toMatch(/\/x$/)
    const headers = init?.headers as Headers
    expect(headers.get('X-Requested-With')).toBe('travel-split')
    expect(headers.get('X-Device-Token')).toBeNull()
    expect(init?.credentials).toBe('same-origin')
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(init?.body).toBe('{"a":1}')
  })

  it('maps the API error format to ApiError', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'not_found', message: 'ไม่พบทริป' } }), { status: 404 }),
    )
    await expect(apiFetch('/trips/x')).rejects.toMatchObject({ status: 404, code: 'not_found', message: 'ไม่พบทริป' })
  })

  it('announces an expired session', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'login_required', message: 'x' } }), { status: 401 }),
    )
    const seen = vi.fn()
    window.addEventListener(SESSION_EXPIRED_EVENT, seen)
    await expect(apiFetch('/trips/x/expenses')).rejects.toMatchObject({ status: 401 })
    window.removeEventListener(SESSION_EXPIRED_EVENT, seen)
    expect(seen).toHaveBeenCalledOnce()
  })

  it('reports network failures', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('failed'))
    await expect(apiFetch('/healthz')).rejects.toBeInstanceOf(ApiError)
  })
})
