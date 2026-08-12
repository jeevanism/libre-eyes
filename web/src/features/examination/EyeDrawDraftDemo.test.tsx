import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type * as EyeDrawRuntimeModule from './eyedrawRuntime'

import { EyeDrawDraftDemo } from './EyeDrawDraftDemo'
import { loadEyeDrawRuntime, resetEyeDrawRuntime } from './eyedrawRuntime'

vi.mock('./eyedrawRuntime', async (importOriginal) => {
  const actual = await importOriginal<typeof EyeDrawRuntimeModule>()
  return {
    ...actual,
    loadEyeDrawRuntime: vi.fn().mockResolvedValue(undefined),
    resetEyeDrawRuntime: vi.fn(),
  }
})

const addDoodle = vi.fn()
const repaint = vi.fn()
const setParameterForDoodleOfClass = vi.fn()
const drawing = {
  addDoodle,
  deselectDoodles: vi.fn(),
  deleteAllDoodles: vi.fn(),
  loadDoodles: vi.fn(),
  repaint,
  save: vi.fn(() => `[, {"subclass":"AntSeg"}, {"tags":[]}]`),
  setParameterForDoodleOfClass,
}

function renderDemo() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}><EyeDrawDraftDemo csrfToken="synthetic-csrf-token" episodeId="22222222-2222-4222-8222-222222222222" /></QueryClientProvider>)
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
  delete window.ED
})

describe('EyeDrawDraftDemo', () => {
  it('saves only the filtered development-only drawing payload', async () => {
    let requestBody = ''
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      requestBody = typeof init?.body === 'string' ? init.body : ''
      return Promise.resolve(new Response(JSON.stringify({ id: '33333333-3333-4333-8333-333333333333', episodeId: '22222222-2222-4222-8222-222222222222', eventTypeCode: 'ophthalmology.eyedraw_anterior_segment_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload: {}, version: 1, expiresAt: '2026-09-01T10:00:00Z', newerCommittedEdits: false }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    }))
    window.ED = {
      eye: { Right: 0, Left: 1 },
      init: (_properties, done) => done({ drawing }),
    }
    renderDemo()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Add anterior segment' })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: 'Add anterior segment' }))
    fireEvent.change(screen.getByLabelText('Anterior segment pupil size'), { target: { value: 'Small' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save demo draft' }))
    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Demo drawing draft saved'))
    expect(addDoodle).toHaveBeenCalledWith('AntSeg')
    expect(setParameterForDoodleOfClass).toHaveBeenCalledWith('AntSeg', 'pupilSize', 'Small')
    expect(JSON.parse(requestBody)).toEqual({
      eventTypeCode: 'ophthalmology.eyedraw_anterior_segment_demo', intent: 'create', mode: 'manual', schemaVersion: 1,
      payload: { recordMode: 'development_eyedraw_anterior_segment', canvasCode: 'development_exam_ant_seg_v1', laterality: 'right', drawing: [{ className: 'AntSeg', subclass: 'AntSeg' }] },
    })
    expect(requestBody).not.toContain('patientId')
    expect(requestBody).not.toContain('tags')
  })

  it('cleans up the runtime and creates one replacement canvas on remount', async () => {
    const init = vi.fn((_properties: Record<string, unknown>, done: (value: { drawing: typeof drawing }) => void) => done({ drawing }))
    window.ED = { eye: { Right: 0, Left: 1 }, init }
    const first = renderDemo()
    await waitFor(() => expect(init).toHaveBeenCalledTimes(1))
    first.unmount()
    expect(resetEyeDrawRuntime).toHaveBeenCalled()
    renderDemo()
    await waitFor(() => expect(init).toHaveBeenCalledTimes(2))
    expect(loadEyeDrawRuntime).toHaveBeenCalled()
  })

  it('replaces the canvas DOM node before reinitializing for the other eye', async () => {
    const init = vi.fn((properties: Record<string, unknown>, done: (value: { drawing: typeof drawing }) => void) => done({ drawing }))
    window.ED = { eye: { Right: 0, Left: 1 }, init }
    renderDemo()
    await waitFor(() => expect(init).toHaveBeenCalledTimes(1))
    const firstProperties = init.mock.calls[0]?.[0]
    if (!firstProperties) throw new Error('first EyeDraw initialization was not recorded')
    const firstCanvasID = firstProperties.canvasId

    fireEvent.change(screen.getByLabelText('EyeDraw laterality'), { target: { value: 'left' } })

    await waitFor(() => expect(init).toHaveBeenCalledTimes(2))
    const secondProperties = init.mock.calls[1]?.[0]
    if (!secondProperties) throw new Error('second EyeDraw initialization was not recorded')
    expect(secondProperties).toMatchObject({ eye: 1 })
    expect(secondProperties.canvasId).not.toBe(firstCanvasID)
    expect(resetEyeDrawRuntime).toHaveBeenCalled()
  })
})
