import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { PrescriptionDraftDemo } from './PrescriptionDraftDemo'

function renderDemo() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><PrescriptionDraftDemo csrfToken="synthetic-csrf" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
}

describe('PrescriptionDraftDemo', () => {
  it('sends a bounded synthetic medication draft without clinical context fields', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: '44444444-4444-4444-8444-444444444444' }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    renderDemo()
    fireEvent.change(screen.getByLabelText('Dose'), { target: { value: '2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    await screen.findByRole('status')
    const call = fetchMock.mock.calls[0] as [RequestInfo | URL, RequestInit | undefined]
    const rawBody = call[1]?.body
    if (typeof rawBody !== 'string') throw new Error('expected JSON request body')
    const body = JSON.parse(rawBody) as { eventTypeCode: string; payload: Record<string, unknown> }
    expect(body.eventTypeCode).toBe('ophthalmology.prescription_demo')
    expect(body.payload).toMatchObject({ recordMode: 'development_synthetic_medication_order' })
    expect(body.payload).not.toHaveProperty('institutionId')
    expect(body.payload).not.toHaveProperty('allergyOverride')
  })
})
