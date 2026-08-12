import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { DevelopmentTheatreBookingBoard } from './DevelopmentTheatreBookingBoard'

const sessionA = 'b1111111-1111-4111-8111-111111111111'
const sessionB = 'c1111111-1111-4111-8111-111111111111'
const requestA = 'd1111111-1111-4111-8111-111111111111'

function board(status: 'waiting' | 'scheduled' | 'cancelled' = 'waiting') {
  return { developmentOnly: true, sessionDate: '2026-08-10', theatreSessions: [
    { id: sessionA, syntheticRoomLabel: 'Development Theatre One', startsAt: '2026-08-10T08:00:00Z', endsAt: '2026-08-10T12:00:00Z', capacityMinutes: 180, allocatedMinutes: 0, version: 1 },
    { id: sessionB, syntheticRoomLabel: 'Development Theatre One', startsAt: '2026-08-10T13:00:00Z', endsAt: '2026-08-10T17:00:00Z', capacityMinutes: 180, allocatedMinutes: 0, version: 1 },
  ], bookingRequests: [{ id: requestA, syntheticLabel: 'Synthetic booking request A', requestedDurationMinutes: 60, status, assignedSessionId: status === 'scheduled' ? sessionA : null, version: 1 }], whiteboard: [] }
}

function response(body: unknown, status = 200) { return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json' } }) }
function renderBoard(allowed = true) { const query = new QueryClient({ defaultOptions: { queries: { retry: false } } }); return render(<QueryClientProvider client={query}><DevelopmentTheatreBookingBoard allowed={allowed} contextVersion={1} csrfToken="synthetic-csrf-token" /></QueryClientProvider>) }

afterEach(() => vi.unstubAllGlobals())

describe('DevelopmentTheatreBookingBoard', () => {
  it('renders synthetic data and schedules with a versioned command', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(response(board())).mockResolvedValueOnce(response({ ...board('scheduled').bookingRequests[0], assignedSessionId: sessionA, version: 2 })).mockResolvedValueOnce(response(board('scheduled')))
    vi.stubGlobal('fetch', fetchMock)
    renderBoard()
    expect(await screen.findByText('Synthetic booking request A')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Schedule' }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
    expect(fetchMock).toHaveBeenNthCalledWith(2, expect.stringContaining(`/development/theatre-booking/requests/${requestA}/schedule`), expect.objectContaining({ body: JSON.stringify({ expectedVersion: 1, targetSessionId: sessionA }) }))
  })

  it('withholds the board without querying before permission is present', () => {
    const fetchMock = vi.fn(); vi.stubGlobal('fetch', fetchMock); renderBoard(false)
    expect(screen.getByText('Theatre-booking controls are withheld for this session.')).toBeVisible()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('refocuses the alert after identical command failures', async () => {
    const conflict = { type: 'about:blank', title: 'Conflict', status: 409, code: 'insufficient_capacity', correlationId: 'synthetic-correlation' }
    const fetchMock = vi.fn().mockResolvedValueOnce(response(board())).mockResolvedValueOnce(response(conflict, 409)).mockResolvedValueOnce(response(conflict, 409))
    vi.stubGlobal('fetch', fetchMock)
    renderBoard()
    const schedule = await screen.findByRole('button', { name: 'Schedule' })
    fireEvent.click(schedule)
    const alert = await screen.findByRole('alert')
    await waitFor(() => expect(document.activeElement).toBe(alert))
    schedule.focus()
    fireEvent.click(schedule)
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole('alert')))
  })
})
