import { Save, ScanEye } from 'lucide-react'
import { useRef, useState, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { FormEvent } from 'react'
import { episodesAPI, type VisualFieldsDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

const results = [
  ['demo_normal', 'Demo normal field'],
  ['demo_generalised_reduction', 'Demo generalised reduction'],
  ['demo_field_defect', 'Demo field defect'],
] as const

export function VisualFieldsDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [right, setRight] = useState<VisualFieldsDemoDraftPayload['eyes'][number]['resultCode']>('demo_normal')
  const [left, setLeft] = useState<VisualFieldsDemoDraftPayload['eyes'][number]['resultCode']>('demo_normal')
  const [eyeMode, setEyeMode] = useState<'both' | 'right' | 'left'>('both')
  const [comment, setComment] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: VisualFieldsDemoDraftPayload) => episodesAPI.createVisualFieldsDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); const eyes = eyeMode === 'both' ? [{ eye: 'right' as const, resultCode: right }, { eye: 'left' as const, resultCode: left }] : [{ eye: eyeMode, resultCode: eyeMode === 'right' ? right : left }]; save.mutate({ recordMode: 'demo_visual_fields', strategyCode: 'demo_sita_standard', patternCode: 'demo_24_2', eyes, comment }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo visual-fields draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`visual-fields-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`visual-fields-demo-${episodeId}`}>Visual fields draft</h3></div><ScanEye size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">Synthetic 24-2 field summary only. It does not import images, calculate indices, interpret results, or create a clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo strategy</span><input value="Demo SITA Standard" readOnly /></label>
      <label><span>Demo pattern</span><input value="Demo 24-2" readOnly /></label>
      <label><span>Eyes recorded</span><select value={eyeMode} onChange={e => setEyeMode(e.target.value as typeof eyeMode)}><option value="both">Both eyes</option><option value="right">Right eye</option><option value="left">Left eye</option></select></label>
      <label><span>Right-eye result</span><select value={right} onChange={e => setRight(e.target.value as typeof right)}>{results.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
      {eyeMode !== 'right' && <label><span>Left-eye result</span><select value={left} onChange={e => setLeft(e.target.value as typeof left)}>{results.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>}
      <label><span>Demo comment</span><textarea maxLength={2000} rows={3} value={comment} onChange={e => setComment(e.target.value)} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo visual-fields draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
