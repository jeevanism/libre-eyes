import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ClipboardList, LogIn, Play, RotateCcw, Check } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { ReactNode } from 'react'

import { ApiError, developmentClinicFlowAPI, type DevelopmentFlowTicket } from '../../api/client'

const clinicFlowKey = (contextVersion: number) => ['development-clinic-flow', contextVersion] as const

interface DevelopmentClinicFlowBoardProps {
  csrfToken: string
  contextVersion: number
  allowed: boolean
}

export function DevelopmentClinicFlowBoard({ csrfToken, contextVersion, allowed }: DevelopmentClinicFlowBoardProps) {
  const queryClient = useQueryClient()
  const alertRef = useRef<HTMLParagraphElement>(null)
  const [activeTicketID, setActiveTicketID] = useState<string | null>(null)
  const tickets = useQuery({
    queryKey: clinicFlowKey(contextVersion),
    queryFn: () => developmentClinicFlowAPI.list(csrfToken),
    enabled: allowed,
  })
  const command = useMutation({
    mutationFn: ({ ticket, action }: { ticket: DevelopmentFlowTicket, action: Action }) =>
      developmentClinicFlowAPI.command(ticket.id, action, ticket.version, csrfToken),
    onMutate: ({ ticket }) => setActiveTicketID(ticket.id),
    onSuccess: (updated) => {
      queryClient.setQueryData<{ items: DevelopmentFlowTicket[] }>(clinicFlowKey(contextVersion), (current) => {
        if (!current) return current
        return { items: current.items.map((ticket) => ticket.id === updated.id ? updated : ticket) }
      })
    },
    onSettled: () => {
      setActiveTicketID(null)
      void queryClient.invalidateQueries({ queryKey: clinicFlowKey(contextVersion) })
    },
  })
  const error = command.error instanceof ApiError && command.error.status === 404
    ? 'The selected demonstration ticket is no longer available in this context.'
    : command.error instanceof ApiError && command.error.status === 409
      ? 'The selected demonstration ticket changed. Refresh and try again.'
      : command.isError ? 'The demo queue command could not be completed.' : ''

  useEffect(() => {
    if (error) alertRef.current?.focus()
  }, [error])

  return (
    <section className="summary-panel development-clinic-flow" aria-labelledby="development-clinic-flow-title">
      <div className="summary-panel-heading">
        <div>
          <p className="development-kicker">Demo</p>
          <h2 id="development-clinic-flow-title">Clinic flow queue</h2>
        </div>
        <ClipboardList size={18} aria-hidden="true" />
      </div>
      <p className="development-clinic-flow-note">Synthetic queue state only. This does not check in a patient or create a clinical record.</p>
      {!allowed && <p className="summary-muted">Clinic-flow controls are withheld for this session.</p>}
      {allowed && tickets.isPending && <p className="summary-muted">Loading demo queue…</p>}
      {allowed && tickets.isError && <p className="inline-error" role="alert" tabIndex={-1}>The demo queue is temporarily unavailable.</p>}
      {allowed && tickets.data?.items.length === 0 && <p className="summary-muted">No synthetic tickets are available in this context.</p>}
      {allowed && tickets.data && tickets.data.items.length > 0 && (
        <ol className="development-clinic-flow-list">
          {tickets.data.items.map((ticket) => <TicketRow key={ticket.id} ticket={ticket} pending={command.isPending} active={activeTicketID === ticket.id} onCommand={(action) => command.mutate({ ticket, action })} />)}
        </ol>
      )}
      {error && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{error}</p>}
    </section>
  )
}

type Action = 'arrive' | 'claim' | 'release' | 'complete'

function TicketRow({ ticket, pending, active, onCommand }: { ticket: DevelopmentFlowTicket, pending: boolean, active: boolean, onCommand: (action: Action) => void }) {
  const actions = actionsFor(ticket)
  return (
    <li aria-busy={active} className="development-clinic-flow-ticket">
      <div>
        <strong>{ticket.syntheticPatientLabel}</strong>
        <span className={`development-flow-status development-flow-status-${ticket.status}`}>{displayStatus(ticket.status)}</span>
        {ticket.assigneeDisplayName && <span className="development-flow-owner">Owner: {ticket.assigneeDisplayName}</span>}
      </div>
      <div className="development-flow-actions">
        {actions.map((action) => <button aria-label={`${action.label} for ${ticket.syntheticPatientLabel}`} className="secondary-button development-flow-action" disabled={pending} key={action.name} onClick={() => onCommand(action.name)} type="button">{action.icon}{active ? 'Updating ticket...' : action.label}</button>)}
      </div>
    </li>
  )
}

function actionsFor(ticket: DevelopmentFlowTicket): Array<{ name: Action, label: string, icon: ReactNode }> {
  switch (ticket.status) {
    case 'waiting': return [{ name: 'arrive', label: 'Mark arrived', icon: <LogIn size={15} aria-hidden="true" /> }]
    case 'arrived': return [{ name: 'claim', label: 'Claim ticket', icon: <Play size={15} aria-hidden="true" /> }]
    case 'in_progress': return [
      { name: 'release', label: 'Release ticket', icon: <RotateCcw size={15} aria-hidden="true" /> },
      { name: 'complete', label: 'Complete ticket', icon: <Check size={15} aria-hidden="true" /> },
    ]
    default: return []
  }
}

function displayStatus(status: DevelopmentFlowTicket['status']): string {
  switch (status) {
    case 'waiting': return 'Waiting'
    case 'arrived': return 'Arrived'
    case 'in_progress': return 'In progress'
    case 'completed': return 'Completed'
  }
}
