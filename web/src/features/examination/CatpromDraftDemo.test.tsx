import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { CatpromDraftDemo } from './CatpromDraftDemo'

afterEach(() => vi.unstubAllGlobals())

describe('CatpromDraftDemo', () => {
  it('saves six synthetic answers without score fields', async () => {
    let body = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => { body = typeof init?.body === 'string' ? init.body : ''; return Promise.resolve(new Response('{}', { status: 201, headers: { 'Content-Type': 'application/json' } })) }))
    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><CatpromDraftDemo csrfToken="synthetic" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
    fireEvent.click(screen.getByRole('button', { name: 'Save demo cataract questionnaire' }))
    await waitFor(() => expect(screen.getByText('Demo cataract questionnaire draft saved. It remains uncommitted.')).toBeVisible())
    expect(body).toContain('demo_q6_a1')
    expect(body).not.toContain('rasch')
    expect(body).not.toContain('score')
  })
})
