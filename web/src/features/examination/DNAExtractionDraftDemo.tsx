import { Dna, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { FormEvent } from 'react'

import { episodesAPI, type DNAExtractionDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function DNAExtractionDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [sampleLabel, setSampleLabel] = useState('Demo sample A')
  const [status, setStatus] = useState<DNAExtractionDemoDraftPayload['status']>('prepared')
  const [storageAddress, setStorageAddress] = useState('Demo box A / slot 01')
  const [extractionDate, setExtractionDate] = useState('')
  const [volume, setVolume] = useState('')
  const [comment, setComment] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: DNAExtractionDemoDraftPayload) => episodesAPI.createDNAExtractionDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_dna_extraction', profileCode: 'demo_dna_extraction_v1', sampleLabel, status, storageAddress, extractionDate, volume, comment }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo DNA-extraction draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`dna-extraction-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`dna-extraction-demo-${episodeId}`}>DNA extraction draft</h3></div><Dna size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores synthetic specimen and storage metadata only. It does not create a genetic result, diagnosis, laboratory submission, or clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo sample label</span><input value={sampleLabel} maxLength={80} onChange={(e) => setSampleLabel(e.target.value)} disabled={save.isPending} /></label>
      <label><span>Demo extraction status</span><select value={status} onChange={(e) => setStatus(e.target.value as DNAExtractionDemoDraftPayload['status'])} disabled={save.isPending}><option value="prepared">Prepared</option><option value="extracted">Extracted</option><option value="stored">Stored</option></select></label>
      <label><span>Demo storage address</span><input value={storageAddress} maxLength={80} onChange={(e) => setStorageAddress(e.target.value)} disabled={save.isPending} /></label>
      <label><span>Extraction date (optional)</span><input type="date" value={extractionDate} onChange={(e) => setExtractionDate(e.target.value)} disabled={save.isPending} /><small>Today or an earlier date only.</small></label>
      <label><span>Demo volume (optional)</span><input inputMode="decimal" value={volume} maxLength={16} onChange={(e) => setVolume(e.target.value)} disabled={save.isPending} /></label>
      <label><span>Demo comment</span><textarea rows={2} maxLength={500} value={comment} onChange={(e) => setComment(e.target.value)} disabled={save.isPending} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo DNA-extraction draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending || !sampleLabel.trim() || !storageAddress.trim()}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
