import EventEmitter2 from 'eventemitter2'
import $ from 'jquery'
import Mustache from 'mustache'

const assetBasePath = '/vendor/eyedraw'
const stylesheetPath = `${assetBasePath}/styles/oe-eyedraw.css`
const scriptPaths = [
  `${assetBasePath}/runtime/eyedraw.js`,
  `${assetBasePath}/runtime/oe-eyedraw.js`,
] as const

export const allowedEyeDrawDoodleClasses = ['AntSeg', 'PCIOL', 'PhakoIncision', 'SidePort'] as const
export type AllowedEyeDrawDoodleClass = (typeof allowedEyeDrawDoodleClasses)[number]

export interface EyeDrawDoodle {
  className: string
}

export interface EyeDrawDrawing {
  addDoodle: (className: AllowedEyeDrawDoodleClass) => EyeDrawDoodle | undefined
  deselectDoodles: () => void
  deleteAllDoodles: (reallyAll?: boolean) => void
  loadDoodles: (inputID: string) => void
  repaint: () => void
  save: () => string
  setParameterForDoodleOfClass: (className: string, parameter: string, value: string) => void
}

export interface EyeDrawController {
  drawing: EyeDrawDrawing
}

interface EyeDrawRuntime {
  Checker?: { reset?: () => void }
  Controller?: unknown
  eye: { Right: number; Left: number }
  init: (properties: Record<string, unknown>, done: (controller: EyeDrawController) => void) => void
}

declare global {
  interface Window {
    $: typeof $
    ED?: EyeDrawRuntime
    EventEmitter2: typeof EventEmitter2
    Mustache: typeof Mustache
    jQuery: typeof $
  }
}

let loadPromise: Promise<void> | undefined

function stylesheet() {
  const selector = `link[data-libreeyes-eyedraw-asset="${stylesheetPath}"]`
  const existing = document.querySelector<HTMLLinkElement>(selector)
  if (existing) return existing
  const link = document.createElement('link')
  link.dataset.libreeyesEyedrawAsset = stylesheetPath
  link.href = stylesheetPath
  link.rel = 'stylesheet'
  document.head.append(link)
  return link
}

function script(source: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const selector = `script[data-libreeyes-eyedraw-asset="${source}"]`
    const existing = document.querySelector<HTMLScriptElement>(selector)
    if (existing?.dataset.loaded === 'true') {
      resolve()
      return
    }
    const element = existing ?? document.createElement('script')
    if (!existing) {
      element.async = false
      element.dataset.libreeyesEyedrawAsset = source
      element.src = source
      document.body.append(element)
    }
    element.addEventListener('load', () => {
      element.dataset.loaded = 'true'
      resolve()
    }, { once: true })
    element.addEventListener('error', () => reject(new Error('EyeDraw runtime assets could not be loaded.')), { once: true })
  })
}

export function loadEyeDrawRuntime(): Promise<void> {
  window.$ = $
  window.jQuery = $
  window.Mustache = Mustache
  window.EventEmitter2 = EventEmitter2
  stylesheet()
  loadPromise ??= scriptPaths.reduce((pending, source) => pending.then(() => script(source)), Promise.resolve())
  return loadPromise
}

export function eyeDrawPayloadObjects(serialized: string): Array<Record<string, unknown>> {
  const parsed: unknown = JSON.parse(serialized.replace(/^\[\s*,/, '['))
  if (!Array.isArray(parsed)) throw new Error('EyeDraw did not return a drawing array.')
  return parsed.flatMap((item): Array<Record<string, unknown>> => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) return []
    const runtimeDoodle = item as Record<string, unknown>
    const className = typeof runtimeDoodle.className === 'string'
      ? runtimeDoodle.className
      : runtimeDoodle.subclass
    if (typeof className !== 'string' || !(allowedEyeDrawDoodleClasses as readonly string[]).includes(className)) return []
    // EyeDraw uses `subclass`; LibreEyes persists the explicit allow-listed
    // `className` alongside it so the server can validate and reload safely.
    return [{ ...runtimeDoodle, className, subclass: className }]
  })
}

export function eyeDrawSerializedDrawing(drawing: Array<Record<string, unknown>>): string {
  return JSON.stringify(drawing.map((doodle) => ({
    ...doodle,
    subclass: typeof doodle.subclass === 'string' ? doodle.subclass : doodle.className,
  })))
}

export function resetEyeDrawRuntime(): void {
  window.ED?.Checker?.reset?.()
}
