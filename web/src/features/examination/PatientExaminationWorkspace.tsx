import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, Stethoscope } from 'lucide-react'

import { patientSummaryAPI, type Session } from '../../api/client'
import { EpisodeTimeline } from '../episodes/EpisodeTimeline'
import type { ExaminationTool } from './examinationNavigation'
import { presentationTerminology } from '../../config/terminology'

interface PatientExaminationWorkspaceProps {
  patientId: string
  session: Session
  selectedTool: ExaminationTool
  onToolChange: (tool: ExaminationTool) => void
}

export function PatientExaminationWorkspace({ patientId, session, selectedTool, onToolChange }: PatientExaminationWorkspaceProps) {
  const header = useQuery({
    queryKey: ['patient-summary-header', patientId, session.contextVersion],
    queryFn: () => patientSummaryAPI.header(patientId, session.csrfToken, session.contextVersion),
  })

  if (header.isPending) return <div className="route-status">Loading examination workspace…</div>
  if (header.isError) return <div className="route-status route-error" role="alert">Examination workspace is temporarily unavailable.</div>
  const name = [header.data.givenName, header.data.familyName].filter(Boolean).join(' ') || 'Name not recorded'

  return (
    <section className="patient-examination-page" aria-labelledby="patient-examination-title">
      <div className="workspace-title examination-page-title">
        <div>
          <p>Selected patient</p>
          <h1 id="patient-examination-title">{name}</h1>
        </div>
        <Link className="secondary-button" params={{ patientId }} to="/patients/$patientId">
          <ArrowLeft size={16} aria-hidden="true" />Patient summary
        </Link>
      </div>
      <div className="examination-page-intro">
        <Stethoscope size={20} aria-hidden="true" />
        <div><h2>Examination workspace</h2><span>Select an active care episode, then work in one {presentationTerminology.environmentLabel.toLowerCase()} tool at a time.</span></div>
      </div>
      <EpisodeTimeline
        allowed={session.permissions.includes('episode.read')}
        canCreateExaminationDraft={session.permissions.includes('event_draft.create')}
        contextVersion={session.contextVersion}
        csrfToken={session.csrfToken}
        patientId={patientId}
        workspace="examination"
        selectedTool={selectedTool}
        onToolChange={onToolChange}
      />
    </section>
  )
}
