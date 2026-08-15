import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DevelopmentReferralAppointmentBoard } from './DevelopmentReferralAppointmentBoard'

function response(body: unknown, status = 200) { return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }) }
function renderBoard() { const client = new QueryClient({ defaultOptions: { queries: { retry: false } } }); return render(<QueryClientProvider client={client}><DevelopmentReferralAppointmentBoard allowed contextVersion={1} csrfToken="csrf" /></QueryClientProvider>) }
afterEach(() => vi.unstubAllGlobals())

describe('DevelopmentReferralAppointmentBoard', () => {
  it('renders synthetic requests and submits a versioned transition', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({ items: [{ id: '77777777-7777-4777-8777-777777777771', syntheticPatientLabel: 'Demo referral to GP', recipientRole: 'demo_gp', clinicCode: 'demo_general_eye_clinic', appointmentDate: '2026-08-18', appointmentTime: '09:00', priority: 'routine', notes: '', status: 'requested', version: 1 }] }))
      .mockResolvedValueOnce(response({ id: '77777777-7777-4777-8777-777777777771', syntheticPatientLabel: 'Demo referral to GP', recipientRole: 'demo_gp', clinicCode: 'demo_general_eye_clinic', appointmentDate: '2026-08-18', appointmentTime: '09:00', priority: 'routine', notes: '', status: 'scheduled', version: 2 }))
      .mockResolvedValueOnce(response({ items: [] }))
    vi.stubGlobal('fetch', fetchMock)
    renderBoard()
    expect(await screen.findByText('Demo referral to GP')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Schedule' }))
    await waitFor(() => expect(fetchMock).toHaveBeenNthCalledWith(2, expect.stringContaining('/development/referral-appointments/requests/77777777-7777-4777-8777-777777777771/schedule'), expect.objectContaining({ body: JSON.stringify({ expectedVersion: 1 }) })))
  })
})
