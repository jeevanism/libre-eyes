import { Activity, ClipboardList, Eye, Gauge, ScanEye, Stethoscope } from 'lucide-react'
import { useId, useState } from 'react'

import { DiagnosisDraftDemo } from './DiagnosisDraftDemo'
import { EyeDrawDraftDemo } from './EyeDrawDraftDemo'
import { IOPDraftDemo } from './IOPDraftDemo'
import { VisualAcuityDraftDemo } from './VisualAcuityDraftDemo'
import { OperativeNoteDraftDemo } from './OperativeNoteDraftDemo'
import { PrescriptionDraftDemo } from './PrescriptionDraftDemo'

interface ExaminationDraftWorkspaceProps {
  csrfToken: string
  episodeId: string
}

type ExaminationTool = 'acuity' | 'pressure' | 'selection' | 'drawing' | 'operative-note' | 'prescription'

const tools: Array<{ id: ExaminationTool, label: string, Icon: typeof Eye }> = [
  { id: 'acuity', label: 'Visual acuity', Icon: Eye },
  { id: 'pressure', label: 'Intraocular pressure', Icon: Gauge },
  { id: 'selection', label: 'Ophthalmology selection', Icon: Activity },
  { id: 'drawing', label: 'Anterior segment drawing', Icon: ScanEye },
  { id: 'operative-note', label: 'Operative note', Icon: Activity },
  { id: 'prescription', label: 'Medication order', Icon: ClipboardList },
]

// Development-only clinical demonstrations stay isolated to one selected tool.
export function ExaminationDraftWorkspace({ csrfToken, episodeId }: ExaminationDraftWorkspaceProps) {
  const [selectedTool, setSelectedTool] = useState<ExaminationTool>('acuity')
  const tabID = useId().replace(/[:]/g, '')

  return (
    <section className="examination-draft-workspace" aria-labelledby={`examination-tools-${episodeId}`}>
      <div className="examination-workspace-heading">
        <div>
          <p>Demo</p>
          <h3 id={`examination-tools-${episodeId}`}>Examination tools</h3>
        </div>
        <Stethoscope size={18} aria-hidden="true" />
      </div>
      <div aria-label="Examination tool" className="examination-tool-tabs" role="tablist">
        {tools.map(({ id, label, Icon }) => (
          <button
            aria-controls={`${tabID}-${id}`}
            aria-selected={selectedTool === id}
            id={`${tabID}-${id}-tab`}
            key={id}
            onClick={() => setSelectedTool(id)}
            role="tab"
            type="button"
          >
            <Icon size={16} aria-hidden="true" />
            {label}
          </button>
        ))}
      </div>
      <div aria-labelledby={`${tabID}-${selectedTool}-tab`} className="examination-tool-panel" id={`${tabID}-${selectedTool}`} role="tabpanel">
        {selectedTool === 'acuity' && <VisualAcuityDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
        {selectedTool === 'pressure' && <IOPDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
        {selectedTool === 'selection' && <DiagnosisDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
        {selectedTool === 'drawing' && <EyeDrawDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
        {selectedTool === 'operative-note' && <OperativeNoteDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
        {selectedTool === 'prescription' && <PrescriptionDraftDemo csrfToken={csrfToken} episodeId={episodeId} />}
      </div>
    </section>
  )
}
