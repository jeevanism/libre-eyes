import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { LabResultsDraftDemo } from './LabResultsDraftDemo'

function renderDemo() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><LabResultsDraftDemo csrfToken="synthetic" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
}

afterEach(() => vi.unstubAllGlobals())

describe('LabResultsDraftDemo', () => {
  it('saves only the synthetic lab result payload and shows a soft warning', async () => {
    let body = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => { body = typeof init?.body === 'string' ? init.body : ''; return Promise.resolve(new Response(JSON.stringify({ id: '33333333-3333-4333-8333-333333333333', version: 1 }), { status: 201, headers: { 'Content-Type': 'application/json' } })) }))
    renderDemo()
    fireEvent.change(screen.getByLabelText('Demo value'), { target: { value: '7.2' } })
    expect(screen.getByRole('status')).toHaveTextContent('Outside the demo normal range')
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    await waitFor(() => expect(screen.getByText('Demo lab-result draft saved. It remains uncommitted.')).toBeVisible())
    const parsed: { payload: Record<string, unknown> } = JSON.parse(body) as { payload: Record<string, unknown> }
    expect(parsed.payload).toMatchObject({ recordMode: 'demo_lab_result', isSynthetic: true, resultTypeCode: 'demo_lab_hba1c', fieldKind: 'numeric', value: '7.2' })
    expect(body).not.toContain('institutionId')
  })
})
