import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiFetch } from './client'

afterEach(() => vi.restoreAllMocks())

describe('apiFetch', () => {
  it('sends JSON body and member header', async () => {
    const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    )
    const out = await apiFetch<{ ok: boolean }>('/x', { method: 'POST', body: { a: 1 }, memberId: 'm1' })

    expect(out).toEqual({ ok: true })
    const [url, init] = spy.mock.calls[0]
    expect(String(url)).toMatch(/\/x$/)
    const headers = init?.headers as Headers
    expect(headers.get('X-Member-Id')).toBe('m1')
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(init?.body).toBe('{"a":1}')
  })

  it('maps the API error format to ApiError', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'not_found', message: 'ไม่พบทริป' } }), { status: 404 }),
    )
    await expect(apiFetch('/trips/x')).rejects.toMatchObject({ status: 404, code: 'not_found', message: 'ไม่พบทริป' })
  })

  it('reports network failures', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('failed'))
    await expect(apiFetch('/healthz')).rejects.toBeInstanceOf(ApiError)
  })
})
