import type { components, paths } from './schema'
import type {
	components as patientSearchComponents,
	paths as patientSearchPaths,
} from './patient-search-schema'
import type {
  components as episodeComponents,
  paths as episodePaths,
} from './episodes-schema'
import type {
  components as worklistComponents,
  paths as worklistPaths,
} from './worklist-schema'
import type {
  components as theatreBookingComponents,
  paths as theatreBookingPaths,
} from './theatre-booking-schema'

export type LoginOptions = components['schemas']['LoginOptions']
export type Session = components['schemas']['SessionRepresentation']
export type LoginRequest = components['schemas']['LoginRequest']
export type Problem = components['schemas']['Problem']
export type ReplaceContextRequest = components['schemas']['ReplaceContextRequest']
export type PatientSearchRequest = patientSearchComponents['schemas']['PatientSearchRequest']
export type PatientSearchPage = patientSearchComponents['schemas']['PatientSearchPage']
export type PatientSearchResult = patientSearchComponents['schemas']['PatientSearchResult']
export type DuplicateCandidateRequest = patientSearchComponents['schemas']['DuplicateCandidateRequest']
export type DuplicateCandidateResponse = patientSearchComponents['schemas']['DuplicateCandidateResponse']
export type PatientSearchCriteria = patientSearchComponents['schemas']['PatientSearchCriteria']
export type GenderCode = patientSearchComponents['schemas']['GenderCode']
export type Episode = episodeComponents['schemas']['Episode']
export type EpisodePage = episodeComponents['schemas']['EpisodePage']
export type EventHeader = episodeComponents['schemas']['EventHeader']
export type EventPage = episodeComponents['schemas']['EventPage']
export type EventDraft = episodeComponents['schemas']['EventDraft']
export type DevelopmentFlowTicket = worklistComponents['schemas']['Ticket']
export type DevelopmentTheatreBoard = theatreBookingComponents['schemas']['Board']
export type DevelopmentTheatreBookingRequest = theatreBookingComponents['schemas']['BookingRequest']
export type VisualAcuityDraftPayload = {
  recordMode: 'simple'
  eyes: Array<{
    eye: 'left' | 'right'
    assessment: 'recorded'
    readings: Array<{
      unitCode: string
      valueCode: string
      methodCode: string
    }>
  }>
}
export type IOPDraftPayload = episodeComponents['schemas']['DevelopmentIOPDraftPayload']
export type DiagnosisDemoDraftPayload = {
  recordMode: 'development_synthetic_diagnosis'
  profileCode: 'development_ophthalmology_diagnosis_v1'
  selectionCode: 'development_cataract' | 'development_glaucoma' | 'development_macular_condition'
  laterality: 'left' | 'right' | 'bilateral'
  diagnosisDate: string
}
export type EyeDrawDemoDraftPayload = {
  recordMode: 'development_eyedraw_anterior_segment'
  canvasCode: 'development_exam_ant_seg_v1'
  laterality: 'left' | 'right'
  drawing: Array<Record<string, unknown>>
}
export type OperativeNoteDemoDraftPayload = {
  recordMode: 'development_synthetic_operative_note'
  procedureCode: 'development_cataract_extraction' | 'development_trabeculectomy'
  laterality: 'development_right_eye' | 'development_left_eye' | 'development_bilateral'
  surgeonCode: 'development_surgeon_a' | 'development_surgeon_b'
  anaestheticCode: 'development_local_anaesthetic' | 'development_general_anaesthetic' | 'development_no_anaesthetic'
  deliveryCodes: Array<'development_subtenons' | 'development_topical' | 'development_other'>
  comment: string
}
export type PrescriptionDemoDraftPayload = {
  recordMode: 'development_synthetic_medication_order'
  headerComment: string
  items: Array<{
    medicationCode: string
    dose: string
    doseUnit: string
    routeCode: string
    frequencyCode: string
    durationCode: string
    laterality: 'development_left' | 'development_right' | 'development_bilateral' | 'development_not_applicable'
    startDate: string
    comment: string
    taper: { dose: string; frequencyCode: string; durationCode: string } | null
  }>
}
export type ConsentDemoDraftPayload = {
  recordMode: 'development_synthetic_consent'
  formTypeCode: 'development_form_type_1' | 'development_form_type_2' | 'development_form_type_3' | 'development_form_type_4'
  procedureCode: 'development_cataract_extraction' | 'development_trabeculectomy'
  laterality: 'development_left_eye' | 'development_right_eye' | 'development_both_eyes'
  anaestheticCode: 'development_local_anaesthetic' | 'development_general_anaesthetic' | 'development_no_anaesthetic'
  comment: string
}
export type CorrespondenceDemoDraftPayload = {
  recordMode: 'demo_correspondence'
  templateCode: 'demo_clinic_update' | 'demo_referral_summary' | 'demo_follow_up'
  recipientRole: 'demo_gp' | 'demo_optometrist' | 'demo_consultant'
  subject: string
  body: string
  footer: string
  clinicDate: string
}
export type LabResultDemoDraftPayload = {
  recordMode: 'demo_lab_result'
  isSynthetic: true
  resultTypeCode: 'demo_lab_hba1c' | 'demo_lab_creatinine' | 'demo_lab_status'
  fieldKind: 'numeric' | 'choice'
  value: string
  unit: string
  observedAt: string
  comment: string
}
export type VisualFieldsDemoDraftPayload = {
  recordMode: 'demo_visual_fields'
  strategyCode: 'demo_sita_standard'
  patternCode: 'demo_24_2'
  eyes: Array<{ eye: 'left' | 'right'; resultCode: 'demo_normal' | 'demo_generalised_reduction' | 'demo_field_defect' }>
  comment: string
}
export type DevelopmentReferralAppointment = {
  id: string
  syntheticPatientLabel: string
  recipientRole: 'demo_gp' | 'demo_optometrist' | 'demo_consultant'
  clinicCode: 'demo_general_eye_clinic' | 'demo_glaucoma_clinic' | 'demo_retina_clinic'
  appointmentDate: string
  appointmentTime: string
  priority: 'routine' | 'soon' | 'urgent'
  notes: string
  status: 'requested' | 'scheduled' | 'arrived' | 'completed' | 'abandoned'
  version: number
  retentionKind?: 'autosave' | 'manual'
}
export type PatientSummaryHeader = {
  patientId: string
  givenName: string | null
  familyName: string | null
  dateOfBirth: string
  ageYears: number | null
  gender: string
  deceased: boolean
  dateOfDeath: string | null
  clinicalDisclosure: 'authorized' | 'withheld'
  allergyStatus?: string
  alertStatus?: string
  patientVersion: number
  warningVersion?: number
}
export type PatientWarningDetails = {
  patientId: string
  allergies: { status: string }
  alerts: { status: string }
  items: Array<{ kind: string; code: string | null; label: string; reaction: string | null; comment: string | null }>
  complete: boolean
  warningVersion: number
}

type LoginResponse = paths['/auth/sessions']['post']['responses']['200']['content']['application/json']
type PatientSearchResponse = patientSearchPaths['/patients/searches']['post']['responses']['200']['content']['application/json']
type DuplicateCandidatesResponse = patientSearchPaths['/patients/duplicate-candidates']['post']['responses']['200']['content']['application/json']
type EpisodeListResponse = episodePaths['/patients/{patientId}/episodes']['get']['responses']['200']['content']['application/json']
type EventListResponse = episodePaths['/episodes/{episodeId}/events']['get']['responses']['200']['content']['application/json']
type DevelopmentFlowTicketPage = worklistPaths['/api/v1/development/clinic-flow/tickets']['get']['responses']['200']['content']['application/json']
type DevelopmentTheatreBoardResponse = theatreBookingPaths['/api/v1/development/theatre-booking/board']['get']['responses']['200']['content']['application/json']

export class ApiError extends Error {
  readonly status: number
  readonly problem: Problem | undefined
  readonly retryAfterSeconds: number | undefined

  constructor(status: number, problem?: Problem, retryAfterSeconds?: number) {
    super(problem?.title ?? `Request failed with status ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.problem = problem
    this.retryAfterSeconds = retryAfterSeconds
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    cache: 'no-store',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...init?.headers,
    },
  })
  if (!response.ok) {
    const contentType = response.headers.get('content-type') ?? ''
    const body = contentType.includes('application/problem+json')
      ? ((await response.json()) as Problem)
      : undefined
    const retryAfter = Number.parseInt(response.headers.get('Retry-After') ?? '', 10)
    throw new ApiError(
      response.status,
      body,
      Number.isSafeInteger(retryAfter) && retryAfter > 0 ? retryAfter : undefined,
    )
  }
  if (response.status === 204) {
    return undefined as T
  }
  return (await response.json()) as T
}

export const authAPI = {
  loginOptions: () => request<LoginOptions>('/auth/login-options'),
  login: (body: LoginRequest) =>
    request<LoginResponse>('/auth/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  currentSession: () => request<Session>('/auth/session'),
  logout: (csrfToken: string) =>
    request<void>('/auth/session', {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': csrfToken },
    }),
  replaceContext: (body: ReplaceContextRequest, csrfToken: string, contextVersion: number) =>
    request<Session>('/auth/session/context', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
        'If-Match': `"${contextVersion}"`,
      },
      body: JSON.stringify(body),
    }),
}

export const patientSearchAPI = {
  search: (body: PatientSearchRequest, csrfToken: string) =>
    request<PatientSearchResponse>('/patients/searches', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      body: JSON.stringify(body),
    }),
  findDuplicateCandidates: (body: DuplicateCandidateRequest, csrfToken: string) =>
    request<DuplicateCandidatesResponse>('/patients/duplicate-candidates', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      body: JSON.stringify(body),
    }),
}

export const patientSummaryAPI = {
  header: (patientId: string, csrfToken: string, contextVersion: number) =>
    request<PatientSummaryHeader>(`/patients/${encodeURIComponent(patientId)}/summary-header`, {
      headers: { 'X-CSRF-Token': csrfToken, 'X-Context-Version': String(contextVersion) },
    }),
  warnings: (patientId: string, csrfToken: string, contextVersion: number) =>
    request<PatientWarningDetails>(`/patients/${encodeURIComponent(patientId)}/summary-header/warnings`, {
      headers: { 'X-CSRF-Token': csrfToken, 'X-Context-Version': String(contextVersion) },
    }),
}

export const episodesAPI = {
  list: (patientId: string, csrfToken: string, cursor?: string) => {
    const parameters = new URLSearchParams()
    if (cursor) parameters.set('cursor', cursor)
    const suffix = parameters.size > 0 ? `?${parameters.toString()}` : ''
    return request<EpisodeListResponse>(`/patients/${encodeURIComponent(patientId)}/episodes${suffix}`, {
      headers: { 'X-CSRF-Token': csrfToken },
    })
  },
  listEvents: (episodeId: string, csrfToken: string, cursor?: string) => {
    const parameters = new URLSearchParams()
    if (cursor) parameters.set('cursor', cursor)
    const suffix = parameters.size > 0 ? `?${parameters.toString()}` : ''
    return request<EventListResponse>(`/episodes/${encodeURIComponent(episodeId)}/events${suffix}`, {
      headers: { 'X-CSRF-Token': csrfToken },
    })
  },
  createVisualAcuityDraft: (episodeId: string, csrfToken: string, payload: VisualAcuityDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      body: JSON.stringify({
        eventTypeCode: 'ophthalmology.visual_acuity',
        intent: 'create',
        mode: 'manual',
        schemaVersion: 1,
        payload,
      }),
    }),
  createIOPDraft: (episodeId: string, csrfToken: string, payload: IOPDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      body: JSON.stringify({
        eventTypeCode: 'ophthalmology.intraocular_pressure',
        intent: 'create',
        mode: 'manual',
        schemaVersion: 1,
        payload,
      }),
    }),
  createDiagnosisDemoDraft: (episodeId: string, csrfToken: string, payload: DiagnosisDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({
        eventTypeCode: 'ophthalmology.principal_diagnosis_demo',
        intent: 'create',
        mode: 'manual',
        schemaVersion: 1,
        payload,
      }),
    }),
  createEyeDrawDemoDraft: (episodeId: string, csrfToken: string, payload: EyeDrawDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({
        eventTypeCode: 'ophthalmology.eyedraw_anterior_segment_demo',
        intent: 'create',
        mode: 'manual',
        schemaVersion: 1,
        payload,
      }),
    }),
  getDraft: (draftId: string, csrfToken: string) =>
    request<EventDraft>(`/event-drafts/${encodeURIComponent(draftId)}`, {
      headers: { 'X-CSRF-Token': csrfToken },
    }),
  createOperativeNoteDemoDraft: (episodeId: string, csrfToken: string, payload: OperativeNoteDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({
        eventTypeCode: 'ophthalmology.operative_note_demo',
        intent: 'create', mode: 'manual', schemaVersion: 1, payload,
      }),
    }),
  createPrescriptionDemoDraft: (episodeId: string, csrfToken: string, payload: PrescriptionDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ eventTypeCode: 'ophthalmology.prescription_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }),
    }),
  createConsentDemoDraft: (episodeId: string, csrfToken: string, payload: ConsentDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ eventTypeCode: 'ophthalmology.consent_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }),
    }),
  createCorrespondenceDemoDraft: (episodeId: string, csrfToken: string, payload: CorrespondenceDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ eventTypeCode: 'ophthalmology.correspondence_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }),
    }),
  createLabResultDemoDraft: (episodeId: string, csrfToken: string, payload: LabResultDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ eventTypeCode: 'ophthalmology.lab_result_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }),
    }),
  createVisualFieldsDemoDraft: (episodeId: string, csrfToken: string, payload: VisualFieldsDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ eventTypeCode: 'ophthalmology.visual_fields_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }),
    }),
  updateEyeDrawDemoDraft: (draftId: string, csrfToken: string, expectedVersion: number, payload: EyeDrawDemoDraftPayload) =>
    request<EventDraft>(`/event-drafts/${encodeURIComponent(draftId)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ expectedVersion, schemaVersion: 1, payload }),
    }),
}

export const developmentClinicFlowAPI = {
  list: (csrfToken: string) => request<DevelopmentFlowTicketPage>('/development/clinic-flow/tickets', {
    headers: { 'X-CSRF-Token': csrfToken },
  }),
  command: (ticketId: string, command: 'arrive' | 'claim' | 'release' | 'complete', expectedVersion: number, csrfToken: string) =>
    request<DevelopmentFlowTicket>(`/development/clinic-flow/tickets/${encodeURIComponent(ticketId)}/${command}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ expectedVersion }),
    }),
}

export const developmentTheatreBookingAPI = {
  board: () => request<DevelopmentTheatreBoardResponse>('/development/theatre-booking/board'),
  schedule: (requestId: string, expectedVersion: number, targetSessionId: string, csrfToken: string) =>
    request<DevelopmentTheatreBookingRequest>(`/development/theatre-booking/requests/${encodeURIComponent(requestId)}/schedule`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ expectedVersion, targetSessionId }),
    }),
  reschedule: (requestId: string, expectedVersion: number, targetSessionId: string, csrfToken: string) =>
    request<DevelopmentTheatreBookingRequest>(`/development/theatre-booking/requests/${encodeURIComponent(requestId)}/reschedule`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ expectedVersion, targetSessionId }),
    }),
  cancel: (requestId: string, expectedVersion: number, csrfToken: string) =>
    request<DevelopmentTheatreBookingRequest>(`/development/theatre-booking/requests/${encodeURIComponent(requestId)}/cancel`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ expectedVersion }),
    }),
}

export const developmentReferralAppointmentsAPI = {
  list: (csrfToken: string) => request<{ items: DevelopmentReferralAppointment[] }>('/development/referral-appointments/requests', { headers: { 'X-CSRF-Token': csrfToken } }),
  create: (body: Omit<DevelopmentReferralAppointment, 'id' | 'status' | 'version'> & { syntheticPatientId: string }, csrfToken: string) => request<DevelopmentReferralAppointment>('/development/referral-appointments/requests', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  command: (id: string, command: 'schedule' | 'arrive' | 'complete' | 'abandon', expectedVersion: number, csrfToken: string) => request<DevelopmentReferralAppointment>(`/development/referral-appointments/requests/${encodeURIComponent(id)}/${command}`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ expectedVersion }) }),
}
