import { FlaskConical, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { FormEvent } from 'react'

import { episodesAPI, type LabResultDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

const catalogue = [
  { code: 'demo_lab_hba1c', label: 'Demo HbA1c', kind: 'numeric', unit: '%', min: 0, max: 20, normalMin: 4, normalMax: 6 },
  { code: 'demo_lab_creatinine', label: 'Demo serum creatinine', kind: 'numeric', unit: 'umol/L', min: 0, max: 2000, normalMin: 45, normalMax: 110 },
  { code: 'demo_lab_status', label: 'Demo laboratory status', kind: 'choice', unit: '', choices: ['pending', 'complete', 'not available'] },
] as const

export function LabResultsDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [typeCode, setTypeCode] = useState<(typeof catalogue)[number]['code']>('demo_lab_hba1c')
  const [value, setValue] = useState('5.5')
  const [observedAt, setObservedAt] = useState('09:30')
  const [comment, setComment] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const selected = catalogue.find((item) => item.code === typeCode) ?? catalogue[0]
  const save = useMutation({ mutationFn: (payload: LabResultDemoDraftPayload) => episodesAPI.createLabResultDemoDraft(episodeId, csrfToken, payload) })
  const numericValue = selected.kind === 'numeric' ? Number(value) : null
  const warning = selected.kind === 'numeric' && numericValue !== null && Number.isFinite(numericValue) && (numericValue < selected.normalMin || numericValue > selected.normalMax)
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_lab_result', isSynthetic: true, resultTypeCode: typeCode, fieldKind: selected.kind, value, unit: selected.unit, observedAt, comment }) }
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo lab-result draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`lab-results-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`lab-results-demo-${episodeId}`}>Lab result draft</h3></div><FlaskConical size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores a synthetic, clinician-owned draft only. It does not create a clinical result, interpretation, alert, or device import.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo result type</span><select value={typeCode} onChange={(event) => { setTypeCode(event.target.value as typeof typeCode); setValue('') }} disabled={save.isPending}>{catalogue.map((item) => <option key={item.code} value={item.code}>{item.label}</option>)}</select></label>
      <label><span>Demo value</span>{selected.kind === 'choice' ? <select value={value} onChange={(event) => setValue(event.target.value)} disabled={save.isPending}><option value="">Select value</option>{selected.choices.map((choice) => <option key={choice} value={choice}>{choice}</option>)}</select> : <input inputMode="decimal" value={value} onChange={(event) => setValue(event.target.value)} disabled={save.isPending} />}</label>
      <label><span>Unit</span><input value={selected.unit} readOnly aria-readonly="true" /></label>
      <label><span>Observed time</span><input type="time" value={observedAt} onChange={(event) => setObservedAt(event.target.value)} disabled={save.isPending} /></label>
      <label><span>Demo comment</span><textarea maxLength={1000} rows={2} value={comment} onChange={(event) => setComment(event.target.value)} disabled={save.isPending} /></label>
      {warning && <p className="inline-warning" role="status">Outside the demo normal range; saving remains allowed.</p>}
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo lab-result draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending || !value.trim()}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
