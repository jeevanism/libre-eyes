import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { IOPDraftDemo } from './IOPDraftDemo'

function renderDemo() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}>
      <IOPDraftDemo csrfToken="synthetic-csrf-token" episodeId="22222222-2222-4222-8222-222222222222" />
    </QueryClientProvider>,
  )
}

afterEach(() => vi.unstubAllGlobals())

describe('IOPDraftDemo', () => {
  it('saves a catalogue-only development draft without browser-supplied clinical scope', async () => {
    let requestBody = ''
    const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      requestBody = typeof init?.body === 'string' ? init.body : ''
      return Promise.resolve(new Response(JSON.stringify({
        id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222',
        eventTypeCode: 'ophthalmology.intraocular_pressure', intent: 'create', mode: 'manual', schemaVersion: 1,
        payload: {}, version: 1, expiresAt: '2026-09-01T10:00:00Z', newerCommittedEdits: false,
      }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    })
    vi.stubGlobal('fetch', fetchMock)
    const { container } = renderDemo()

    fireEvent.change(screen.getByLabelText('Right eye development IOP value'), { target: { value: 'development_iop_14' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Demo draft saved'))
    expect(JSON.parse(requestBody)).toEqual({
      eventTypeCode: 'ophthalmology.intraocular_pressure', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: { recordMode: 'development_raw_mmhg', profileCode: 'development_iop_manual_mmhg', eyes: [{ eye: 'right', valueCode: 'development_iop_14' }] },
    })
    expect(requestBody).not.toContain('patientId')
    expect(requestBody).not.toContain('institutionId')
    expect(requestBody).not.toContain('"mmhg"')
    expect((await axe(container)).violations).toEqual([])
  })

  it('requires at least one eye and focuses its alert feedback', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderDemo()
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Select a development demonstration value for at least one eye')
    expect(document.activeElement).toBe(screen.getByRole('alert'))
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('sends bilateral selections and maps a conflict without exposing selections', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      type: 'about:blank', title: 'Request failed', status: 409, code: 'conflict', correlationId: 'synthetic-correlation',
    }), { status: 409, headers: { 'Content-Type': 'application/problem+json' } })))
    renderDemo()
    fireEvent.change(screen.getByLabelText('Right eye development IOP value'), { target: { value: 'development_iop_14' } })
    fireEvent.change(screen.getByLabelText('Left eye development IOP value'), { target: { value: 'development_iop_18' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('episode changed'))
    expect(screen.getByRole('alert')).not.toHaveTextContent('development_iop_14')
  })
})
