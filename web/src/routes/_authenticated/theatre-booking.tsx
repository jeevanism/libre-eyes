import { createFileRoute } from '@tanstack/react-router'

import { DevelopmentTheatreBookingBoard } from '../../features/theatrebooking/DevelopmentTheatreBookingBoard'

export const Route = createFileRoute('/_authenticated/theatre-booking')({ component: TheatreBookingRoute })

function TheatreBookingRoute() {
  const { session } = Route.useRouteContext()
  if (session.capabilities && !session.capabilities.includes('theatre_booking')) return <section className="workspace-title"><p>Demo</p><h1>Capability unavailable</h1><span>Theatre scheduling is disabled for this clinic profile.</span></section>
  return <section className="clinic-flow-page" aria-labelledby="theatre-booking-page-title"><div className="workspace-title"><p>Demo</p><h1 id="theatre-booking-page-title">Theatre schedule</h1></div><DevelopmentTheatreBookingBoard allowed={session.permissions.includes('theatre.development_booking.manage')} contextVersion={session.contextVersion} csrfToken={session.csrfToken} /></section>
}
