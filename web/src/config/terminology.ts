/** Customer-facing wording only; internal development_* identifiers are intentionally unchanged. */
const configuredEnvironmentLabel = import.meta.env.VITE_UI_ENVIRONMENT_LABEL?.trim()

export const presentationTerminology = {
  environmentLabel: configuredEnvironmentLabel || 'Demo',
  environmentHeading: configuredEnvironmentLabel ? `${configuredEnvironmentLabel} environment` : 'Demonstration environment',
  draftLabel: configuredEnvironmentLabel ? `${configuredEnvironmentLabel} draft` : 'Demonstration draft',
} as const
