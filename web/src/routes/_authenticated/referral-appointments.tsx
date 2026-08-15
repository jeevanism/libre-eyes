import { createFileRoute } from '@tanstack/react-router'
import { DevelopmentReferralAppointmentBoard } from '../../features/referral/DevelopmentReferralAppointmentBoard'
export const Route = createFileRoute('/_authenticated/referral-appointments')({ component: ReferralAppointmentsRoute })
function ReferralAppointmentsRoute() { const { session } = Route.useRouteContext(); return <section className="clinic-flow-page" aria-labelledby="referral-appointments-page-title"><div className="workspace-title"><p>Demo</p><h1 id="referral-appointments-page-title">Referrals and appointments</h1></div><DevelopmentReferralAppointmentBoard allowed={session.permissions.includes('referral.development_appointment.manage')} contextVersion={session.contextVersion} csrfToken={session.csrfToken} /></section> }
