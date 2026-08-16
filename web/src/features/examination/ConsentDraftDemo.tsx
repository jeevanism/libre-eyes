import { useMutation } from '@tanstack/react-query'
import { FileCheck2, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { episodesAPI, type ConsentDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

interface Props { csrfToken: string; episodeId: string }

export function ConsentDraftDemo({ csrfToken, episodeId }: Props) {
  const [formTypeCode, setFormTypeCode] = useState<ConsentDemoDraftPayload['formTypeCode']>('development_form_type_1')
  const [procedureCode, setProcedureCode] = useState<ConsentDemoDraftPayload['procedureCode']>('development_cataract_extraction')
  const [laterality, setLaterality] = useState<ConsentDemoDraftPayload['laterality']>('development_right_eye')
  const [anaestheticCode, setAnaestheticCode] = useState<ConsentDemoDraftPayload['anaestheticCode']>('development_local_anaesthetic')
  const [comment, setComment] = useState('')
  const [error, setError] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: ConsentDemoDraftPayload) => episodesAPI.createConsentDemoDraft(episodeId, csrfToken, payload) })
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo consent draft') : ''
  const message = error || failure
  useEffect(() => { if (message) alertRef.current?.focus() }, [message])
  function clearFeedback() { setError(''); save.reset() }
  function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); if (!formTypeCode || !procedureCode || !laterality || !anaestheticCode) { setError('Complete the demo consent fields before saving.'); save.reset(); return }; setError(''); save.mutate({ recordMode: 'development_synthetic_consent', formTypeCode, procedureCode, laterality, anaestheticCode, comment: comment.trim() }) }
  return <section className="examination-draft-demo" aria-labelledby={`consent-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`consent-demo-${episodeId}`}>Consent form draft</h3></div><FileCheck2 size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores a synthetic, clinician-owned draft only. It does not create a legal consent, signature, PDF, booking, or clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo form type</span><select aria-label="Demo form type" value={formTypeCode} disabled={save.isPending} onChange={e => { setFormTypeCode(e.target.value as ConsentDemoDraftPayload['formTypeCode']); setError(''); save.reset() }}>{[['development_form_type_1','Demo form type 1'],['development_form_type_2','Demo form type 2'],['development_form_type_3','Demo form type 3'],['development_form_type_4','Demo form type 4']].map(([c,l]) => <option key={c} value={c}>{l}</option>)}</select></label>
      <label><span>Demo procedure</span><select aria-label="Demo consent procedure" value={procedureCode} disabled={save.isPending} onChange={e => { clearFeedback(); setProcedureCode(e.target.value as ConsentDemoDraftPayload['procedureCode']) }}><option value="development_cataract_extraction">Demo cataract extraction</option><option value="development_trabeculectomy">Demo trabeculectomy</option></select></label>
      <label><span>Laterality</span><select aria-label="Consent laterality" value={laterality} disabled={save.isPending} onChange={e => { clearFeedback(); setLaterality(e.target.value as ConsentDemoDraftPayload['laterality']) }}><option value="development_right_eye">Right eye</option><option value="development_left_eye">Left eye</option><option value="development_both_eyes">Both eyes</option></select></label>
      <label><span>Demo anaesthetic</span><select aria-label="Demo anaesthetic" value={anaestheticCode} disabled={save.isPending} onChange={e => { clearFeedback(); setAnaestheticCode(e.target.value as ConsentDemoDraftPayload['anaestheticCode']) }}><option value="development_local_anaesthetic">Demo local anaesthetic</option><option value="development_general_anaesthetic">Demo general anaesthetic</option><option value="development_no_anaesthetic">Demo no anaesthetic</option></select></label>
      <label><span>Demo comments</span><textarea maxLength={2000} rows={3} value={comment} disabled={save.isPending} onChange={e => setComment(e.target.value)} /></label>
      <div className="examination-draft-demo-actions">{message && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{message}</p>}{save.isSuccess && <p className="inline-success" role="status">Demo consent draft saved. It remains uncommitted.</p>}<button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo consent draft'}</button></div>
    </form>
  </section>
}
