import { useMutation } from '@tanstack/react-query'
import { CircleDot, Eraser, FlaskConical, Save } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { episodesAPI, type EventDraft, type EyeDrawDemoDraftPayload } from '../../api/client'
import { describeDraftSaveError } from './draftError'
import {
  eyeDrawPayloadObjects,
  eyeDrawSerializedDrawing,
  loadEyeDrawRuntime,
  resetEyeDrawRuntime,
  type AllowedEyeDrawDoodleClass,
  type EyeDrawDrawing,
} from './eyedrawRuntime'

interface EyeDrawDraftDemoProps {
  csrfToken: string
  episodeId: string
}

const recordMode = 'development_eyedraw_anterior_segment' as const
const canvasCode = 'development_exam_ant_seg_v1' as const

export function EyeDrawDraftDemo({ csrfToken, episodeId }: EyeDrawDraftDemoProps) {
  const reactId = useId().replace(/[:]/g, '')
  const [canvasGeneration, setCanvasGeneration] = useState(0)
  const canvasId = `visionopus-eyedraw-${reactId}-${canvasGeneration}`
  const inputId = `${canvasId}-data`
  const drawingRef = useRef<EyeDrawDrawing | null>(null)
  const [laterality, setLaterality] = useState<EyeDrawDemoDraftPayload['laterality']>('right')
  const [pupilSize, setPupilSize] = useState('Large')
  const [runtimeState, setRuntimeState] = useState<'loading' | 'ready' | 'error'>('loading')
  const [clientError, setClientError] = useState('')
  const [savedDraft, setSavedDraft] = useState<EventDraft | null>(null)
  const alertRef = useRef<HTMLParagraphElement>(null)

  useEffect(() => {
    let active = true
    loadEyeDrawRuntime().then(() => {
      if (!active || drawingRef.current || !window.ED?.init) return
      resetEyeDrawRuntime()
      window.ED.init({
        drawingName: canvasId,
        canvasId,
        inputId,
        eye: laterality === 'right' ? window.ED.eye.Right : window.ED.eye.Left,
        idSuffix: canvasId,
        isEditable: true,
        graphicsPath: `${'/vendor/eyedraw'}/img`,
        scale: 1,
        toggleScale: 0,
        offsetX: 0,
        offsetY: 0,
        toImage: false,
        onReadyCommandArray: [],
        onDoodlesLoadedCommandArray: [],
        bindingArray: {},
        deleteValueArray: {},
        listenerArray: [],
        showDoodlePopupForDoodles: [],
      }, (controller) => {
        if (!active) return
        drawingRef.current = controller.drawing
        setRuntimeState('ready')
      })
    }).catch(() => {
      if (active) setRuntimeState('error')
    })
    return () => {
      active = false
      drawingRef.current = null
      resetEyeDrawRuntime()
    }
  }, [canvasId, inputId, laterality])

  const saveDraft = useMutation({
    mutationFn: async (payload: EyeDrawDemoDraftPayload) => {
      if (savedDraft) return episodesAPI.updateEyeDrawDemoDraft(savedDraft.id, csrfToken, savedDraft.version, payload)
      return episodesAPI.createEyeDrawDemoDraft(episodeId, csrfToken, payload)
    },
    onSuccess: setSavedDraft,
  })
  const reloadDraft = useMutation({
    mutationFn: () => episodesAPI.getDraft(savedDraft!.id, csrfToken),
    onSuccess: (draft) => {
      const payload = draft.payload as unknown as EyeDrawDemoDraftPayload
      if (!payload || !Array.isArray(payload.drawing)) {
        setClientError('Saved EyeDraw draft could not be recovered. No clinical record was created.')
        return
      }
      const input = document.getElementById(inputId) as HTMLInputElement | null
      if (!input || !drawingRef.current) {
        setClientError('EyeDraw is not ready. Reload the canvas before recovering this draft.')
        return
      }
      const serializedDrawing = eyeDrawSerializedDrawing(payload.drawing)
      try {
        JSON.parse(serializedDrawing)
        drawingRef.current.deleteAllDoodles(true)
        drawingRef.current.deselectDoodles()
        input.value = serializedDrawing
        drawingRef.current.loadDoodles(inputId)
        drawingRef.current.repaint()
      } catch {
        drawingRef.current.repaint()
        setClientError('Saved EyeDraw draft could not be recovered. No clinical record was created.')
        return
      }
      setSavedDraft(draft)
    },
  })

  const failure = saveDraft.isError || reloadDraft.isError
    ? (saveDraft.isError ? describeDraftSaveError(saveDraft.error, 'EyeDraw demo draft') : 'The saved EyeDraw demo draft could not be recovered. Reload the page and try again. No clinical record was created.')
    : ''
  const error = clientError || failure

  useEffect(() => {
    if (error) alertRef.current?.focus()
  }, [error])

  function clearFeedback() {
    setClientError('')
    saveDraft.reset()
    reloadDraft.reset()
  }

  function addDoodle(className: AllowedEyeDrawDoodleClass) {
    clearFeedback()
    drawingRef.current?.addDoodle(className)
    drawingRef.current?.repaint()
  }

  function changePupilSize(value: string) {
    clearFeedback()
    setPupilSize(value)
    drawingRef.current?.setParameterForDoodleOfClass('AntSeg', 'pupilSize', value)
    drawingRef.current?.repaint()
  }

  function clearDrawing() {
    clearFeedback()
    drawingRef.current?.deleteAllDoodles(true)
    drawingRef.current?.repaint()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!drawingRef.current) {
      setClientError('Wait for the EyeDraw canvas before saving the demonstration draft.')
      return
    }
    try {
      const drawing = eyeDrawPayloadObjects(drawingRef.current.save())
      if (drawing.length === 0) {
        setClientError('Add an approved anterior-segment control before saving the demonstration draft.')
        return
      }
      clearFeedback()
      saveDraft.mutate({ recordMode, canvasCode, laterality, drawing })
    } catch {
      setClientError('The EyeDraw canvas returned an invalid development drawing. No clinical record was created.')
    }
  }

  return (
    <section className="eyedraw-draft-demo" aria-labelledby={`eyedraw-demo-${episodeId}`}>
      <div className="examination-draft-demo-heading">
        <div><p>Demo</p><h3 id={`eyedraw-demo-${episodeId}`}>Anterior segment drawing draft</h3></div>
        <FlaskConical size={18} aria-hidden="true" />
      </div>
      <p className="examination-draft-demo-note">This stores a bounded, clinician-owned drawing draft only. It does not create a clinical observation, diagnosis, report, or image.</p>
      <p className="summary-muted">EyeDraw is provided under AGPL-3.0-only: <a href="/vendor/eyedraw/LICENSE" rel="noreferrer" target="_blank">licence</a>, <a href="/vendor/eyedraw/UPSTREAM-NOTICE.md" rel="noreferrer" target="_blank">upstream notice</a>, and <a href="/vendor/eyedraw/ATTRIBUTION.md" rel="noreferrer" target="_blank">attribution</a>.</p>
      <form className="eyedraw-draft-demo-form" onSubmit={submit}>
        <label className="eyedraw-laterality"><span>Eye</span><select aria-label="EyeDraw laterality" disabled={runtimeState !== 'ready' || saveDraft.isPending} onChange={(event) => { clearFeedback(); setRuntimeState('loading'); setLaterality(event.target.value as EyeDrawDemoDraftPayload['laterality']); setCanvasGeneration((generation) => generation + 1) }} value={laterality}><option value="right">Right</option><option value="left">Left</option></select></label>
        <div className="eyedraw-tools" aria-label="Approved EyeDraw controls">
          <button className="secondary-button" disabled={runtimeState !== 'ready'} onClick={() => addDoodle('AntSeg')} type="button"><CircleDot size={16} aria-hidden="true" />Add anterior segment</button>
          <label><span>Pupil size</span><select aria-label="Anterior segment pupil size" disabled={runtimeState !== 'ready'} onChange={(event) => changePupilSize(event.target.value)} value={pupilSize}><option>Large</option><option>Medium</option><option>Small</option></select></label>
          <button className="secondary-button" disabled={runtimeState !== 'ready'} onClick={() => addDoodle('PCIOL')} type="button">Add PCIOL</button>
          <button className="secondary-button" disabled={runtimeState !== 'ready'} onClick={() => addDoodle('PhakoIncision')} type="button">Add phako incision</button>
          <button className="secondary-button" disabled={runtimeState !== 'ready'} onClick={() => addDoodle('SidePort')} type="button">Add side port</button>
          <button className="secondary-button" disabled={runtimeState !== 'ready'} onClick={clearDrawing} type="button"><Eraser size={16} aria-hidden="true" />Clear drawing</button>
        </div>
        <div className="eyedraw-canvas-shell ed2-widget" key={canvasId}>
          <canvas aria-label="Anterior segment EyeDraw canvas" className="ed-canvas" height="500" id={canvasId} tabIndex={0} width="500" />
          <input defaultValue="[]" id={inputId} name={inputId} type="hidden" readOnly />
          {runtimeState === 'loading' && <p className="summary-muted" role="status">Loading EyeDraw canvas…</p>}
          {runtimeState === 'error' && <p className="inline-error" role="alert">EyeDraw runtime assets are unavailable.</p>}
        </div>
        <div className="examination-draft-demo-actions">
          {error && <p className="inline-error" ref={alertRef} role="alert" tabIndex={-1}>{error}</p>}
          {saveDraft.isSuccess && <p className="inline-success" role="status">Demo drawing draft saved. It remains uncommitted.</p>}
          <div className="eyedraw-actions">
            {savedDraft && <button className="secondary-button" disabled={reloadDraft.isPending || saveDraft.isPending} type="button" onClick={() => { clearFeedback(); reloadDraft.mutate() }}>{reloadDraft.isPending ? 'Reloading draft…' : 'Reload saved draft'}</button>}
            <button className="primary-button" disabled={runtimeState !== 'ready' || saveDraft.isPending} type="submit"><Save size={16} aria-hidden="true" />{saveDraft.isPending ? 'Saving draft…' : savedDraft ? 'Update demo draft' : 'Save demo draft'}</button>
          </div>
        </div>
      </form>
    </section>
  )
}
