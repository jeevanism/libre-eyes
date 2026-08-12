import { useMutation } from '@tanstack/react-query'
import { FlaskConical, Save } from 'lucide-react'
import { useState } from 'react'
import type { FormEvent } from 'react'

import { ApiError, episodesAPI, type VisualAcuityDraftPayload } from '../../api/client'
import {
  developmentVisualAcuityMethods,
  developmentVisualAcuityUnit,
  developmentVisualAcuityValues,
} from './demoVisualAcuityCatalogue'

interface VisualAcuityDraftDemoProps {
  csrfToken: string
  episodeId: string
}

export function VisualAcuityDraftDemo({ csrfToken, episodeId }: VisualAcuityDraftDemoProps) {
  const [rightValueCode, setRightValueCode] = useState('')
  const [leftValueCode, setLeftValueCode] = useState('')
  const [rightMethodCode, setRightMethodCode] = useState('development_unaided')
  const [leftMethodCode, setLeftMethodCode] = useState('development_unaided')
  const [clientError, setClientError] = useState('')
  const saveDraft = useMutation({
    mutationFn: (payload: VisualAcuityDraftPayload) => episodesAPI.createVisualAcuityDraft(episodeId, csrfToken, payload),
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!rightValueCode || !leftValueCode) {
      saveDraft.reset()
      setClientError('Select a demonstration value for both eyes before saving the draft.')
      return
    }
    setClientError('')
    saveDraft.mutate({
      recordMode: 'simple',
      eyes: [
        { eye: 'right', assessment: 'recorded', readings: [{ unitCode: developmentVisualAcuityUnit, valueCode: rightValueCode, methodCode: rightMethodCode }] },
        { eye: 'left', assessment: 'recorded', readings: [{ unitCode: developmentVisualAcuityUnit, valueCode: leftValueCode, methodCode: leftMethodCode }] },
      ],
    })
  }

  function clearFeedback() {
    setClientError('')
    saveDraft.reset()
  }

  const failure = saveDraft.error instanceof ApiError && saveDraft.error.status === 409
    ? 'This draft could not be saved because the episode changed. Refresh and try again.'
    : saveDraft.isError ? 'The demo draft could not be saved. No clinical observation was created.' : ''

  return (
    <section className="visual-acuity-demo" aria-labelledby={`visual-acuity-demo-${episodeId}`}>
      <div className="visual-acuity-demo-heading">
        <div>
          <p>Demo</p>
          <h3 id={`visual-acuity-demo-${episodeId}`}>Visual Acuity draft</h3>
        </div>
        <FlaskConical size={18} aria-hidden="true" />
      </div>
      <p className="visual-acuity-demo-note">This uses the non-production 4 m LogMAR demonstration reference profile and stores a temporary, clinician-owned draft only. It does not create a clinical observation.</p>
      <form className="visual-acuity-demo-form" onSubmit={submit}>
        <EyeControls
          eye="Right eye"
          methodCode={rightMethodCode}
          disabled={saveDraft.isPending}
          onMethodChange={(value) => { clearFeedback(); setRightMethodCode(value) }}
          onValueChange={(value) => { clearFeedback(); setRightValueCode(value) }}
          valueCode={rightValueCode}
        />
        <EyeControls
          eye="Left eye"
          methodCode={leftMethodCode}
          disabled={saveDraft.isPending}
          onMethodChange={(value) => { clearFeedback(); setLeftMethodCode(value) }}
          onValueChange={(value) => { clearFeedback(); setLeftValueCode(value) }}
          valueCode={leftValueCode}
        />
        <div className="visual-acuity-demo-actions">
          {(clientError || failure) && <p className="inline-error" role="alert">{clientError || failure}</p>}
          {saveDraft.isSuccess && <p className="inline-success" role="status">Demo draft saved. It remains uncommitted.</p>}
          <button className="primary-button" disabled={saveDraft.isPending} type="submit">
            <Save size={16} aria-hidden="true" />
            {saveDraft.isPending ? 'Saving draft…' : 'Save demo draft'}
          </button>
        </div>
      </form>
    </section>
  )
}

function EyeControls({ eye, valueCode, methodCode, disabled, onValueChange, onMethodChange }: {
  eye: string
  valueCode: string
  methodCode: string
  disabled: boolean
  onValueChange: (value: string) => void
  onMethodChange: (value: string) => void
}) {
  const valueLabel = `${eye} demonstration value`
  const methodLabel = `${eye} demonstration method`
  return (
    <fieldset className="visual-acuity-eye">
      <legend>{eye}</legend>
      <label>
        <span>Demo LogMAR value</span>
        <select aria-label={valueLabel} disabled={disabled} onChange={(event) => onValueChange(event.target.value)} value={valueCode}>
          <option value="">Select value</option>
          {developmentVisualAcuityValues.map(([code, label]) => <option key={code} value={code}>{label}</option>)}
        </select>
      </label>
      <label>
        <span>Method</span>
        <select aria-label={methodLabel} disabled={disabled} onChange={(event) => onMethodChange(event.target.value)} value={methodCode}>
          {developmentVisualAcuityMethods.map(([code, label]) => <option key={code} value={code}>{label}</option>)}
        </select>
      </label>
    </fieldset>
  )
}
