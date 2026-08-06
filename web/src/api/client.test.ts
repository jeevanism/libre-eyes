import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError, patientSearchAPI, type PatientSearchRequest } from './client'

afterEach(() => vi.unstubAllGlobals())

describe('patientSearchAPI', () => {
  it('posts criteria and cursor only in JSON with the session CSRF token', async () => {
    const response = {
      items: [],
      page: { hasMore: false, nextCursor: null },
    }
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(response), {
      status: 200,
      headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' },
    }))
    vi.stubGlobal('fetch', fetchMock)
    const request: PatientSearchRequest = {
      criteria: { kind: 'identifier', identifierTypeId: '7', value: 'ABC123' },
      limit: 25,
      cursor: 'opaque-cursor',
    }

    await expect(patientSearchAPI.search(request, 'csrf-token')).resolves.toEqual(response)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/patients/searches')
    expect(url).not.toContain('ABC123')
    expect(init).toMatchObject({
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'csrf-token',
      },
      body: JSON.stringify(request),
    })
  })

  it('retains a bounded retry delay from a rate-limit response', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      type: 'about:blank', title: 'Too many requests', status: 429,
      code: 'rate_limited', correlationId: 'synthetic-correlation',
    }), {
      status: 429,
      headers: { 'Content-Type': 'application/problem+json', 'Retry-After': '7' },
    }))
    vi.stubGlobal('fetch', fetchMock)

    const error = await patientSearchAPI.search({
      criteria: { kind: 'demographic', familyName: 'Patient', dateOfBirth: '1980-01-01', gender: 'unknown' },
      limit: 25,
    }, 'csrf-token').catch((reason: unknown) => reason)

    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({ status: 429, retryAfterSeconds: 7 })
  })
})
