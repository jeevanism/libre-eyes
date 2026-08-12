import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { DevelopmentClinicFlowBoard } from './DevelopmentClinicFlowBoard'

function renderBoard(allowed = true) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}><DevelopmentClinicFlowBoard allowed={allowed} contextVersion={1} csrfToken="synthetic-csrf-token" /></QueryClientProvider>)
}

function response(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json' } })
}

afterEach(() => vi.unstubAllGlobals())

describe('DevelopmentClinicFlowBoard', () => {
  it('renders synthetic tickets and submits a versioned state command', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({ items: [{ id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'waiting', assigneeDisplayName: null, version: 1 }] }))
      .mockResolvedValueOnce(response({ id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'arrived', assigneeDisplayName: null, version: 2 }))
      .mockResolvedValueOnce(response({ items: [{ id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'arrived', assigneeDisplayName: null, version: 2 }] }))
    vi.stubGlobal('fetch', fetchMock)
    const { container } = renderBoard()

    expect(await screen.findByText('Synthetic queue patient A')).toBeVisible()
    expect((await axe(container)).violations).toEqual([])
    fireEvent.click(screen.getByRole('button', { name: 'Mark arrived for Synthetic queue patient A' }))
    await waitFor(() => expect(screen.getByText('Arrived')).toBeVisible())
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      expect.stringContaining('/development/clinic-flow/tickets/77777777-7777-4777-8777-777777777777/arrive'),
      expect.objectContaining({
        body: JSON.stringify({ expectedVersion: 1 }),
      }),
    )
  })

  it('withholds the board before requesting it without permission', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderBoard(false)
    expect(screen.getByText('Clinic-flow controls are withheld for this session.')).toBeVisible()
    await waitFor(() => expect(fetchMock).not.toHaveBeenCalled())
  })

  it('announces, focuses, and refreshes after a conflict', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({ items: [{ id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'waiting', assigneeDisplayName: null, version: 1 }] }))
      .mockResolvedValueOnce(response({ title: 'Ticket changed' }, 409))
      .mockResolvedValueOnce(response({ items: [{ id: '77777777-7777-4777-8777-777777777777', syntheticPatientLabel: 'Synthetic queue patient A', status: 'arrived', assigneeDisplayName: null, version: 2 }] }))
    vi.stubGlobal('fetch', fetchMock)
    renderBoard()
    fireEvent.click(await screen.findByRole('button', { name: 'Mark arrived for Synthetic queue patient A' }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('changed')
    expect(document.activeElement).toBe(alert)
    expect(await screen.findByRole('button', { name: 'Claim ticket for Synthetic queue patient A' })).toBeVisible()
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })
})
