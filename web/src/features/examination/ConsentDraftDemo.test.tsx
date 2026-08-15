import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ConsentDraftDemo } from './ConsentDraftDemo'

function renderDemo() { return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><ConsentDraftDemo csrfToken="synthetic-csrf" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>) }
afterEach(() => vi.unstubAllGlobals())

describe('ConsentDraftDemo', () => {
  it('saves only the bounded synthetic consent payload', async () => {
    let body = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => { body = typeof init?.body === 'string' ? init.body : ''; return Promise.resolve(new Response(JSON.stringify({ id: 'demo' }), { status: 201, headers: { 'Content-Type': 'application/json' } })) }))
    const { container } = renderDemo()
    fireEvent.change(screen.getByLabelText('Demo consent procedure'), { target: { value: 'development_trabeculectomy' } })
    fireEvent.change(screen.getByLabelText('Demo comments'), { target: { value: '  synthetic note  ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo consent draft' }))
    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('remains uncommitted'))
    expect(JSON.parse(body)).toMatchObject({ eventTypeCode: 'ophthalmology.consent_demo', payload: { procedureCode: 'development_trabeculectomy', recordMode: 'development_synthetic_consent', comment: 'synthetic note' } })
    expect(body).not.toContain('signature'); expect(body).not.toContain('patientId')
    expect((await axe(container)).violations).toEqual([])
  })

  it('rejects an incomplete form without calling the API', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderDemo()
    fireEvent.change(screen.getByLabelText('Demo form type'), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo consent draft' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Complete the demo consent fields')
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
