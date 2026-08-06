import { createFileRoute } from '@tanstack/react-router'

import { PatientSearchPage } from '../../../features/patient-search/PatientSearchPage'

export const Route = createFileRoute('/_authenticated/patients/search')({
  component: PatientSearchRoute,
})

function PatientSearchRoute() {
  const { session } = Route.useRouteContext()
  return <PatientSearchPage session={session} />
}
