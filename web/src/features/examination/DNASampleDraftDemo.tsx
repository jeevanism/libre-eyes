import { Dna, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { FormEvent } from 'react'
import { episodesAPI, type DNASampleDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function DNASampleDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [sampleType, setSampleType] = useState<DNASampleDemoDraftPayload['sampleType']>('demo_blood')
  const [consentedBy, setConsentedBy] = useState<DNASampleDemoDraftPayload['consentedBy']>('demo_clinician')
  const [sampleDate, setSampleDate] = useState('')
  const [volume, setVolume] = useState('10')
  const [comment, setComment] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: DNASampleDemoDraftPayload) => episodesAPI.createDNASampleDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_dna_sample', profileCode: 'demo_dna_sample_v1', sampleType, consentedBy, sampleDate, volume: Number(volume), comment }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo DNA-sample draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`dna-sample-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`dna-sample-demo-${episodeId}`}>DNA sample draft</h3></div><Dna size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores synthetic sample metadata only. It does not create a genetic result, research study, laboratory transfer, or clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo sample type</span><select value={sampleType} onChange={(e) => setSampleType(e.target.value as DNASampleDemoDraftPayload['sampleType'])} disabled={save.isPending}><option value="demo_blood">Demo blood</option><option value="demo_saliva">Demo saliva</option><option value="demo_other">Demo other</option></select></label>
      <label><span>Demo consented by</span><select value={consentedBy} onChange={(e) => setConsentedBy(e.target.value as DNASampleDemoDraftPayload['consentedBy'])} disabled={save.isPending}><option value="demo_clinician">Demo clinician</option><option value="demo_patient">Demo patient</option><option value="demo_guardian">Demo guardian</option></select></label>
      <label><span>Sample date</span><input type="date" value={sampleDate} onChange={(e) => setSampleDate(e.target.value)} disabled={save.isPending} /><small>Today or an earlier date only.</small></label>
      <label><span>Demo volume (1–99)</span><input inputMode="numeric" value={volume} onChange={(e) => setVolume(e.target.value)} disabled={save.isPending} /></label>
      <label><span>Demo comment</span><textarea rows={2} maxLength={500} value={comment} onChange={(e) => setComment(e.target.value)} disabled={save.isPending} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo DNA-sample draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending || !sampleDate || Number(volume) < 1 || Number(volume) > 99}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
