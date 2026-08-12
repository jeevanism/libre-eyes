import { useMutation } from '@tanstack/react-query'
import { FlaskConical, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { ApiError, episodesAPI, type IOPDraftPayload } from '../../api/client'
import { developmentIOPProfileCode, developmentIOPValues } from './demoIOPCatalogue'

interface IOPDraftDemoProps {
  csrfToken: string
  episodeId: string
}

export function IOPDraftDemo({ csrfToken, episodeId }: IOPDraftDemoProps) {
  const [rightValueCode, setRightValueCode] = useState('')
  const [leftValueCode, setLeftValueCode] = useState('')
  const [clientError, setClientError] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const saveDraft = useMutation({
    mutationFn: (payload: IOPDraftPayload) => episodesAPI.createIOPDraft(episodeId, csrfToken, payload),
  })

  function clearFeedback() {
    setClientError('')
    saveDraft.reset()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const eyes: IOPDraftPayload['eyes'] = []
    if (rightValueCode) eyes.push({ eye: 'right', valueCode: rightValueCode })
    if (leftValueCode) eyes.push({ eye: 'left', valueCode: leftValueCode })
    if (eyes.length === 0) {
      saveDraft.reset()
      setClientError('Select a development demonstration value for at least one eye before saving the draft.')
      return
    }
    setClientError('')
    saveDraft.mutate({ recordMode: 'development_raw_mmhg', profileCode: developmentIOPProfileCode, eyes })
  }

  const failure = saveDraft.error instanceof ApiError && saveDraft.error.status === 409
    ? 'This draft could not be saved because the episode changed. Refresh and try again.'
    : saveDraft.isError ? 'The demo draft could not be saved. No clinical observation was created.' : ''

  useEffect(() => {
    if (clientError || failure) alertRef.current?.focus()
  }, [clientError, failure])

  return (
    <section className="examination-draft-demo" aria-labelledby={`iop-demo-${episodeId}`}>
      <div className="examination-draft-demo-heading">
        <div><p>Demo</p><h3 id={`iop-demo-${episodeId}`}>Intraocular Pressure draft</h3></div>
        <FlaskConical size={18} aria-hidden="true" />
      </div>
      <p className="examination-draft-demo-note">This stores a temporary, clinician-owned demonstration draft only. It does not create a clinical observation.</p>
      <form className="examination-draft-demo-form iop-draft-demo-form" onSubmit={submit}>
        <EyeSelect eye="Right eye" disabled={saveDraft.isPending} valueCode={rightValueCode} onChange={(value) => { clearFeedback(); setRightValueCode(value) }} />
        <EyeSelect eye="Left eye" disabled={saveDraft.isPending} valueCode={leftValueCode} onChange={(value) => { clearFeedback(); setLeftValueCode(value) }} />
        <div className="examination-draft-demo-actions">
          {(clientError || failure) && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{clientError || failure}</p>}
          {saveDraft.isSuccess && <p className="inline-success" role="status">Demo draft saved. It remains uncommitted.</p>}
          <button className="primary-button" disabled={saveDraft.isPending} type="submit"><Save size={16} aria-hidden="true" />{saveDraft.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
        </div>
      </form>
    </section>
  )
}

function EyeSelect({ eye, valueCode, disabled, onChange }: { eye: string, valueCode: string, disabled: boolean, onChange: (value: string) => void }) {
  return (
    <fieldset className="examination-draft-eye">
      <legend>{eye}</legend>
      <label>
        <span>Demo IOP value</span>
        <select aria-label={`${eye} development IOP value`} disabled={disabled} onChange={(event) => onChange(event.target.value)} value={valueCode}>
          <option value="">Not recorded in this demonstration draft</option>
          {developmentIOPValues.map(([code, label]) => <option key={code} value={code}>{label}</option>)}
        </select>
      </label>
    </fieldset>
  )
}
