import { Activity, ClipboardList, Eye, FileCheck2, FileText, FlaskConical, Gauge, Mail, ScanEye, Syringe, UserRoundX, Zap } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export type ExaminationTool = 'acuity' | 'pressure' | 'lab-results' | 'visual-fields' | 'biometry' | 'intravitreal' | 'laser' | 'checklist' | 'did-not-attend' | 'document' | 'selection' | 'drawing' | 'operative-note' | 'prescription' | 'consent' | 'correspondence'

export type ExaminationToolItem = {
  id: ExaminationTool
  label: string
  Icon: LucideIcon
}

export type ExaminationToolGroup = {
  id: string
  label: string
  items: ExaminationToolItem[]
}

export const examinationToolGroups: ExaminationToolGroup[] = [
  {
    id: 'measurements',
    label: 'Measurements',
    items: [
      { id: 'acuity', label: 'Visual acuity', Icon: Eye },
      { id: 'pressure', label: 'Intraocular pressure', Icon: Gauge },
      { id: 'lab-results', label: 'Lab results', Icon: FlaskConical },
      { id: 'visual-fields', label: 'Visual fields', Icon: ScanEye },
      { id: 'biometry', label: 'Biometry', Icon: ScanEye },
      { id: 'intravitreal', label: 'Intravitreal injection', Icon: Syringe },
      { id: 'laser', label: 'Laser treatment', Icon: Zap },
    ],
  },
  {
    id: 'findings',
    label: 'Findings',
    items: [
      { id: 'selection', label: 'Ophthalmology selection', Icon: Activity },
      { id: 'drawing', label: 'Anterior segment drawing', Icon: ScanEye },
    ],
  },
  {
    id: 'documentation',
    label: 'Documentation',
    items: [
      { id: 'operative-note', label: 'Operative note', Icon: Activity },
      { id: 'prescription', label: 'Medication order', Icon: ClipboardList },
      { id: 'consent', label: 'Consent form', Icon: FileCheck2 },
      { id: 'correspondence', label: 'Correspondence', Icon: Mail },
      { id: 'checklist', label: 'Operation checklist', Icon: ClipboardList },
      { id: 'did-not-attend', label: 'Did not attend', Icon: UserRoundX },
      { id: 'document', label: 'Document', Icon: FileText },
    ],
  },
]

export const examinationTools = examinationToolGroups.flatMap((group) => group.items)

export function isExaminationTool(value: unknown): value is ExaminationTool {
  return typeof value === 'string' && examinationTools.some((tool) => tool.id === value)
}
