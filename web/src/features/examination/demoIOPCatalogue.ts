export const developmentIOPProfileCode = 'development_iop_manual_mmhg'

export const developmentIOPValues = Array.from(
  { length: 100 },
  (_, mmhg) => [`development_iop_${String(mmhg).padStart(2, '0')}`, `${mmhg} mmHg`] as const,
)
