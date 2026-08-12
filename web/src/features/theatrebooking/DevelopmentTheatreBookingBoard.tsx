import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarDays, CalendarX2, Clock3, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

import { ApiError, developmentTheatreBookingAPI, type DevelopmentTheatreBoard, type DevelopmentTheatreBookingRequest } from '../../api/client'

const boardKey = (contextVersion: number) => ['development-theatre-booking', contextVersion] as const

export function DevelopmentTheatreBookingBoard({ csrfToken, contextVersion, allowed }: { csrfToken: string, contextVersion: number, allowed: boolean }) {
  const queryClient = useQueryClient()
  const alertRef = useRef<HTMLParagraphElement>(null)
  const [activeID, setActiveID] = useState<string | null>(null)
  const board = useQuery({ queryKey: boardKey(contextVersion), queryFn: developmentTheatreBookingAPI.board, enabled: allowed })
  const mutation = useMutation({
    mutationFn: ({ item, command, targetSessionId }: { item: DevelopmentTheatreBookingRequest, command: 'schedule' | 'reschedule' | 'cancel', targetSessionId?: string | undefined }) => {
      if (command === 'cancel') return developmentTheatreBookingAPI.cancel(item.id, item.version, csrfToken)
      if (!targetSessionId) throw new Error('A demo theatre session is required')
      return command === 'schedule'
        ? developmentTheatreBookingAPI.schedule(item.id, item.version, targetSessionId, csrfToken)
        : developmentTheatreBookingAPI.reschedule(item.id, item.version, targetSessionId, csrfToken)
    },
    onMutate: ({ item }) => setActiveID(item.id),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: boardKey(contextVersion) }),
    onSettled: () => setActiveID(null),
  })
  const error = describeError(mutation.error)
  useEffect(() => { if (mutation.error) alertRef.current?.focus() }, [mutation.error])

  return <section className="summary-panel theatre-booking-board" aria-labelledby="theatre-booking-title">
    <div className="summary-panel-heading"><div><p className="development-kicker">Demo</p><h2 id="theatre-booking-title">Theatre booking</h2></div><CalendarDays size={18} aria-hidden="true" /></div>
    <p className="development-clinic-flow-note">Synthetic schedule only. This does not create an operation, consent, patient, or clinical record.</p>
    {!allowed && <p className="summary-muted">Theatre-booking controls are withheld for this session.</p>}
    {allowed && board.isPending && <p className="summary-muted">Loading synthetic theatre schedule...</p>}
    {allowed && board.isError && <p className="inline-error" role="alert">The synthetic theatre schedule is temporarily unavailable.</p>}
    {allowed && board.data && <BoardContent board={board.data} pending={mutation.isPending} activeID={activeID} onCommand={(item, command, sessionID) => mutation.mutate({ item, command, targetSessionId: sessionID })} />}
    {error && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{error}</p>}
  </section>
}

function BoardContent({ board, pending, activeID, onCommand }: { board: DevelopmentTheatreBoard, pending: boolean, activeID: string | null, onCommand: (item: DevelopmentTheatreBookingRequest, command: 'schedule' | 'reschedule' | 'cancel', sessionID?: string) => void }) {
  return <div className="theatre-booking-content">
    <div><h3>Fixed synthetic sessions</h3><ul className="theatre-session-list">{board.theatreSessions.map((session) => <li key={session.id}><strong>{session.syntheticRoomLabel}</strong><span>{formatSession(session.startsAt, session.endsAt)}</span><span>{session.allocatedMinutes} of {session.capacityMinutes} minutes allocated</span></li>)}</ul></div>
    <div><h3>Booking requests</h3><ul className="theatre-request-list">{board.bookingRequests.map((item) => <BookingRow item={item} board={board} pending={pending} active={activeID === item.id} onCommand={onCommand} key={item.id} />)}</ul></div>
    <div className="theatre-whiteboard"><h3>Read-only whiteboard</h3>{board.whiteboard.map((group) => <div key={group.sessionId}><strong>{group.syntheticRoomLabel}</strong>{group.entries.length === 0 ? <p className="summary-muted">No scheduled synthetic requests.</p> : <ul>{group.entries.map((entry) => <li key={entry.requestId}>{entry.syntheticLabel} <span>{entry.requestedDurationMinutes} min</span></li>)}</ul>}</div>)}</div>
  </div>
}

function BookingRow({ item, board, pending, active, onCommand }: { item: DevelopmentTheatreBookingRequest, board: DevelopmentTheatreBoard, pending: boolean, active: boolean, onCommand: (item: DevelopmentTheatreBookingRequest, command: 'schedule' | 'reschedule' | 'cancel', sessionID?: string) => void }) {
  const [target, setTarget] = useState(item.assignedSessionId ?? board.theatreSessions[0]?.id ?? '')
  const sessions = item.status === 'scheduled' ? board.theatreSessions.filter((session) => session.id !== item.assignedSessionId) : board.theatreSessions
  const selectedTarget = sessions.some((session) => session.id === target) ? target : (sessions[0]?.id ?? '')
  return <li className="theatre-request" aria-busy={active}><div><strong>{item.syntheticLabel}</strong><span className={`development-flow-status theatre-status-${item.status}`}>{item.status}</span><span>{item.requestedDurationMinutes} minutes</span></div>
    {item.status !== 'cancelled' && <div className="theatre-request-actions"><label><span className="sr-only">Target session for {item.syntheticLabel}</span><select value={selectedTarget} onChange={(event) => setTarget(event.target.value)} disabled={pending || sessions.length === 0}>{sessions.map((session) => <option key={session.id} value={session.id}>{session.syntheticRoomLabel} - {formatSession(session.startsAt, session.endsAt)}</option>)}</select></label>
      <button type="button" className="secondary-button" disabled={pending || !selectedTarget} onClick={() => onCommand(item, item.status === 'waiting' ? 'schedule' : 'reschedule', selectedTarget)}>{active ? 'Updating...' : item.status === 'waiting' ? <><Save size={15} aria-hidden="true" />Schedule</> : <><Clock3 size={15} aria-hidden="true" />Reschedule</>}</button>
      <button type="button" className="secondary-button danger-button" disabled={pending} onClick={() => onCommand(item, 'cancel')}><CalendarX2 size={15} aria-hidden="true" />Cancel</button></div>}
  </li>
}

function formatSession(startsAt: string, endsAt: string) { const start = new Date(startsAt); const end = new Date(endsAt); return `${start.toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })} ${start.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })}-${end.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })}` }
function describeError(error: unknown) { if (!(error instanceof ApiError)) return error ? 'The synthetic booking command could not be completed.' : ''; switch (error.problem?.code) { case 'insufficient_capacity': return 'The selected synthetic session does not have enough unused minutes.'; case 'stale_version': return 'The selected synthetic booking changed. Refresh and try again.'; case 'invalid_state': return 'This synthetic booking can no longer make that transition.'; default: return error.status === 404 ? 'The selected synthetic booking is no longer available in this context.' : 'The synthetic booking command could not be completed.' } }
