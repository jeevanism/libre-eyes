import { FileText, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { episodesAPI, type DocumentDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function DocumentDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [documentType, setDocumentType] = useState<DocumentDemoDraftPayload['documentType']>('demo_clinic_letter')
  const [title, setTitle] = useState('')
  const [laterality, setLaterality] = useState<DocumentDemoDraftPayload['laterality']>('not_applicable')
  const [documentDate, setDocumentDate] = useState(new Date().toISOString().slice(0, 10))
  const [comment, setComment] = useState('')
  const save = useMutation({ mutationFn: (payload: DocumentDemoDraftPayload) => episodesAPI.createDocumentDemoDraft(episodeId, csrfToken, payload) })
  function submit(event: FormEvent) {
    event.preventDefault(); save.reset()
    save.mutate({ recordMode: 'demo_document', profileCode: 'demo_document_v1', documentType, title: title.trim(), laterality: laterality ?? 'not_applicable', documentDate, comment: comment.trim() })
  }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo document draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`document-demo-${episodeId}`}><div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`document-demo-${episodeId}`}>Document metadata draft</h3></div><FileText size={18} aria-hidden="true" /></div><p className="examination-draft-demo-note">Metadata only. No file is uploaded, stored, printed, exported, signed, delivered, or added to the clinical record.</p><form className="examination-draft-demo-form" onSubmit={submit}><label><span>Demo document type</span><select value={documentType} onChange={e => setDocumentType(e.target.value as DocumentDemoDraftPayload['documentType'])}><option value="demo_clinic_letter">Demo clinic letter</option><option value="demo_scan_summary">Demo scan summary</option><option value="demo_external_document">Demo external document</option></select></label><label><span>Document title</span><input required maxLength={160} value={title} onChange={e => setTitle(e.target.value)} /></label><label><span>Laterality (optional)</span><select value={laterality} onChange={e => setLaterality(e.target.value as DocumentDemoDraftPayload['laterality'])}><option value="not_applicable">Not applicable</option><option value="right">Right eye</option><option value="left">Left eye</option><option value="bilateral">Both eyes</option></select></label><label><span>Document date</span><input type="date" value={documentDate} onChange={e => setDocumentDate(e.target.value)} /><small>Today or an earlier date only.</small></label><label><span>Demo comment (optional)</span><textarea maxLength={1000} rows={3} value={comment} onChange={e => setComment(e.target.value)} /></label>{failure && <p className="inline-error" role="alert">{failure}</p>}{save.isSuccess && <p className="inline-success" role="status">Demo document draft saved. It remains uncommitted.</p>}<button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo document draft'}</button></form></section>
}
