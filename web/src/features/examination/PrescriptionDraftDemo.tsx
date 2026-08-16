import { useMutation } from '@tanstack/react-query'
import { ClipboardList, Save } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { episodesAPI, type PrescriptionDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'
import { demoPrescriptionDurations, demoPrescriptionFrequencies, demoPrescriptionLateralities, demoPrescriptionMedicines, demoPrescriptionRoutes } from './demoPrescriptionCatalogue'

interface Props { csrfToken: string; episodeId: string }
type Item = PrescriptionDemoDraftPayload['items'][number]
const blankItem = (): Item => ({ medicationCode: demoPrescriptionMedicines[0][0], dose: '1', doseUnit: 'drop', routeCode: demoPrescriptionRoutes[0][0], frequencyCode: demoPrescriptionFrequencies[0][0], durationCode: demoPrescriptionDurations[0][0], laterality: demoPrescriptionLateralities[0][0], startDate: new Date().toISOString().slice(0, 10), comment: '', taper: null })

export function PrescriptionDraftDemo({ csrfToken, episodeId }: Props) {
  const [items, setItems] = useState<Item[]>([blankItem()])
  const [headerComment, setHeaderComment] = useState('')
  const [clientError, setClientError] = useState('')
  const alertRef = useRef<HTMLParagraphElement>(null)
  const save = useMutation({ mutationFn: (payload: PrescriptionDemoDraftPayload) => episodesAPI.createPrescriptionDemoDraft(episodeId, csrfToken, payload) })
  const update = (index: number, patch: Partial<Item>) => { setItems(current => current.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item)); setClientError(''); save.reset() }
  function submit(event: FormEvent) { event.preventDefault(); if (items.some(item => !item.dose.trim() || !item.doseUnit.trim())) { setClientError('Complete dose and unit for every medication item before saving.'); return }; setClientError(''); save.mutate({ recordMode: 'development_synthetic_medication_order', headerComment, items }) }
  const failure = save.isError ? describeDraftSaveError(save.error, 'demo medication draft') : ''
  useEffect(() => { if (clientError || failure) alertRef.current?.focus() }, [clientError, failure])
  return <section className="examination-draft-demo" aria-labelledby={`prescription-demo-${episodeId}`}>
    <div className="examination-draft-demo-heading"><div><p>Demo</p><h3 id={`prescription-demo-${episodeId}`}>Medication-order draft</h3></div><ClipboardList size={18} aria-hidden="true" /></div>
    <p className="examination-draft-demo-note">This stores a synthetic, clinician-owned draft only. It does not issue, sign, print, check allergies, or create a clinical medication record.</p>
    <form className="examination-draft-demo-form" onSubmit={submit}>
      <label><span>Demo comments</span><textarea maxLength={2000} rows={2} value={headerComment} onChange={event => { setHeaderComment(event.target.value); save.reset() }} /></label>
      {items.map((item, index) => <MedicationItem key={index} index={index} item={item} disabled={save.isPending} update={update} remove={() => setItems(current => current.filter((_, itemIndex) => itemIndex !== index))} />)}
      <div className="examination-draft-demo-actions"><button className="secondary-button" disabled={items.length >= 3 || save.isPending} type="button" onClick={() => setItems(current => [...current, blankItem()])}>Add medication item</button><button className="primary-button" disabled={save.isPending} type="submit"><Save size={16} aria-hidden="true" />{save.isPending ? 'Saving draft…' : 'Save demo draft'}</button></div>
      {(clientError || failure) && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{clientError || failure}</p>}{save.isSuccess && <p className="inline-success" role="status">Demo medication draft saved. It remains uncommitted.</p>}
    </form>
  </section>
}

function MedicationItem({ index, item, disabled, update, remove }: { index: number; item: Item; disabled: boolean; update: (index: number, patch: Partial<Item>) => void; remove: () => void }) {
  return <fieldset className="examination-draft-eye"><legend>Medication item {index + 1}</legend>
    <label><span>Demo medication</span><select value={item.medicationCode} onChange={event => update(index, { medicationCode: event.target.value })} disabled={disabled}>{demoPrescriptionMedicines.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
    <label><span>Dose</span><input value={item.dose} maxLength={40} onChange={event => update(index, { dose: event.target.value })} disabled={disabled} /></label>
    <label><span>Unit</span><input value={item.doseUnit} maxLength={45} onChange={event => update(index, { doseUnit: event.target.value })} disabled={disabled} /></label>
    <label><span>Route</span><select value={item.routeCode} onChange={event => update(index, { routeCode: event.target.value })} disabled={disabled}>{demoPrescriptionRoutes.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
    <label><span>Frequency</span><select value={item.frequencyCode} onChange={event => update(index, { frequencyCode: event.target.value })} disabled={disabled}>{demoPrescriptionFrequencies.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
    <label><span>Duration</span><select value={item.durationCode} onChange={event => update(index, { durationCode: event.target.value })} disabled={disabled}>{demoPrescriptionDurations.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
    <label><span>Laterality</span><select value={item.laterality} onChange={event => update(index, { laterality: event.target.value as Item['laterality'] })} disabled={disabled}>{demoPrescriptionLateralities.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label>
    <label><span>Start date</span><input type="date" value={item.startDate} onChange={event => update(index, { startDate: event.target.value })} disabled={disabled} /></label>
    <label><span>Item comment</span><input value={item.comment} maxLength={1000} onChange={event => update(index, { comment: event.target.value })} disabled={disabled} /></label>
    <label><span><input type="checkbox" checked={item.taper !== null} disabled={disabled} onChange={event => update(index, { taper: event.target.checked ? { dose: '1', frequencyCode: item.frequencyCode, durationCode: item.durationCode } : null })} /> Include one demo taper row</span></label>
    {item.taper && <fieldset><legend>Demo taper</legend><label><span>Taper dose</span><input value={item.taper.dose} maxLength={40} onChange={event => update(index, { taper: { ...item.taper!, dose: event.target.value } })} disabled={disabled} /></label><label><span>Taper frequency</span><select value={item.taper.frequencyCode} onChange={event => update(index, { taper: { ...item.taper!, frequencyCode: event.target.value } })} disabled={disabled}>{demoPrescriptionFrequencies.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label><label><span>Taper duration</span><select value={item.taper.durationCode} onChange={event => update(index, { taper: { ...item.taper!, durationCode: event.target.value } })} disabled={disabled}>{demoPrescriptionDurations.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></label></fieldset>}
    <button className="secondary-button" disabled={disabled || index === 0} type="button" onClick={remove}>Remove item</button>
  </fieldset>
}
