import { Stethoscope } from 'lucide-react'

import { DiagnosisDraftDemo } from './DiagnosisDraftDemo'
import { EyeDrawDraftDemo } from './EyeDrawDraftDemo'
import { IOPDraftDemo } from './IOPDraftDemo'
import { VisualAcuityDraftDemo } from './VisualAcuityDraftDemo'
import { OperativeNoteDraftDemo } from './OperativeNoteDraftDemo'
import { PrescriptionDraftDemo } from './PrescriptionDraftDemo'
import { ConsentDraftDemo } from './ConsentDraftDemo'
import { ExaminationToolNavigation } from './ExaminationToolNavigation'
import type { ExaminationTool } from './examinationNavigation'

interface ExaminationDraftWorkspaceProps {
  csrfToken: string
  episodeId: string
  selectedTool?: ExaminationTool
  onToolChange?: ((tool: ExaminationTool) => void) | undefined
}

// Development-only clinical demonstrations stay isolated to one selected tool.
export function ExaminationDraftWorkspace({ csrfToken, episodeId, selectedTool = 'acuity', onToolChange }: ExaminationDraftWorkspaceProps) {
  return (
    <section className="examination-draft-workspace" aria-labelledby={`examination-tools-${episodeId}`}>
      <div className="examination-workspace-heading">
        <div>
          <p>Demo</p>
          <h3 id={`examination-tools-${episodeId}`}>Examination tools</h3>
        </div>
        <Stethoscope size={18} aria-hidden="true" />
      </div>
      <div className="examination-tool-layout">
        <ExaminationToolNavigation selectedTool={selectedTool} onSelect={(tool) => onToolChange?.(tool)} />
        <div className="examination-tool-panel" aria-live="polite">
          {selectedTool === 'acuity' && <VisualAcuityDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
          {selectedTool === 'pressure' && <IOPDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
          {selectedTool === 'selection' && <DiagnosisDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
          {selectedTool === 'drawing' && <EyeDrawDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
          {selectedTool === 'operative-note' && <OperativeNoteDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
          {selectedTool === 'prescription' && <PrescriptionDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
          {selectedTool === 'consent' && <ConsentDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
        </div>
      </div>
    </section>
  )
}
