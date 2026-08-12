import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { EpisodeTimeline } from './EpisodeTimeline'

function renderTimeline(allowed = true) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <EpisodeTimeline patientId="11111111-1111-4111-8111-111111111111" csrfToken="synthetic-csrf-token" contextVersion={1} allowed={allowed} canCreateExaminationDraft={false} />
    </QueryClientProvider>,
  )
}

function response(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

afterEach(() => vi.unstubAllGlobals())

describe('EpisodeTimeline', () => {
  it('renders scoped episode headers and follows an opaque cursor', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({
        items: [{
          id: '22222222-2222-4222-8222-222222222222', patientId: '11111111-1111-4111-8111-111111111111',
          status: 'active', startedAt: '2026-08-07T09:30:00Z', endedAt: null,
          supportServices: false, changeTracker: false, version: 2,
        }], nextCursor: 'opaque.synthetic.cursor',
      }))
      .mockResolvedValueOnce(response({
        items: [{
          id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222',
          eventTypeCode: 'core.examination', occurredAt: '2026-08-07T10:00:00Z', status: 'current', version: 1,
        }], nextCursor: null,
      }))
      .mockResolvedValueOnce(response({ items: [], nextCursor: null }))
    vi.stubGlobal('fetch', fetchMock)
    const { container } = renderTimeline()

    expect(await screen.findByText('active')).toBeVisible()
    expect(screen.getByText(/Started 07 Aug 2026/)).toBeVisible()
    expect((await axe(container)).violations).toEqual([])
    fireEvent.click(screen.getByRole('button', { name: 'Show events' }))
    expect(await screen.findByText('core.examination')).toBeVisible()
    expect(fetchMock.mock.calls[1]?.[0]).toContain('/episodes/22222222-2222-4222-8222-222222222222/events')
    fireEvent.click(screen.getByRole('button', { name: 'Load earlier episodes' }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
    expect(fetchMock.mock.calls[2]?.[0]).toContain('cursor=opaque.synthetic.cursor')
  })

  it('withholds the timeline before requesting it without permission', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderTimeline(false)
    expect(screen.getByText('Episode details are withheld for this session.')).toBeVisible()
    await waitFor(() => expect(fetchMock).not.toHaveBeenCalled())
  })

  it('renders an empty state and an error state without stale episode data', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response({ items: [], nextCursor: null }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ title: 'Request failed' }), { status: 503, headers: { 'Content-Type': 'application/problem+json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const { rerender } = renderTimeline()
    expect(await screen.findByText('No care episodes are available in this clinical context.')).toBeVisible()

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <EpisodeTimeline patientId="22222222-2222-4222-8222-222222222222" csrfToken="synthetic-csrf-token" contextVersion={1} allowed canCreateExaminationDraft={false} />
      </QueryClientProvider>,
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('Care episodes are temporarily unavailable.')
    expect(screen.queryByText('No care episodes are available in this clinical context.')).not.toBeInTheDocument()
  })
})
