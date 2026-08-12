import { createFileRoute } from '@tanstack/react-router'

import { PatientExaminationWorkspace } from '../../../../features/examination/PatientExaminationWorkspace'

export const Route = createFileRoute('/_authenticated/patients/$patientId/examination')({
  component: PatientExaminationRoute,
})

function PatientExaminationRoute() {
  const { patientId } = Route.useParams()
  const { session } = Route.useRouteContext()
  return <PatientExaminationWorkspace patientId={patientId} session={session} />
}
