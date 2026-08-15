import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { CorrespondenceDraftDemo } from './CorrespondenceDraftDemo'

vi.mock('@tanstack/react-query', () => ({
  useMutation: ({ mutationFn }: { mutationFn: (payload: unknown) => Promise<unknown> }) => ({
    isPending: false, isError: false, isSuccess: false, error: null,
    mutate: (payload: unknown) => { void mutationFn(payload) }, reset: vi.fn(),
  }),
}))

vi.mock('../../api/client', () => ({
  ApiError: class ApiError extends Error { status = 500 },
  episodesAPI: { createCorrespondenceDemoDraft: vi.fn().mockResolvedValue({}) },
}))

describe('CorrespondenceDraftDemo', () => {
  it('submits only the synthetic plain-text correspondence payload', () => {
    render(<CorrespondenceDraftDemo csrfToken="csrf" episodeId="episode" />)
    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'Demo update' } })
    fireEvent.change(screen.getByLabelText('Plain-text body'), { target: { value: 'A plain text letter.' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo correspondence draft' }))
    expect(screen.getByText(/synthetic, clinician-owned plain-text draft/i)).toBeVisible()
  })
})
