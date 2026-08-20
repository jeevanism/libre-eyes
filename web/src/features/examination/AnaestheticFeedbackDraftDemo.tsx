import { ClipboardList, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { episodesAPI, type AnaestheticFeedbackDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function AnaestheticFeedbackDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [anaesthetic, setAnaesthetic] = useState<AnaestheticFeedbackDemoDraftPayload['anaesthetic']>('demo_local')
  const [satisfaction, setSatisfaction] = useState<AnaestheticFeedbackDemoDraftPayload['satisfaction']>('demo_satisfied')
  const [note, setNote] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: AnaestheticFeedbackDemoDraftPayload) => episodesAPI.createAnaestheticFeedbackDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_anaesthetic_feedback', profileCode: 'demo_anaesthetic_feedback_v1', anaesthetic, satisfaction, note: note.trim() }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo anaesthetic-feedback draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`anaesthetic-feedback-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`anaesthetic-feedback-demo-${episodeId}`}>Anaesthetic feedback draft</h3></div><ClipboardList size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">Synthetic feedback only. It is not a clinical audit, quality metric, patient-safety decision, or clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo anaesthetic</span><select value={anaesthetic} onChange={e => setAnaesthetic(e.target.value as AnaestheticFeedbackDemoDraftPayload['anaesthetic'])}><option value="demo_general">Demo general</option><option value="demo_local">Demo local</option><option value="demo_none">Demo none</option></select></label>
      <label><span>Demo satisfaction</span><select value={satisfaction} onChange={e => setSatisfaction(e.target.value as AnaestheticFeedbackDemoDraftPayload['satisfaction'])}><option value="demo_very_satisfied">Demo very satisfied</option><option value="demo_satisfied">Demo satisfied</option><option value="demo_neutral">Demo neutral</option><option value="demo_dissatisfied">Demo dissatisfied</option></select></label>
      <label><span>Demo feedback note</span><textarea maxLength={500} rows={4} value={note} onChange={e => setNote(e.target.value)} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo anaesthetic-feedback draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
