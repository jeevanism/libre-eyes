import { describe, expect, it } from 'vitest'

import { evaluateBrandingContrast } from './contrast'

describe('branding contrast', () => {
  it('reports the measured ratio and threshold for inaccessible action colours', () => {
    const primary = evaluateBrandingContrast('primary', '#e4e651')
    const hover = evaluateBrandingContrast('primaryHover', '#f03891')

    expect(primary.ratio).toBeCloseTo(1.33, 2)
    expect(primary.minimum).toBe(4.5)
    expect(primary.valid).toBe(false)
    expect(hover.ratio).toBeCloseTo(3.70, 2)
    expect(hover.valid).toBe(false)
  })

  it('uses the correct comparison surface for every semantic token', () => {
    expect(evaluateBrandingContrast('primary', '#116466').valid).toBe(true)
    expect(evaluateBrandingContrast('primaryHover', '#0c5355').valid).toBe(true)
    expect(evaluateBrandingContrast('selectedSurface', '#deefee').valid).toBe(true)
    expect(evaluateBrandingContrast('focus', '#0b6fcc').valid).toBe(true)
  })
})
