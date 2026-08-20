import { Activity, ClipboardList, Crosshair, Dna, Eye, FileCheck2, FileText, FlaskConical, Gauge, Mail, MessageSquare, ScanEye, Syringe, UserRoundX, Zap } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export type ExaminationTool = 'acuity' | 'pressure' | 'phasing' | 'catprom' | 'dna-extraction' | 'dna-sample' | 'cvi' | 'lab-results' | 'visual-fields' | 'biometry' | 'intravitreal' | 'laser' | 'therapy-intent' | 'checklist' | 'did-not-attend' | 'document' | 'messaging' | 'selection' | 'drawing' | 'operative-note' | 'prescription' | 'consent' | 'correspondence'

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
      { id: 'phasing', label: 'IOP phasing', Icon: Gauge },
      { id: 'catprom', label: 'Cataract questionnaire', Icon: ClipboardList },
      { id: 'dna-extraction', label: 'DNA extraction', Icon: Dna },
      { id: 'dna-sample', label: 'DNA sample', Icon: Dna },
      { id: 'cvi', label: 'CVI information', Icon: ClipboardList },
      { id: 'lab-results', label: 'Lab results', Icon: FlaskConical },
      { id: 'visual-fields', label: 'Visual fields', Icon: ScanEye },
      { id: 'biometry', label: 'Biometry', Icon: ScanEye },
      { id: 'intravitreal', label: 'Intravitreal injection', Icon: Syringe },
      { id: 'laser', label: 'Laser treatment', Icon: Zap },
      { id: 'therapy-intent', label: 'Therapy intent', Icon: Crosshair },
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
      { id: 'messaging', label: 'Internal messaging', Icon: MessageSquare },
    ],
  },
]

export const examinationTools = examinationToolGroups.flatMap((group) => group.items)

export function isExaminationTool(value: unknown): value is ExaminationTool {
  return typeof value === 'string' && examinationTools.some((tool) => tool.id === value)
}
