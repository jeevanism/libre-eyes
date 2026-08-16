import { Save, UserRoundX } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { episodesAPI, type DidNotAttendDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

export function DidNotAttendDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [eventDate, setEventDate] = useState(new Date().toISOString().slice(0, 10))
  const [comment, setComment] = useState('')
  const save = useMutation({ mutationFn: (payload: DidNotAttendDemoDraftPayload) => episodesAPI.createDidNotAttendDemoDraft(episodeId, csrfToken, payload) })
  function submit(event: FormEvent) {
    event.preventDefault()
    save.reset()
    save.mutate({ recordMode: 'demo_did_not_attend', profileCode: 'demo_did_not_attend_v1', eventDate, source: 'demo_clinic_flow', comment: comment.trim() })
  }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo Did Not Attend draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`dna-demo-${episodeId}`}><div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`dna-demo-${episodeId}`}>Did Not Attend draft</h3></div><UserRoundX size={18} aria-hidden="true" /></div><p className="examination-draft-demo-note">Synthetic outcome only. It does not change clinic flow, appointments, referrals, or clinical records.</p><form className="examination-draft-demo-form" onSubmit={submit}><label><span>Demo event date</span><input type="date" value={eventDate} onChange={e => setEventDate(e.target.value)} /></label><label><span>Demo comment</span><textarea maxLength={255} rows={3} value={comment} onChange={e => setComment(e.target.value)} /></label>{failure && <p className="inline-error" role="alert">{failure}</p>}{save.isSuccess && <p className="inline-success" role="status">Demo Did Not Attend draft saved. It remains uncommitted.</p>}<button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button></form></section>
}
