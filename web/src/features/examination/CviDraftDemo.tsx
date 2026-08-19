import { ClipboardList, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { FormEvent } from 'react'
import { episodesAPI, type CviDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function CviDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [status, setStatus] = useState<CviDemoDraftPayload['status']>('demo_new')
  const [preferredFormat, setPreferredFormat] = useState<CviDemoDraftPayload['preferredFormat']>('demo_digital')
  const [note, setNote] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: CviDemoDraftPayload) => episodesAPI.createCviDemoDraft(episodeId, csrfToken, payload) })
  useEffect(() => { if (save.isError) alertRef.current?.focus() }, [save.isError])
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_cvi', profileCode: 'demo_cvi_v1', status, preferredFormat, note }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo CVI-information draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`cvi-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`cvi-demo-${episodeId}`}>CVI information draft</h3></div><ClipboardList size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores synthetic information only. It is not a certificate and creates no signature, PDF, delivery, diagnosis, or clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo status</span><select value={status} onChange={(e) => setStatus(e.target.value as CviDemoDraftPayload['status'])} disabled={save.isPending}><option value="demo_new">Demo new</option><option value="demo_in_review">Demo in review</option><option value="demo_ready_for_discussion">Demo ready for discussion</option></select></label>
      <label><span>Preferred format</span><select value={preferredFormat} onChange={(e) => setPreferredFormat(e.target.value as CviDemoDraftPayload['preferredFormat'])} disabled={save.isPending}><option value="demo_digital">Demo digital</option><option value="demo_large_print">Demo large print</option><option value="demo_audio">Demo audio</option></select></label>
      <label><span>Demo note</span><textarea rows={3} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} disabled={save.isPending} /></label>
      {failure && <p className="inline-error" role="alert" tabIndex={-1} ref={alertRef}>{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo CVI-information draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button>
    </form>
  </section>
}
