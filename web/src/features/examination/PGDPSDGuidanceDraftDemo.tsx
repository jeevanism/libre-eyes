import { ClipboardList, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { episodesAPI, type PGDPSDGuidanceDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function PGDPSDGuidanceDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [pathway, setPathway] = useState<PGDPSDGuidanceDemoDraftPayload['pathway']>('demo_pgd')
  const [medicationLabel, setMedicationLabel] = useState('')
  const [laterality, setLaterality] = useState<PGDPSDGuidanceDemoDraftPayload['laterality']>('right')
  const [note, setNote] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: PGDPSDGuidanceDemoDraftPayload) => episodesAPI.createPGDPSDGuidanceDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_pgd_psd_guidance', profileCode: 'demo_pgd_psd_guidance_v1', pathway, medicationLabel: medicationLabel.trim(), laterality, note: note.trim() }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo PGD/PSD guidance draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`pgd-psd-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`pgd-psd-demo-${episodeId}`}>PGD/PSD guidance draft</h3></div><ClipboardList size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">Planning guidance only. It does not assess eligibility, funding, prescribe, administer, sign, print, send, or create a clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo pathway</span><select value={pathway} onChange={e => setPathway(e.target.value as PGDPSDGuidanceDemoDraftPayload['pathway'])}><option value="demo_pgd">Demo PGD</option><option value="demo_psd">Demo PSD</option></select></label>
      <label><span>Demo medication label</span><input required maxLength={120} value={medicationLabel} onChange={e => setMedicationLabel(e.target.value)} /></label>
      <label><span>Laterality</span><select value={laterality} onChange={e => setLaterality(e.target.value as PGDPSDGuidanceDemoDraftPayload['laterality'])}><option value="right">Right eye</option><option value="left">Left eye</option><option value="bilateral">Both eyes</option><option value="not_applicable">Not applicable</option></select></label>
      <label><span>Demo planning note</span><textarea maxLength={500} rows={4} value={note} onChange={e => setNote(e.target.value)} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo PGD/PSD guidance draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
