import { ApiError } from '../../api/client'

/** Convert transport failures into safe, actionable messages for demo editors. */
export function describeDraftSaveError(error: unknown, noun = 'demo draft'): string {
  if (!(error instanceof ApiError)) return `The ${noun} could not be saved. Check your connection and try again. No clinical record was created.`
  switch (error.status) {
    case 400:
      return `Some ${noun} values are invalid. Review the fields and try again. No clinical record was created.`
    case 401:
      return 'Your session has expired. Sign in again before saving this draft.'
    case 403:
      return `You do not have permission to save this ${noun}. No clinical record was created.`
    case 404:
      return 'The selected care episode is no longer available. Return to the patient summary and try again.'
    case 409:
      return 'This draft could not be saved because the episode changed. Refresh and try again.'
    case 422:
      return `One or more ${noun} values are outside the allowed demonstration range. Review the fields and try again.`
    case 429:
      return 'Too many save attempts were made. Wait briefly and try again.'
    default:
      return `The ${noun} could not be saved right now. Check your connection and try again. No clinical record was created.`
  }
}
