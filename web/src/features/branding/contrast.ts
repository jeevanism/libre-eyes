import type { BrandingProfile } from '../../api/client'

type BrandingColors = BrandingProfile['colors']

export type BrandingColorField = keyof BrandingColors

export interface BrandingContrastCheck {
  field: BrandingColorField
  label: string
  against: string
  ratio: number
  minimum: number
  valid: boolean
}

const contrastDefinitions: Record<BrandingColorField, {
  label: string
  background: string
  against: string
  minimum: number
}> = {
  primary: { label: 'Primary action', background: '#ffffff', against: 'white text', minimum: 4.5 },
  primaryHover: { label: 'Primary hover', background: '#ffffff', against: 'white text', minimum: 4.5 },
  selectedSurface: { label: 'Selected surface', background: '#172124', against: 'interface text', minimum: 4.5 },
  focus: { label: 'Focus indicator', background: '#ffffff', against: 'white surface', minimum: 3 },
}

export function evaluateBrandingContrast(field: BrandingColorField, color: string): BrandingContrastCheck {
  const definition = contrastDefinitions[field]
  const ratio = contrastRatio(color, definition.background)
  return {
    field,
    label: definition.label,
    against: definition.against,
    ratio,
    minimum: definition.minimum,
    valid: ratio >= definition.minimum,
  }
}

function contrastRatio(first: string, second: string) {
  const left = relativeLuminance(first)
  const right = relativeLuminance(second)
  const lighter = Math.max(left, right)
  const darker = Math.min(left, right)
  return (lighter + 0.05) / (darker + 0.05)
}

function relativeLuminance(color: string) {
  if (!/^#[0-9a-f]{6}$/i.test(color)) return 0
  const channel = (offset: number) => {
    const value = Number.parseInt(color.slice(offset, offset + 2), 16) / 255
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * channel(1) + 0.7152 * channel(3) + 0.0722 * channel(5)
}
