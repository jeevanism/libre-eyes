import { MessageSquare, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { episodesAPI, type MessagingDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'

const recipients: Array<MessagingDemoDraftPayload['primaryRecipient']> = ['demo_gp', 'demo_optometrist', 'demo_consultant', 'demo_clinic_staff']
const labels: Record<string, string> = { demo_gp: 'Demo GP', demo_optometrist: 'Demo optometrist', demo_consultant: 'Demo consultant', demo_clinic_staff: 'Demo clinic staff' }

export function MessagingDraftDemo({ csrfToken, episodeId }: { csrfToken: string; episodeId: string }) {
  const [messageType, setMessageType] = useState<MessagingDemoDraftPayload['messageType']>('demo_clinic_update')
  const [primaryRecipient, setPrimaryRecipient] = useState<MessagingDemoDraftPayload['primaryRecipient']>('demo_gp')
  const [ccRecipients, setCCRecipients] = useState<MessagingDemoDraftPayload['ccRecipients']>([])
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const save = useMutation({ mutationFn: (payload: MessagingDemoDraftPayload) => episodesAPI.createMessagingDemoDraft(episodeId, csrfToken, payload) })
  function submit(event: FormEvent) { event.preventDefault(); save.reset(); save.mutate({ recordMode: 'demo_messaging', profileCode: 'demo_messaging_v1', messageType, primaryRecipient, ccRecipients, subject: subject.trim(), body: body.trim(), readState: 'demo_unread' }) }
  function toggleCC(value: MessagingDemoDraftPayload['primaryRecipient']) { setCCRecipients(current => current.includes(value) ? current.filter(item => item !== value) : [...current, value]) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo messaging draft') : ''
  return <section className="examination-draft-demo" aria-labelledby={`messaging-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`messaging-demo-${episodeId}`}>Internal messaging draft</h3></div><MessageSquare size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">Synthetic internal metadata only. Nothing is sent, delivered, notified, printed, exported, or added to a clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo message type</span><select value={messageType} onChange={e => setMessageType(e.target.value as MessagingDemoDraftPayload['messageType'])}><option value="demo_clinic_update">Demo clinic update</option><option value="demo_review_request">Demo review request</option><option value="demo_task_note">Demo task note</option></select></label>
      <label><span>Primary recipient</span><select value={primaryRecipient} onChange={e => { const next = e.target.value as MessagingDemoDraftPayload['primaryRecipient']; setPrimaryRecipient(next); setCCRecipients(current => current.filter(item => item !== next)) }}>{recipients.map(item => <option key={item} value={item}>{labels[item]}</option>)}</select></label>
      <fieldset><legend>Demo copied recipients (up to 5)</legend>{recipients.map(item => <label key={item}><input type="checkbox" checked={ccRecipients.includes(item)} disabled={item === primaryRecipient} onChange={() => toggleCC(item)} />{labels[item]}</label>)}</fieldset>
      <label><span>Subject</span><input required maxLength={160} value={subject} onChange={e => setSubject(e.target.value)} /></label>
      <label><span>Plain-text body</span><textarea required maxLength={4000} rows={5} value={body} onChange={e => setBody(e.target.value)} /></label>
      {failure && <p className="inline-error" role="alert">{failure}</p>}
      {save.isSuccess && <p className="inline-success" role="status">Demo messaging draft saved. It remains uncommitted.</p>}
      <button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo messaging draft'}</button>
    </form>
  </section>
}
