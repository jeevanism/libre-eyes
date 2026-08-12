import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { DiagnosisDraftDemo } from './DiagnosisDraftDemo'

function renderDemo() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><DiagnosisDraftDemo csrfToken="synthetic-csrf-token" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
}

afterEach(() => vi.unstubAllGlobals())

describe('DiagnosisDraftDemo', () => {
  it('saves only the synthetic development payload without browser-supplied scope', async () => {
    let requestBody = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      requestBody = typeof init?.body === 'string' ? init.body : ''
      return Promise.resolve(new Response(JSON.stringify({ id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222', eventTypeCode: 'ophthalmology.principal_diagnosis_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload: {}, version: 1, expiresAt: '2026-09-01T10:00:00Z' }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    }))
    const { container } = renderDemo()
    fireEvent.change(screen.getByLabelText('Demo example'), { target: { value: 'development_cataract' } })
    fireEvent.change(screen.getByLabelText('Laterality'), { target: { value: 'left' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Demo draft saved'))
    expect(JSON.parse(requestBody)).toMatchObject({ eventTypeCode: 'ophthalmology.principal_diagnosis_demo', payload: { recordMode: 'development_synthetic_diagnosis', profileCode: 'development_ophthalmology_diagnosis_v1', selectionCode: 'development_cataract', laterality: 'left' } })
    expect(requestBody).not.toContain('patientId')
    expect(requestBody).not.toContain('conceptId')
    expect((await axe(container)).violations).toEqual([])
  })

  it('requires every synthetic field and focuses the error', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderDemo()
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Select a development example')
    expect(screen.getByLabelText('Demo example')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('Demo example')).toHaveAttribute('aria-describedby', 'diagnosis-demo-error-22222222-2222-4222-8222-222222222222')
    expect(document.activeElement).toBe(screen.getByRole('alert'))
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
