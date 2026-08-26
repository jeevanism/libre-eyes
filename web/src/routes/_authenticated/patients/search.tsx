import { createFileRoute } from '@tanstack/react-router'

import { PatientSearchPage } from '../../../features/patient-search/PatientSearchPage'

export const Route = createFileRoute('/_authenticated/patients/search')({
  component: PatientSearchRoute,
})

function PatientSearchRoute() {
  const { session } = Route.useRouteContext()
  if (session.capabilities && !session.capabilities.includes('patient_search')) return <section className="workspace-title"><p>Demo</p><h1>Capability unavailable</h1><span>Patient search is disabled for this clinic profile.</span></section>
  return <PatientSearchPage session={session} />
}
