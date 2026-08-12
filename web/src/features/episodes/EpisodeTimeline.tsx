import { useInfiniteQuery } from '@tanstack/react-query'
import { CalendarDays, Clock3 } from 'lucide-react'
import { useState } from 'react'

import { episodesAPI, type Episode, type EventHeader } from '../../api/client'
import { ExaminationDraftWorkspace } from '../examination/ExaminationDraftWorkspace'

const episodeKeys = {
  timeline: (patientId: string, contextVersion: number) => ['episodes', 'timeline', patientId, contextVersion] as const,
  events: (episodeId: string, contextVersion: number) => ['episodes', 'events', episodeId, contextVersion] as const,
}

interface EpisodeTimelineProps {
  patientId: string
  csrfToken: string
  contextVersion: number
  allowed: boolean
  canCreateExaminationDraft: boolean
  workspace?: 'summary' | 'examination'
}

export function EpisodeTimeline({ patientId, csrfToken, contextVersion, allowed, canCreateExaminationDraft, workspace = 'summary' }: EpisodeTimelineProps) {
  const timeline = useInfiniteQuery({
    queryKey: episodeKeys.timeline(patientId, contextVersion),
    queryFn: ({ pageParam }) => episodesAPI.list(patientId, csrfToken, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.nextCursor ?? undefined,
    enabled: allowed,
  })
  const episodes = timeline.data?.pages.flatMap((page) => page.items) ?? []
  const [expandedEpisodeID, setExpandedEpisodeID] = useState<string | null>(null)
  const defaultExpandedEpisodeID = workspace === 'examination'
    ? episodes.find((episode) => episode.status === 'active')?.id ?? episodes[0]?.id ?? null
    : null
  const selectedEpisodeID = expandedEpisodeID ?? defaultExpandedEpisodeID

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
            {episodes.map((episode) => (
              <EpisodeRow
                csrfToken={csrfToken}
                contextVersion={contextVersion}
                episode={episode}
                key={episode.id}
                canCreateExaminationDraft={canCreateExaminationDraft}
                expanded={selectedEpisodeID === episode.id}
                onExpandedChange={() => setExpandedEpisodeID((current) => current === episode.id ? null : episode.id)}
                workspace={workspace}
              />
            ))}
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

function EpisodeRow({ csrfToken, contextVersion, episode, canCreateExaminationDraft, expanded, onExpandedChange, workspace }: { csrfToken: string, contextVersion: number, episode: Episode, canCreateExaminationDraft: boolean, expanded: boolean, onExpandedChange: () => void, workspace: 'summary' | 'examination' }) {
  const eventsID = `episode-events-${episode.id}`
  return (
    <li className="episode-row">
      <div className="episode-row-header">
        <span className={`episode-status episode-status-${episode.status}`}>{episode.status}</span>
        <div className="episode-dates">
          <span><CalendarDays size={15} aria-hidden="true" />Started {formatDateTime(episode.startedAt)}</span>
          {episode.endedAt && <span>Closed {formatDateTime(episode.endedAt)}</span>}
        </div>
        <button
          aria-controls={eventsID}
          aria-expanded={expanded}
          className="secondary-button episode-events-toggle"
          onClick={onExpandedChange}
          type="button"
        >
          {expanded ? 'Hide events' : 'Show events'}
        </button>
      </div>
      {expanded && <EventHeaders csrfToken={csrfToken} contextVersion={contextVersion} episodeId={episode.id} id={eventsID} />}
      {expanded && workspace === 'examination' && episode.status === 'active' && canCreateExaminationDraft && <ExaminationDraftWorkspace csrfToken={csrfToken} episodeId={episode.id} />}
    </li>
  )
}

function EventHeaders({ csrfToken, contextVersion, episodeId, id }: { csrfToken: string, contextVersion: number, episodeId: string, id: string }) {
  const timeline = useInfiniteQuery({
    queryKey: episodeKeys.events(episodeId, contextVersion),
    queryFn: ({ pageParam }) => episodesAPI.listEvents(episodeId, csrfToken, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.nextCursor ?? undefined,
  })
  const events = timeline.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <div className="event-headers" id={id}>
      {timeline.isPending && <p className="summary-muted">Loading event headers...</p>}
      {timeline.isError && <p className="summary-muted timeline-error" role="alert">Event headers are temporarily unavailable.</p>}
      {timeline.isSuccess && events.length === 0 && <p className="summary-muted">No current event headers are available.</p>}
      {!timeline.isError && events.length > 0 && (
        <>
          <ol className="event-header-list">
            {events.map((event) => <EventHeaderRow event={event} key={event.id} />)}
          </ol>
          {timeline.hasNextPage && (
            <button className="secondary-button" type="button" onClick={() => { void timeline.fetchNextPage() }} disabled={timeline.isFetchingNextPage}>
              {timeline.isFetchingNextPage ? 'Loading events...' : 'Load earlier events'}
            </button>
          )}
        </>
      )}
    </div>
  )
}

function EventHeaderRow({ event }: { event: EventHeader }) {
  return (
    <li>
      <span>{event.eventTypeCode}</span>
      <time dateTime={event.occurredAt}>{formatDateTime(event.occurredAt)}</time>
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
