import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarPlus, Check, Clock, LogIn, X } from 'lucide-react'
import { developmentReferralAppointmentsAPI, type DevelopmentReferralAppointment, ApiError } from '../../api/client'

const key = (contextVersion: number) => ['development-referral-appointments', contextVersion] as const
export function DevelopmentReferralAppointmentBoard({ csrfToken, contextVersion, allowed }: { csrfToken: string, contextVersion: number, allowed: boolean }) {
  const client = useQueryClient()
  const query = useQuery({ queryKey: key(contextVersion), queryFn: () => developmentReferralAppointmentsAPI.list(csrfToken), enabled: allowed })
  const mutation = useMutation({ mutationFn: ({ item, command }: { item: DevelopmentReferralAppointment, command: 'schedule' | 'arrive' | 'complete' | 'abandon' }) => developmentReferralAppointmentsAPI.command(item.id, command, item.version, csrfToken), onSuccess: () => void client.invalidateQueries({ queryKey: key(contextVersion) }) })
  return <section className="summary-panel referral-appointment-board" aria-labelledby="referral-appointment-title">
    <div className="summary-panel-heading"><div><p className="development-kicker">Demo</p><h2 id="referral-appointment-title">Referral and appointment queue</h2></div><CalendarPlus size={18} aria-hidden="true" /></div>
    <p className="development-clinic-flow-note">Synthetic referral coordination only. No NHS e-RS, external delivery, calendar, or clinical record is created.</p>
    {!allowed && <p className="summary-muted">Referral controls are withheld for this session.</p>}
    {allowed && query.isPending && <p className="summary-muted">Loading demo referrals...</p>}
    {allowed && query.isError && <p className="inline-error" role="alert">The demo referral queue is temporarily unavailable.</p>}
    {allowed && query.data && <ul className="development-clinic-flow-list">{query.data.items.map((item) => <ReferralRow key={item.id} item={item} pending={mutation.isPending} onCommand={(command) => mutation.mutate({ item, command })} />)}</ul>}
    {mutation.error instanceof ApiError && <p className="inline-error" role="alert">The selected demo referral changed. Refresh and try again.</p>}
  </section>
}
function ReferralRow({ item, pending, onCommand }: { item: DevelopmentReferralAppointment, pending: boolean, onCommand: (command: 'schedule' | 'arrive' | 'complete' | 'abandon') => void }) {
  const action = item.status === 'requested' ? ['schedule', 'abandon'] as const : item.status === 'scheduled' ? ['arrive', 'abandon'] as const : item.status === 'arrived' ? ['complete', 'abandon'] as const : []
  return <li className="development-clinic-flow-ticket" aria-busy={pending}><div><strong>{item.syntheticPatientLabel}</strong><span className={`development-flow-status development-flow-status-${item.status}`}>{item.status}</span><span className="development-flow-owner">{item.recipientRole.replace('demo_', '')} · {item.clinicCode.replace('demo_', '').replaceAll('_', ' ')}</span><span>{item.appointmentDate} {item.appointmentTime} · {item.priority}</span></div><div className="development-flow-actions">{action.map((command) => <button key={command} type="button" className="secondary-button" disabled={pending} onClick={() => onCommand(command)}>{command === 'schedule' && <Clock size={15} aria-hidden="true" />}{command === 'arrive' && <LogIn size={15} aria-hidden="true" />}{command === 'complete' && <Check size={15} aria-hidden="true" />}{command === 'abandon' && <X size={15} aria-hidden="true" />}{command.charAt(0).toUpperCase() + command.slice(1)}</button>)}</div></li>
}
