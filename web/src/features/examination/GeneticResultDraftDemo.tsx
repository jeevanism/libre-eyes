import { FlaskConical, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { episodesAPI, type GeneticResultDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function GeneticResultDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [testType, setTestType] = useState<GeneticResultDemoDraftPayload['testType']>('demo_panel')
  const [status, setStatus] = useState<GeneticResultDemoDraftPayload['status']>('demo_available')
  const [sourceLabel, setSourceLabel] = useState('')
  const [resultDate, setResultDate] = useState(new Date().toISOString().slice(0, 10))
  const [summary, setSummary] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: GeneticResultDemoDraftPayload) => episodesAPI.createGeneticResultDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_genetic_result', profileCode: 'demo_genetic_result_v1', testType, status, sourceLabel: sourceLabel.trim(), resultDate, summary: summary.trim() }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo genetic-results draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`genetic-result-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`genetic-result-demo-${episodeId}`}>Genetic results draft</h3></div><FlaskConical size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">Synthetic metadata only. It does not interpret variants, diagnose, enroll research, transfer externally, or create a clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo test type</span><select value={testType} onChange={e => setTestType(e.target.value as GeneticResultDemoDraftPayload['testType'])}><option value="demo_panel">Demo panel</option><option value="demo_single_gene">Demo single gene</option><option value="demo_carrier_screen">Demo carrier screen</option></select></label>
      <label><span>Demo status</span><select value={status} onChange={e => setStatus(e.target.value as GeneticResultDemoDraftPayload['status'])}><option value="demo_pending">Demo pending</option><option value="demo_available">Demo available</option><option value="demo_withdrawn">Demo withdrawn</option></select></label>
      <label><span>Demo source label</span><input required maxLength={120} value={sourceLabel} onChange={e => setSourceLabel(e.target.value)} /></label>
      <label><span>Result date</span><input type="date" value={resultDate} onChange={e => setResultDate(e.target.value)} /><small>Today or an earlier date only.</small></label>
      <label><span>Plain synthetic summary</span><textarea maxLength={500} rows={4} value={summary} onChange={e => setSummary(e.target.value)} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo genetic-results draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
