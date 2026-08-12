import { useQuery } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import { AlertTriangle, ShieldAlert, Stethoscope } from 'lucide-react'

import { ApiError, patientSummaryAPI } from '../../../../api/client'
import { EpisodeTimeline } from '../../../../features/episodes/EpisodeTimeline'

export const Route = createFileRoute('/_authenticated/patients/$patientId/')({
  component: PatientSummaryRoute,
})

function PatientSummaryRoute() {
  const { patientId } = Route.useParams()
  const { session } = Route.useRouteContext()
  const header = useQuery({
    queryKey: ['patient-summary-header', patientId, session.contextVersion],
    queryFn: () => patientSummaryAPI.header(patientId, session.csrfToken, session.contextVersion),
  })
  const canReadWarnings = session.permissions.includes('patient.clinical_summary.read')
  const warnings = useQuery({
    queryKey: ['patient-summary-warnings', patientId, session.contextVersion],
    queryFn: () => patientSummaryAPI.warnings(patientId, session.csrfToken, session.contextVersion),
    enabled: canReadWarnings && header.isSuccess,
  })

  if (header.isPending) return <div className="route-status">Loading patient summary…</div>
  if (header.isError) return <SummaryError error={header.error} />
  const patient = header.data
  const name = [patient.givenName, patient.familyName].filter(Boolean).join(' ') || 'Name not recorded'

  return (
    <section className="patient-summary-page" aria-labelledby="patient-summary-title">
      <div className="workspace-title patient-summary-title">
        <div><p>Selected patient</p><h1 id="patient-summary-title">{name}</h1></div>
        <Link className="primary-button" params={{ patientId }} to="/patients/$patientId/examination"><Stethoscope size={16} aria-hidden="true" />Open examination workspace</Link>
      </div>
      <div className="patient-summary-grid">
        <section className="summary-panel" aria-labelledby="identity-title">
          <div className="summary-panel-heading"><h2 id="identity-title">Identity</h2><span className="status-badge status-current">Verified context</span></div>
          <dl className="summary-details">
            <div><dt>Date of birth</dt><dd>{formatDate(patient.dateOfBirth)}</dd></div>
            <div><dt>Age</dt><dd>{patient.ageYears === null ? 'Unknown' : `${patient.ageYears} years`}</dd></div>
            <div><dt>Administrative sex</dt><dd>{patient.gender}</dd></div>
            <div><dt>Patient status</dt><dd>{patient.deceased ? `Deceased${patient.dateOfDeath ? ` (${formatDate(patient.dateOfDeath)})` : ''}` : 'Current'}</dd></div>
          </dl>
        </section>
        <section className="summary-panel" aria-labelledby="warning-title">
          <div className="summary-panel-heading"><h2 id="warning-title">Clinical warnings</h2><ShieldAlert size={18} aria-hidden="true" /></div>
          {!canReadWarnings && <p className="summary-muted">Clinical warning details are withheld for this session.</p>}
          {canReadWarnings && warnings.isPending && <p className="summary-muted">Loading warning details…</p>}
          {canReadWarnings && warnings.isError && <SummaryError error={warnings.error} />}
          {canReadWarnings && warnings.data && <WarningList details={warnings.data} />}
        </section>
      </div>
      <EpisodeTimeline
        patientId={patientId}
        csrfToken={session.csrfToken}
        contextVersion={session.contextVersion}
        allowed={session.permissions.includes('episode.read')}
        canCreateExaminationDraft={false}
      />
    </section>
  )
}

function formatDate(value: string): string {
  const [year = 0, month = 1, day = 1] = value.split('-').map(Number)
  return new Intl.DateTimeFormat('en-GB', { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' }).format(new Date(Date.UTC(year, month - 1, day)))
}

function WarningList({ details }: { details: Awaited<ReturnType<typeof patientSummaryAPI.warnings>> }) {
  if (details.items.length === 0) return <p className="summary-muted">No warning items recorded.</p>
  return <div className="warning-list" aria-live="polite">{details.items.map((item, index) => <div className="warning-item" key={`${item.kind}-${item.code ?? index}`}><AlertTriangle size={16} aria-hidden="true" /><div><strong>{item.label}</strong><span>{item.kind}</span>{item.reaction && <p>{item.reaction}</p>}</div></div>)}</div>
}

function SummaryError({ error }: { error: Error }) {
  const message = error instanceof ApiError && error.status === 404 ? 'Patient summary is not available.' : 'Patient summary is temporarily unavailable.'
  return <div className="route-status route-error" role="alert">{message}</div>
}
