import { useQuery } from '@tanstack/react-query'
import { ArrowRight, ClipboardList, Search, Users } from 'lucide-react'
import { useState, type FormEvent } from 'react'

import { patientSearchAPI, developmentClinicFlowAPI, type Session } from '../../api/client'

const statusLabel: Record<string, string> = {
  waiting: 'Waiting',
  arrived: 'Arrived',
  in_progress: 'In progress',
  completed: 'Completed',
}

export function HomeDashboard({ session }: { session: Session }) {
  const canSearch = session.permissions.includes('patient.search')
  const canSeeQueue = session.capabilities?.includes('clinic_flow') !== false && session.permissions.includes('worklist.development_flow.manage')
  const recent = useQuery({
    queryKey: ['patients', 'recent'],
    queryFn: () => patientSearchAPI.recent(5, session.csrfToken),
    enabled: canSearch,
  })
  const queue = useQuery({
    queryKey: ['development-clinic-flow', session.contextVersion],
    queryFn: () => developmentClinicFlowAPI.list(session.csrfToken),
    enabled: canSeeQueue,
  })

  return (
    <div className="home-dashboard">
      <header className="page-header home-dashboard-header">
        <div>
          <p className="page-eyebrow">Clinical workspace</p>
          <h1>Today’s work</h1>
          <p className="home-dashboard-intro">Find a patient or review the synthetic demonstration queue for this clinic context.</p>
        </div>
        <div className="home-context-summary"><span>{session.context.site.name}</span><strong>{session.context.firm.name}</strong></div>
      </header>

      {canSearch && <QuickPatientLookup session={session} />}

      <div className="home-dashboard-grid">
        {canSeeQueue && <section className="home-dashboard-section" aria-labelledby="home-queue-heading">
          <div className="home-section-heading"><div><p className="page-eyebrow">Demo queue</p><h2 id="home-queue-heading">Clinic flow</h2></div><ClipboardList size={19} aria-hidden="true" /></div>
          <p className="home-section-note">Read-only overview. Use Clinic flow to update synthetic ticket state.</p>
          {queue.isPending && <p className="summary-muted">Loading queue…</p>}
          {queue.isError && <p className="inline-error" role="alert">The demonstration queue is temporarily unavailable.</p>}
          {queue.data?.items.length === 0 && <p className="summary-muted">No synthetic tickets are available in this context.</p>}
          {queue.data?.items.length ? <ul className="home-list">{queue.data.items.map(ticket => <li key={ticket.id}><div><strong>{ticket.syntheticPatientLabel}</strong>{ticket.assigneeDisplayName && <span>{ticket.assigneeDisplayName}</span>}</div><span className={`home-status home-status-${ticket.status}`}>{statusLabel[ticket.status] ?? ticket.status}</span></li>)}</ul> : null}
          <a className="home-section-link" href="/clinic-flow">Open Clinic flow <ArrowRight size={15} aria-hidden="true" /></a>
        </section>}

        {canSearch && <section className="home-dashboard-section" aria-labelledby="home-recent-heading">
          <div className="home-section-heading"><div><p className="page-eyebrow">Current institution</p><h2 id="home-recent-heading">Recently updated</h2></div><Users size={19} aria-hidden="true" /></div>
          <p className="home-section-note">Minimum-disclosure patient list, limited to five records.</p>
          {recent.isPending && <p className="summary-muted">Loading recent patients…</p>}
          {recent.isError && <p className="inline-error" role="alert">Recent patients could not be loaded. Use search instead.</p>}
          {recent.data?.items.length === 0 && <p className="summary-muted">No recent patients in this context.</p>}
          {recent.data?.items.length ? <ul className="home-list">{recent.data.items.map(patient => <li key={patient.patientId}><a href={`/patients/${patient.patientId}`}>{patient.fullName ?? 'Unnamed patient'}</a><span>{patient.dateOfBirth}</span></li>)}</ul> : null}
          <a className="home-section-link" href="/patients/search">Open patient search <ArrowRight size={15} aria-hidden="true" /></a>
        </section>}
      </div>
    </div>
  )
}

function QuickPatientLookup({ session }: { session: Session }) {
  const [familyName, setFamilyName] = useState('')
  const [givenName, setGivenName] = useState('')
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [gender, setGender] = useState<'female' | 'male' | 'other' | 'unknown'>('unknown')
  const [error, setError] = useState('')
  const [results, setResults] = useState<Awaited<ReturnType<typeof patientSearchAPI.search>>>()
  const [loading, setLoading] = useState(false)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!familyName.trim() || !dateOfBirth) {
      setError('Enter a family name and date of birth for an exact lookup.')
      setResults(undefined)
      return
    }
    setError('')
    setLoading(true)
    try {
      const response = await patientSearchAPI.search({
        criteria: { kind: 'demographic', familyName: familyName.trim(), ...(givenName.trim() ? { givenName: givenName.trim() } : {}), dateOfBirth, gender },
        limit: 10,
      }, session.csrfToken)
      setResults(response)
    } catch {
      setError('Patient lookup could not be completed. Use Patient search for more options.')
      setResults(undefined)
    } finally {
      setLoading(false)
    }
  }

  return <section className="home-lookup" aria-labelledby="home-lookup-heading">
    <div className="home-section-heading"><div><p className="page-eyebrow">Patient access</p><h2 id="home-lookup-heading">Quick patient lookup</h2></div><Search size={19} aria-hidden="true" /></div>
    <p className="home-section-note">Exact demographic lookup within the current institution. Identifier search is available in Patient search.</p>
    <form className="home-lookup-form" onSubmit={submit} noValidate>
      <label>Given name <span>(optional)</span><input value={givenName} onChange={event => setGivenName(event.target.value)} autoComplete="off" /></label>
      <label>Family name<input value={familyName} onChange={event => setFamilyName(event.target.value)} autoComplete="off" /></label>
      <label>Date of birth<input type="date" value={dateOfBirth} onChange={event => setDateOfBirth(event.target.value)} /></label>
      <label>Gender<select value={gender} onChange={event => setGender(event.target.value as typeof gender)}><option value="unknown">Unknown</option><option value="female">Female</option><option value="male">Male</option><option value="other">Other</option></select></label>
      <button className="primary-button" type="submit" disabled={loading}>{loading ? 'Looking up…' : 'Find patient'}</button>
    </form>
    {error && <p className="inline-error" role="alert">{error}</p>}
    {results && <div className="home-lookup-results" aria-live="polite">{results.items.length ? results.items.map(patient => <a key={patient.patientId} href={`/patients/${patient.patientId}`}><span>{patient.fullName ?? 'Unnamed patient'}</span><small>{patient.dateOfBirth}</small><ArrowRight size={15} aria-hidden="true" /></a>) : <p className="summary-muted">No exact patients matched those details.</p>}</div>}
  </section>
}
