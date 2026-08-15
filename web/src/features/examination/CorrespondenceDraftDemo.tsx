import { Mail, Save } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError, episodesAPI, type CorrespondenceDemoDraftPayload } from '../../api/client'

interface Props { csrfToken: string; episodeId: string }

export function CorrespondenceDraftDemo({ csrfToken, episodeId }: Props) {
  const [templateCode, setTemplateCode] = useState<CorrespondenceDemoDraftPayload['templateCode']>('demo_clinic_update')
  const [recipientRole, setRecipientRole] = useState<CorrespondenceDemoDraftPayload['recipientRole']>('demo_gp')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [footer, setFooter] = useState('')
  const [clinicDate, setClinicDate] = useState('')
  const [error, setError] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: CorrespondenceDemoDraftPayload) => episodesAPI.createCorrespondenceDemoDraft(episodeId, csrfToken, payload) })
  const failure = save.error instanceof ApiError && save.error.status === 409 ? 'This draft could not be saved because the episode changed. Refresh and try again.' : save.isError ? 'The demo correspondence draft could not be saved. No clinical record was created.' : ''
  const message = error || failure
  useEffect(() => { if (message) alertRef.current?.focus() }, [message])
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!subject.trim() || !body.trim()) { setError('Enter a demo subject and body before saving.'); save.reset(); return }
    setError('')
    save.mutate({ recordMode: 'demo_correspondence', templateCode, recipientRole, subject: subject.trim(), body: body.trim(), footer: footer.trim(), clinicDate })
  }
  return <section className="examination-draft-demo" aria-labelledby={`correspondence-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`correspondence-demo-${episodeId}`}>Correspondence draft</h3></div><Mail size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores a synthetic, clinician-owned plain-text draft only. It does not send, print, export, sign, or create a clinical record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo template</span><select value={templateCode} disabled={save.isPending} onChange={e => setTemplateCode(e.target.value as CorrespondenceDemoDraftPayload['templateCode'])}><option value="demo_clinic_update">Demo clinic update</option><option value="demo_referral_summary">Demo referral summary</option><option value="demo_follow_up">Demo follow-up letter</option></select></label>
      <label><span>Demo recipient</span><select value={recipientRole} disabled={save.isPending} onChange={e => setRecipientRole(e.target.value as CorrespondenceDemoDraftPayload['recipientRole'])}><option value="demo_gp">Demo GP</option><option value="demo_optometrist">Demo optometrist</option><option value="demo_consultant">Demo consultant</option></select></label>
      <label><span>Subject</span><input maxLength={200} value={subject} disabled={save.isPending} onChange={e => setSubject(e.target.value)} /></label>
      <label><span>Plain-text body</span><textarea maxLength={5000} rows={7} value={body} disabled={save.isPending} onChange={e => setBody(e.target.value)} /></label>
      <label><span>Demo footer</span><textarea maxLength={1000} rows={3} value={footer} disabled={save.isPending} onChange={e => setFooter(e.target.value)} /></label>
      <label><span>Clinic date (optional)</span><input type="date" value={clinicDate} disabled={save.isPending} onChange={e => setClinicDate(e.target.value)} /></label>
      <div className="examination-draft-demo-actions">{message && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{message}</p>}{save.isSuccess && <p className="inline-success" role="status">Demo correspondence draft saved. It remains uncommitted.</p>}<button className="primary-button" type="submit" disabled={save.isPending}><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo correspondence draft'}</button></div>
    </form>
  </section>
}
