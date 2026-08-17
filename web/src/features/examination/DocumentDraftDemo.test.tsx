import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { DocumentDraftDemo } from './DocumentDraftDemo'

function renderDemo() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><DocumentDraftDemo csrfToken="synthetic" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
}

afterEach(() => vi.unstubAllGlobals())

describe('DocumentDraftDemo', () => {
  it('saves metadata only and never sends browser scope or files', async () => {
    let body = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => { body = typeof init?.body === 'string' ? init.body : ''; return Promise.resolve(new Response(JSON.stringify({ id: '33333333-3333-4333-8333-333333333333', version: 1 }), { status: 201, headers: { 'Content-Type': 'application/json' } })) }))
    renderDemo()
    fireEvent.change(screen.getByLabelText('Document title'), { target: { value: 'Demo scan summary' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo document draft' }))
    await waitFor(() => expect(screen.getByText('Demo document draft saved. It remains uncommitted.')).toBeVisible())
    const parsed: { payload: Record<string, unknown> } = JSON.parse(body) as { payload: Record<string, unknown> }
    expect(parsed.payload).toMatchObject({ recordMode: 'demo_document', profileCode: 'demo_document_v1', documentType: 'demo_clinic_letter', title: 'Demo scan summary' })
    expect(parsed.payload).not.toHaveProperty('file')
    expect(body).not.toContain('institutionId')
  })
})
