import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { MessagingDraftDemo } from './MessagingDraftDemo'

afterEach(() => vi.unstubAllGlobals())

describe('MessagingDraftDemo', () => {
  it('saves bounded synthetic recipients without browser scope or delivery fields', async () => {
    let body = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => { body = typeof init?.body === 'string' ? init.body : ''; return Promise.resolve(new Response(JSON.stringify({ id: '33333333-3333-4333-8333-333333333333', version: 1 }), { status: 201, headers: { 'Content-Type': 'application/json' } })) }))
    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><MessagingDraftDemo csrfToken="synthetic" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'Demo review' } })
    fireEvent.change(screen.getByLabelText('Plain-text body'), { target: { value: 'Please review this demo message.' } })
    fireEvent.click(screen.getByLabelText('Demo consultant'))
    fireEvent.click(screen.getByRole('button', { name: 'Save demo messaging draft' }))
    await waitFor(() => expect(screen.getByText('Demo messaging draft saved. It remains uncommitted.')).toBeVisible())
    const parsed: { payload: Record<string, unknown> } = JSON.parse(body) as { payload: Record<string, unknown> }
    expect(parsed.payload).toMatchObject({ recordMode: 'demo_messaging', profileCode: 'demo_messaging_v1', primaryRecipient: 'demo_gp', ccRecipients: ['demo_consultant'] })
    expect(body).not.toContain('email')
    expect(body).not.toContain('institutionId')
  })
})
