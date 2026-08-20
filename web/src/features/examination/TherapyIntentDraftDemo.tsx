import { Crosshair, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { episodesAPI, type TherapyIntentDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function TherapyIntentDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [treatment, setTreatment] = useState<TherapyIntentDemoDraftPayload['treatment']>('demo_anti_vegf')
  const [laterality, setLaterality] = useState<TherapyIntentDemoDraftPayload['laterality']>('right')
  const [note, setNote] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: TherapyIntentDemoDraftPayload) => episodesAPI.createTherapyIntentDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_therapy_intent', profileCode: 'demo_therapy_intent_v1', treatment, laterality, note: note.trim() }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo therapy-intent draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`therapy-intent-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`therapy-intent-demo-${episodeId}`}>Therapy intent draft</h3></div><Crosshair size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">Planning intent only. It does not assess suitability, funding, diagnosis, booking, prescribing, or create a clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo treatment</span><select value={treatment} onChange={e => setTreatment(e.target.value as TherapyIntentDemoDraftPayload['treatment'])}><option value="demo_anti_vegf">Demo anti-VEGF</option><option value="demo_steroid">Demo steroid</option><option value="demo_observation">Demo observation</option></select></label>
      <label><span>Laterality</span><select value={laterality} onChange={e => setLaterality(e.target.value as TherapyIntentDemoDraftPayload['laterality'])}><option value="right">Right eye</option><option value="left">Left eye</option><option value="bilateral">Both eyes</option><option value="not_applicable">Not applicable</option></select></label>
      <label><span>Demo planning note</span><textarea maxLength={500} rows={4} value={note} onChange={e => setNote(e.target.value)} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo therapy-intent draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
