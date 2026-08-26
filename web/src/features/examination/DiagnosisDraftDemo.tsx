import { useMutation } from '@tanstack/react-query'
import { FlaskConical, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { episodesAPI, type DiagnosisDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'
import { developmentDiagnosisProfileCode, developmentDiagnosisSelections } from './demoDiagnosisCatalogue'

interface DiagnosisDraftDemoProps {
  csrfToken: string
  episodeId: string
}

const developmentTimeZone = 'Europe/London'

function developmentToday(): string {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: developmentTimeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date())
  const value = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  return `${value.year}-${value.month}-${value.day}`
}

export function DiagnosisDraftDemo({ csrfToken, episodeId }: DiagnosisDraftDemoProps) {
  const [selectionCode, setSelectionCode] = useState('')
  const [laterality, setLaterality] = useState('')
  const [diagnosisDate, setDiagnosisDate] = useState(developmentToday)
  const [clientError, setClientError] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const saveDraft = useMutation({
    mutationFn: (payload: DiagnosisDemoDraftPayload) => episodesAPI.createDiagnosisDemoDraft(episodeId, csrfToken, payload),
  })

  function clearFeedback() {
    setClientError('')
    saveDraft.reset()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selectionCode || !laterality || !diagnosisDate) {
      saveDraft.reset()
      setClientError('Select a demonstration example, laterality, and date before saving the draft.')
      return
    }
    if (diagnosisDate > developmentToday()) {
      saveDraft.reset()
      setClientError('The demonstration date cannot be in the future.')
      return
    }
    clearFeedback()
    saveDraft.mutate({
      recordMode: 'development_synthetic_diagnosis',
      profileCode: developmentDiagnosisProfileCode,
      selectionCode: selectionCode as DiagnosisDemoDraftPayload['selectionCode'],
      laterality: laterality as DiagnosisDemoDraftPayload['laterality'],
      diagnosisDate,
    })
  }

  const failure = saveDraft.isError ? describeDraftSaveError(saveDraft.error) : ''
  const error = clientError || failure

  useEffect(() => {
    if (error) alertRef.current?.focus()
  }, [error])

  return (
    <section className="examination-draft-demo" aria-labelledby={`diagnosis-demo-${episodeId}`}>
      <div className="examination-draft-demo-heading">
        <div>
          <p>Demo</p>
          <h3 id={`diagnosis-demo-${episodeId}`}>Ophthalmology selection draft</h3>
        </div>
        <FlaskConical size={18} aria-hidden="true" />
      </div>
      <p className="examination-draft-demo-note">This stores a temporary, clinician-owned demonstration draft only. It does not create a clinical record.</p>
      <form className="examination-draft-demo-form diagnosis-draft-demo-form" onSubmit={submit}>
        <label>
          <span>Demo example</span>
          <select aria-describedby={error ? `diagnosis-demo-error-${episodeId}` : undefined} aria-invalid={error ? true : undefined} aria-label="Demo example" disabled={saveDraft.isPending} value={selectionCode} onChange={(event) => { clearFeedback(); setSelectionCode(event.target.value) }}>
            <option value="">Select an example</option>
            {developmentDiagnosisSelections.map(([code, label]) => <option key={code} value={code}>{label}</option>)}
          </select>
        </label>
        <label>
          <span>Laterality</span>
          <select aria-describedby={error ? `diagnosis-demo-error-${episodeId}` : undefined} aria-invalid={error ? true : undefined} aria-label="Laterality" disabled={saveDraft.isPending} value={laterality} onChange={(event) => { clearFeedback(); setLaterality(event.target.value) }}>
            <option value="">Select laterality</option>
            <option value="left">Left</option>
            <option value="right">Right</option>
            <option value="bilateral">Bilateral</option>
          </select>
        </label>
        <label>
          <span>Demo date</span>
          <input aria-describedby={error ? `diagnosis-demo-error-${episodeId}` : undefined} aria-invalid={error ? true : undefined} aria-label="Demo date" disabled={saveDraft.isPending} max={developmentToday()} onChange={(event) => { clearFeedback(); setDiagnosisDate(event.target.value) }} type="date" value={diagnosisDate} />
        </label>
        <div className="examination-draft-demo-actions">
          {error && <p className="inline-error" id={`diagnosis-demo-error-${episodeId}`} ref={alertRef} role="alert" tabIndex={-1}>{error}</p>}
          {saveDraft.isSuccess && <p className="inline-success" role="status">Demo draft saved. It remains uncommitted.</p>}
          <button className="primary-button" disabled={saveDraft.isPending} type="submit"><Save size={16} aria-hidden="true" />{saveDraft.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
        </div>
      </form>
    </section>
  )
}
