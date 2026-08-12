import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { VisualAcuityDraftDemo } from './VisualAcuityDraftDemo'

function renderDemo() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}>
      <VisualAcuityDraftDemo csrfToken="synthetic-csrf-token" episodeId="22222222-2222-4222-8222-222222222222" />
    </QueryClientProvider>,
  )
}

afterEach(() => vi.unstubAllGlobals())

describe('VisualAcuityDraftDemo', () => {
  it('saves a development-only, draft-only payload without browser-supplied scope', async () => {
    let requestBody = ''
    const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      requestBody = typeof init?.body === 'string' ? init.body : ''
      return Promise.resolve(new Response(JSON.stringify({
      id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222',
      eventTypeCode: 'ophthalmology.visual_acuity', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: {}, version: 1, expiresAt: '2026-09-01T10:00:00Z', newerCommittedEdits: false,
      }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    })
    vi.stubGlobal('fetch', fetchMock)
    const { container } = renderDemo()

    fireEvent.change(screen.getByLabelText('Right eye demonstration value'), { target: { value: 'development_value_m028' } })
    fireEvent.change(screen.getByLabelText('Left eye demonstration value'), { target: { value: 'development_value_150' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Demo draft saved'))
    const request: unknown = JSON.parse(requestBody)
    expect(fetchMock.mock.calls[0]?.[0]).toContain('/episodes/22222222-2222-4222-8222-222222222222/event-drafts')
    expect(request).toEqual({
      eventTypeCode: 'ophthalmology.visual_acuity', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: {
        recordMode: 'simple',
        eyes: [
          { eye: 'right', assessment: 'recorded', readings: [{ unitCode: 'development_distance_scale', valueCode: 'development_value_m028', methodCode: 'development_unaided' }] },
          { eye: 'left', assessment: 'recorded', readings: [{ unitCode: 'development_distance_scale', valueCode: 'development_value_150', methodCode: 'development_unaided' }] },
        ],
      },
    })
    expect(request).not.toHaveProperty('patientId')
    expect(request).not.toHaveProperty('institutionId')
    expect((await axe(container)).violations).toEqual([])
  })

  it('requires both demonstration values before saving', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderDemo()
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Select a demonstration value for both eyes')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('clears a previous save confirmation before showing a later client validation error', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222',
      eventTypeCode: 'ophthalmology.visual_acuity', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: {}, version: 1, expiresAt: '2026-09-01T10:00:00Z', newerCommittedEdits: false,
    }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    renderDemo()

    fireEvent.change(screen.getByLabelText('Right eye demonstration value'), { target: { value: 'development_value_m028' } })
    fireEvent.change(screen.getByLabelText('Left eye demonstration value'), { target: { value: 'development_value_150' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    await screen.findByRole('status')

    fireEvent.change(screen.getByLabelText('Right eye demonstration value'), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Select a demonstration value for both eyes')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
