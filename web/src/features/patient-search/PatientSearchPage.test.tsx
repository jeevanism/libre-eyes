import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Session } from '../../api/client'
import { PatientSearchPage } from './PatientSearchPage'

const session: Session = {
  user: { id: '10', displayName: 'Synthetic Clinician' },
  context: {
    institution: { id: '1', name: 'Vision Hospital' },
    site: { id: '2', name: 'Main Clinic' },
    firm: { id: '3', name: 'Ophthalmology' },
  },
  permissions: ['patient.search', 'patient.duplicate_check'],
  csrfToken: 'synthetic-csrf-token',
  idleExpiresAt: '2026-08-06T12:00:00Z',
  absoluteExpiresAt: '2026-08-06T18:00:00Z',
  contextVersion: 1,
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <PatientSearchPage session={session} />
    </QueryClientProvider>,
  )
}

function response(body: unknown, status = 200, headers: HeadersInit = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json', ...headers },
  })
}

afterEach(() => vi.unstubAllGlobals())

describe('PatientSearchPage', () => {
  it('submits structured demographic criteria and renders minimum-disclosure results', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response({
      items: [{
        patientId: '11111111-1111-4111-8111-111111111111',
        fullName: 'Alice Patient', givenName: 'Alice', familyName: 'Patient',
        dateOfBirth: '1980-02-03', gender: 'female',
        primaryIdentifier: { typeId: '7', label: 'Hospital number', value: 'H 123 456' },
        deceased: false, dateOfDeath: null,
      }],
      page: { hasMore: false, nextCursor: null },
    }))
    vi.stubGlobal('fetch', fetchMock)
    const { container } = renderPage()

    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Patient' } })
    fireEvent.change(screen.getByLabelText('Date of birth'), { target: { value: '1980-02-03' } })
    fireEvent.change(screen.getByLabelText('Gender'), { target: { value: 'female' } })
    fireEvent.click(screen.getByRole('button', { name: 'Search patients' }))

    expect(await screen.findByText('Alice Patient')).toBeVisible()
    expect(screen.getByText('H 123 456')).toBeVisible()
    expect(screen.queryByText('11111111-1111-4111-8111-111111111111')).not.toBeInTheDocument()
    expect(document.activeElement).toBe(
      screen.getByRole('heading', { name: 'Search results' }).closest('.result-focus-target'),
    )
    expect((await axe(container)).violations).toEqual([])
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/patients/searches')
    if (typeof init.body !== 'string') throw new Error('patient search body is not JSON text')
    expect(JSON.parse(init.body)).toEqual({
      criteria: { kind: 'demographic', familyName: 'Patient', dateOfBirth: '1980-02-03', gender: 'female' },
      limit: 25,
    })

    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Changed' } })
    expect(screen.queryByText('Alice Patient')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reset' }))
    expect(screen.getByLabelText('Family name')).toHaveValue('')
  })

  it('communicates duplicate-check coverage and truncation without claiming uniqueness', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({
      coverage: 'exact_only', populationCoverage: 'current_institution_only',
      complete: false, noCandidatesDoesNotExcludeDuplicate: true,
      hardConflict: false, truncated: true, candidates: [],
    })))
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'Duplicate check' }))
    fireEvent.change(screen.getByLabelText('Given name'), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Patient' } })
    fireEvent.change(screen.getByLabelText('Date of birth'), { target: { value: '1980-02-03' } })
    fireEvent.click(screen.getByRole('button', { name: 'Check duplicates' }))

    expect(await screen.findByText(/No exact candidates found/)).toBeVisible()
    expect(screen.getByText(/does not exclude a duplicate/)).toBeVisible()
    expect(screen.getByRole('alert')).toHaveTextContent(/limited to 100/)
  })

  it('maps rate limiting to a generic retry message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({
      type: 'about:blank', title: 'Too many requests', status: 429,
      code: 'rate_limited', correlationId: 'synthetic-correlation',
    }, 429, { 'Retry-After': '8' })))
    renderPage()

    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Patient' } })
    fireEvent.change(screen.getByLabelText('Date of birth'), { target: { value: '1980-02-03' } })
    fireEvent.change(screen.getByLabelText('Gender'), { target: { value: 'female' } })
    fireEvent.click(screen.getByRole('button', { name: 'Search patients' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Try again in 8 seconds')
    expect(document.activeElement).toBe(screen.getByRole('alert'))
    expect(screen.queryByText('Patient')).not.toBeInTheDocument()
  })

  it('associates client validation with each invalid field and focuses the alert', async () => {
    vi.stubGlobal('fetch', vi.fn())
    const { container } = renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'Search patients' }))

    const alert = await screen.findByRole('alert')
    expect(document.activeElement).toBe(alert)
    expect(screen.getByLabelText('Family name')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('Family name')).toHaveAttribute('aria-describedby', alert.id)
    expect(screen.getByLabelText('Date of birth')).toHaveAttribute('aria-invalid', 'true')
    expect((await axe(container)).violations).toEqual([])
  })

  it('submits configured identifier criteria without client-side normalization', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response({
      items: [], page: { hasMore: false, nextCursor: null },
    }))
    vi.stubGlobal('fetch', fetchMock)
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'Identifier' }))
    fireEvent.change(screen.getByLabelText('Identifier type ID'), { target: { value: '7' } })
    fireEvent.change(screen.getByLabelText('Identifier value'), { target: { value: ' H 123 ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Search patients' }))

    expect(await screen.findByText(/No matching patients found/)).toBeVisible()
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    if (typeof init.body !== 'string') throw new Error('patient search body is not JSON text')
    expect(JSON.parse(init.body)).toEqual({
      criteria: { kind: 'identifier', identifierTypeId: '7', value: ' H 123 ' },
      limit: 25,
    })
  })

  it('echoes the opaque next cursor without exposing or transforming it', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({
        items: [], page: { hasMore: true, nextCursor: 'opaque.synthetic.cursor' },
      }))
      .mockResolvedValueOnce(response({
        items: [], page: { hasMore: false, nextCursor: null },
      }))
    vi.stubGlobal('fetch', fetchMock)
    renderPage()

    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'Patient' } })
    fireEvent.change(screen.getByLabelText('Date of birth'), { target: { value: '1980-02-03' } })
    fireEvent.change(screen.getByLabelText('Gender'), { target: { value: 'female' } })
    fireEvent.click(screen.getByRole('button', { name: 'Search patients' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Next page' }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    const [, init] = fetchMock.mock.calls[1] as [string, RequestInit]
    if (typeof init.body !== 'string') throw new Error('patient search body is not JSON text')
    expect(JSON.parse(init.body)).toMatchObject({ cursor: 'opaque.synthetic.cursor' })
  })

  it.each([
    [403, 'You do not have permission'],
    [413, 'The search request is too large'],
    [504, 'Patient search timed out'],
  ])('maps HTTP %i without exposing request criteria', async (status, message) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({
      type: 'about:blank', title: 'Request failed', status,
      code: 'synthetic_error', correlationId: 'synthetic-correlation',
    }, status)))
    renderPage()

    fireEvent.change(screen.getByLabelText('Family name'), { target: { value: 'SensitiveName' } })
    fireEvent.change(screen.getByLabelText('Date of birth'), { target: { value: '1980-02-03' } })
    fireEvent.change(screen.getByLabelText('Gender'), { target: { value: 'female' } })
    fireEvent.click(screen.getByRole('button', { name: 'Search patients' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(message)
    expect(alert).not.toHaveTextContent('SensitiveName')
  })

  it('has no automated accessibility violations before a search', async () => {
    vi.stubGlobal('fetch', vi.fn())
    const { container } = renderPage()
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Patient search' })).toBeVisible())
    expect((await axe(container)).violations).toEqual([])
  })
})
