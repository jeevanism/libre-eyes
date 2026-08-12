export const developmentVisualAcuityUnit = 'development_distance_scale'

function developmentVisualAcuityValueCode(hundredths: number): string {
  return hundredths < 0
    ? `development_value_m${String(-hundredths).padStart(3, '0')}`
    : `development_value_${String(hundredths).padStart(3, '0')}`
}

function formatVisualAcuityHundredths(hundredths: number): string {
  const sign = hundredths < 0 ? '-' : ''
  const absolute = Math.abs(hundredths)
  return `${sign}${Math.floor(absolute / 100)}.${String(absolute % 100).padStart(2, '0')}`
}

export const developmentVisualAcuityValues = Array.from({ length: 91 }, (_, index) => {
  const hundredths = -30 + index * 2
  return [developmentVisualAcuityValueCode(hundredths), formatVisualAcuityHundredths(hundredths)] as const
})

export const developmentVisualAcuityMethods = [
  ['development_unaided', 'Development unaided'],
  ['development_habitual', 'Demo habitual correction'],
  ['development_best_corrected', 'Demo best-corrected'],
  ['development_pinhole', 'Development pinhole'],
] as const
