import { useInfiniteQuery } from '@tanstack/react-query'
import { CalendarDays, Clock3 } from 'lucide-react'

import { episodesAPI, type Episode } from '../../api/client'

const episodeKeys = {
  timeline: (patientId: string, contextVersion: number) => ['episodes', 'timeline', patientId, contextVersion] as const,
}

interface EpisodeTimelineProps {
  patientId: string
  csrfToken: string
  contextVersion: number
  allowed: boolean
}

export function EpisodeTimeline({ patientId, csrfToken, contextVersion, allowed }: EpisodeTimelineProps) {
  const timeline = useInfiniteQuery({
    queryKey: episodeKeys.timeline(patientId, contextVersion),
    queryFn: ({ pageParam }) => episodesAPI.list(patientId, csrfToken, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.nextCursor ?? undefined,
    enabled: allowed,
  })
  const episodes = timeline.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <section className="summary-panel episode-timeline" aria-labelledby="episode-timeline-title">
      <div className="summary-panel-heading">
        <h2 id="episode-timeline-title">Care episodes</h2>
        <Clock3 size={18} aria-hidden="true" />
      </div>
      {!allowed && <p className="summary-muted">Episode details are withheld for this session.</p>}
      {allowed && timeline.isPending && <p className="summary-muted">Loading care episodes…</p>}
      {allowed && timeline.isError && <p className="summary-muted timeline-error" role="alert">Care episodes are temporarily unavailable.</p>}
      {allowed && timeline.isSuccess && episodes.length === 0 && <p className="summary-muted">No care episodes are available in this clinical context.</p>}
      {allowed && !timeline.isError && episodes.length > 0 && (
        <>
          <ol className="episode-list">
            {episodes.map((episode) => <EpisodeRow episode={episode} key={episode.id} />)}
          </ol>
          {timeline.hasNextPage && (
            <button className="secondary-button" type="button" onClick={() => { void timeline.fetchNextPage() }} disabled={timeline.isFetchingNextPage}>
              {timeline.isFetchingNextPage ? 'Loading episodes…' : 'Load earlier episodes'}
            </button>
          )}
        </>
      )}
    </section>
  )
}

function EpisodeRow({ episode }: { episode: Episode }) {
  return (
    <li className="episode-row">
      <span className={`episode-status episode-status-${episode.status}`}>{episode.status}</span>
      <div className="episode-dates">
        <span><CalendarDays size={15} aria-hidden="true" />Started {formatDateTime(episode.startedAt)}</span>
        {episode.endedAt && <span>Closed {formatDateTime(episode.endedAt)}</span>}
      </div>
    </li>
  )
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) return 'not recorded'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'not recorded'
  return new Intl.DateTimeFormat('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC',
  }).format(date)
}
