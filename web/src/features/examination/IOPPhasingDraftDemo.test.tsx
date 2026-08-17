import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { IOPPhasingDraftDemo } from './IOPPhasingDraftDemo'

afterEach(() => vi.unstubAllGlobals())

describe('IOPPhasingDraftDemo', () => {
  it('saves timed synthetic readings without interpretation fields', async () => {
    let body = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => { body = typeof init?.body === 'string' ? init.body : ''; return Promise.resolve(new Response('{}', { status: 201, headers: { 'Content-Type': 'application/json' } })) }))
    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><IOPPhasingDraftDemo csrfToken="synthetic" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
    fireEvent.click(screen.getByRole('button', { name: 'Save demo IOP phasing draft' }))
    await waitFor(() => expect(screen.getByText('Demo IOP phasing draft saved. It remains uncommitted.')).toBeVisible())
    expect(body).toContain('measurementTime')
    expect(body).not.toContain('average')
  })
})
