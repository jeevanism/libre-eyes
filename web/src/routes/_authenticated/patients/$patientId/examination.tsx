import { createFileRoute } from '@tanstack/react-router'

import { PatientExaminationWorkspace } from '../../../../features/examination/PatientExaminationWorkspace'
import { isExaminationTool, type ExaminationTool } from '../../../../features/examination/examinationNavigation'

export const Route = createFileRoute('/_authenticated/patients/$patientId/examination')({
  validateSearch: (search: Record<string, unknown>): { tool?: ExaminationTool } => {
    return isExaminationTool(search.tool) ? { tool: search.tool } : {}
  },
  component: PatientExaminationRoute,
})

function PatientExaminationRoute() {
  const { patientId } = Route.useParams()
  const { session } = Route.useRouteContext()
  const { tool } = Route.useSearch()
  const navigate = Route.useNavigate()
  return <PatientExaminationWorkspace patientId={patientId} session={session} selectedTool={tool ?? 'acuity'} onToolChange={(nextTool) => { void navigate({ search: (current) => ({ ...current, tool: nextTool }) }) }} />
}
