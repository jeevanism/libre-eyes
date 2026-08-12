import { useMutation } from '@tanstack/react-query'
import { ClipboardPenLine, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { ApiError, episodesAPI, type OperativeNoteDemoDraftPayload } from '../../api/client'

interface OperativeNoteDraftDemoProps { csrfToken: string; episodeId: string }

const procedures = [
  ['development_cataract_extraction', 'Demo cataract extraction'],
  ['development_trabeculectomy', 'Demo trabeculectomy'],
] as const
const lateralities = [
  ['development_right_eye', 'Right eye'],
  ['development_left_eye', 'Left eye'],
  ['development_bilateral', 'Both eyes'],
] as const
const surgeons = [['development_surgeon_a', 'Demo surgeon A'], ['development_surgeon_b', 'Demo surgeon B']] as const
const anaesthetics = [
  ['development_local_anaesthetic', 'Demo local anaesthetic'],
  ['development_general_anaesthetic', 'Demo general anaesthetic'],
  ['development_no_anaesthetic', 'Demo no anaesthetic'],
] as const
const deliveries = [['development_subtenons', "Demo sub-Tenon's"], ['development_topical', 'Demo topical'], ['development_other', 'Demo other']] as const

export function OperativeNoteDraftDemo({ csrfToken, episodeId }: OperativeNoteDraftDemoProps) {
  const [procedureCode, setProcedureCode] = useState<OperativeNoteDemoDraftPayload['procedureCode']>('development_cataract_extraction')
  const [laterality, setLaterality] = useState<OperativeNoteDemoDraftPayload['laterality']>('development_right_eye')
  const [surgeonCode, setSurgeonCode] = useState<OperativeNoteDemoDraftPayload['surgeonCode']>('development_surgeon_a')
  const [anaestheticCode, setAnaestheticCode] = useState<OperativeNoteDemoDraftPayload['anaestheticCode']>('development_local_anaesthetic')
  const [deliveryCodes, setDeliveryCodes] = useState<OperativeNoteDemoDraftPayload['deliveryCodes']>(['development_topical'])
  const [comment, setComment] = useState('')
  const [clientError, setClientError] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const saveDraft = useMutation({ mutationFn: (payload: OperativeNoteDemoDraftPayload) => episodesAPI.createOperativeNoteDemoDraft(episodeId, csrfToken, payload) })

  function changeAnaesthetic(value: OperativeNoteDemoDraftPayload['anaestheticCode']) {
    setAnaestheticCode(value)
    if (value !== 'development_local_anaesthetic') setDeliveryCodes([])
    else if (deliveryCodes.length === 0) setDeliveryCodes(['development_topical'])
    setClientError(''); saveDraft.reset()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!procedureCode || !laterality || !surgeonCode || !anaestheticCode || (anaestheticCode === 'development_local_anaesthetic' && deliveryCodes.length === 0)) {
      setClientError('Complete the demo procedure, laterality, surgeon, and anaesthetic fields before saving.')
      saveDraft.reset(); return
    }
    setClientError('')
    saveDraft.mutate({ recordMode: 'development_synthetic_operative_note', procedureCode, laterality, surgeonCode, anaestheticCode, deliveryCodes, comment })
  }

  const failure = saveDraft.error instanceof ApiError && saveDraft.error.status === 409
    ? 'This draft could not be saved because the episode changed. Refresh and try again.'
    : saveDraft.isError ? 'The demo draft could not be saved. No clinical record was created.' : ''
  const error = clientError || failure
  useEffect(() => { if (error) alertRef.current?.focus() }, [error])

  return <section className="examination-draft-demo" aria-labelledby={`operative-note-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`operative-note-demo-${episodeId}`}>Operative-note draft</h3></div><ClipboardPenLine size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores a synthetic, clinician-owned draft only. It does not create a clinical operative note or trigger booking, consent, prescription, diagnosis, or correspondence.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo procedure</span><select aria-label="Demo procedure" disabled={saveDraft.isPending} value={procedureCode} onChange={event => { setProcedureCode(event.target.value as OperativeNoteDemoDraftPayload['procedureCode']); setClientError(''); saveDraft.reset() }}>{procedures.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
      <label><span>Laterality</span><select aria-label="Operative-note laterality" disabled={saveDraft.isPending} value={laterality} onChange={event => setLaterality(event.target.value as OperativeNoteDemoDraftPayload['laterality'])}>{lateralities.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
      <label><span>Demo surgeon</span><select aria-label="Demo surgeon" disabled={saveDraft.isPending} value={surgeonCode} onChange={event => setSurgeonCode(event.target.value as OperativeNoteDemoDraftPayload['surgeonCode'])}>{surgeons.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
      <label><span>Anaesthetic</span><select aria-label="Demo anaesthetic" disabled={saveDraft.isPending} value={anaestheticCode} onChange={event => changeAnaesthetic(event.target.value as OperativeNoteDemoDraftPayload['anaestheticCode'])}>{anaesthetics.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
      {anaestheticCode === 'development_local_anaesthetic' && <fieldset><legend>Delivery</legend>{deliveries.map(([code, label]) => <label key={code}><input type="checkbox" checked={deliveryCodes.includes(code)} disabled={saveDraft.isPending} onChange={event => setDeliveryCodes(current => event.target.checked ? [...current, code] : current.filter(item => item !== code))} />{label}</label>)}</fieldset>}
      <label><span>Demo comments</span><textarea aria-label="Demo comments" maxLength={2000} disabled={saveDraft.isPending} value={comment} onChange={event => setComment(event.target.value)} rows={3} /></label>
      <div className="examination-draft-demo-actions">{error && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{error}</p>}{saveDraft.isSuccess && <p className="inline-success" role="status">Demo draft saved. It remains uncommitted.</p>}<button className="primary-button" disabled={saveDraft.isPending} type="submit"><Save size={16} aria-hidden="true" />{saveDraft.isPending ? 'Saving draft…' : 'Save demo draft'}</button></div>
    </form>
  </section>
}
