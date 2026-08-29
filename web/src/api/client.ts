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
import type { components as brandingComponents } from './branding-schema'

export type LoginOptions = components['schemas']['LoginOptions']
export type Session = components['schemas']['SessionRepresentation'] & { capabilities?: string[] }
export type LoginRequest = components['schemas']['LoginRequest']
export type BrandingFieldError = brandingComponents['schemas']['BrandingFieldError']
export type Problem = components['schemas']['Problem'] & { fieldErrors?: BrandingFieldError[] }
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
export type DidNotAttendDemoDraftPayload = {
  recordMode: 'demo_did_not_attend'
  profileCode: 'demo_did_not_attend_v1'
  eventDate: string
  source: 'demo_clinic_flow'
  comment: string
}
export type DocumentDemoDraftPayload = {
  recordMode: 'demo_document'
  profileCode: 'demo_document_v1'
  documentType: 'demo_clinic_letter' | 'demo_scan_summary' | 'demo_external_document'
  title: string
  laterality?: 'not_applicable' | 'right' | 'left' | 'bilateral'
  documentDate: string
  comment?: string
}
export type MessagingDemoDraftPayload = {
  recordMode: 'demo_messaging'
  profileCode: 'demo_messaging_v1'
  messageType: 'demo_clinic_update' | 'demo_review_request' | 'demo_task_note'
  primaryRecipient: 'demo_gp' | 'demo_optometrist' | 'demo_consultant' | 'demo_clinic_staff' | 'demo_nurse' | 'demo_orthoptist'
  ccRecipients: Array<'demo_gp' | 'demo_optometrist' | 'demo_consultant' | 'demo_clinic_staff' | 'demo_nurse' | 'demo_orthoptist'>
  subject: string
  body: string
  readState?: 'demo_unread' | 'demo_read'
}
export type IOPPhasingDemoEye = { instrumentCode: 'demo_goldmann' | 'demo_tono_pen' | 'demo_i_care' | 'demo_perkins' | 'demo_other'; dilated: boolean; comment: string; readings: Array<{ value: number; measurementTime: string }> }
export type IOPPhasingDemoDraftPayload = { recordMode: 'demo_iop_phasing'; profileCode: 'demo_iop_phasing_v1'; eyeMode: 'right' | 'left' | 'both'; rightEye: IOPPhasingDemoEye | null; leftEye: IOPPhasingDemoEye | null }
export type CatpromDemoDraftPayload = { recordMode: 'demo_catprom'; profileCode: 'demo_catprom_v1'; laterality: 'not_applicable' | 'right' | 'left' | 'bilateral'; answers: Array<{ questionCode: string; answerCode: string }>; comment: string }
export type DNAExtractionDemoDraftPayload = { recordMode: 'demo_dna_extraction'; profileCode: 'demo_dna_extraction_v1'; sampleLabel: string; status: 'prepared' | 'extracted' | 'stored'; storageAddress: string; extractionDate: string; volume: string; comment: string }
export type DNASampleDemoDraftPayload = { recordMode: 'demo_dna_sample'; profileCode: 'demo_dna_sample_v1'; sampleType: 'demo_blood' | 'demo_saliva' | 'demo_other'; consentedBy: 'demo_clinician' | 'demo_patient' | 'demo_guardian'; sampleDate: string; volume: number; comment: string }
export type CviDemoDraftPayload = { recordMode: 'demo_cvi'; profileCode: 'demo_cvi_v1'; status: 'demo_new' | 'demo_in_review' | 'demo_ready_for_discussion'; preferredFormat: 'demo_large_print' | 'demo_audio' | 'demo_digital'; note: string }
export type TherapyIntentDemoDraftPayload = { recordMode: 'demo_therapy_intent'; profileCode: 'demo_therapy_intent_v1'; treatment: 'demo_anti_vegf' | 'demo_steroid' | 'demo_observation'; laterality: 'right' | 'left' | 'bilateral' | 'not_applicable'; note: string }
export type PGDPSDGuidanceDemoDraftPayload = { recordMode: 'demo_pgd_psd_guidance'; profileCode: 'demo_pgd_psd_guidance_v1'; pathway: 'demo_pgd' | 'demo_psd'; medicationLabel: string; laterality: 'right' | 'left' | 'bilateral' | 'not_applicable'; note: string }
export type GeneticResultDemoDraftPayload = { recordMode: 'demo_genetic_result'; profileCode: 'demo_genetic_result_v1'; testType: 'demo_panel' | 'demo_single_gene' | 'demo_carrier_screen'; status: 'demo_pending' | 'demo_available' | 'demo_withdrawn'; sourceLabel: string; resultDate: string; summary: string }
export type AnaestheticFeedbackDemoDraftPayload = { recordMode: 'demo_anaesthetic_feedback'; profileCode: 'demo_anaesthetic_feedback_v1'; anaesthetic: 'demo_general' | 'demo_local' | 'demo_none'; satisfaction: 'demo_very_satisfied' | 'demo_satisfied' | 'demo_neutral' | 'demo_dissatisfied'; note: string }
export type VisualFieldsDemoDraftPayload = {
  recordMode: 'demo_visual_fields'
  strategyCode: 'demo_sita_standard'
  patternCode: 'demo_24_2'
  eyes: Array<{ eye: 'left' | 'right'; resultCode: 'demo_normal' | 'demo_generalised_reduction' | 'demo_field_defect' }>
  comment: string
}
export type BiometryDemoEye = { axialLength: string; r1: string; r2: string; r1Axis: number; r2Axis: number; acd: string; wtw: string }
export type BiometryDemoDraftPayload = { recordMode: 'demo_biometry'; profileCode: 'demo_biometry_v1'; deviceCode: 'demo_manual' | 'demo_iolmaster'; lensCode: 'demo_none' | 'demo_ma60ac' | 'demo_sn60wf' | 'demo_sa60at' | 'demo_mta3uo'; measurementDate: string; rightEye: BiometryDemoEye | null; leftEye: BiometryDemoEye | null; comment: string }
export type IntravitrealInjectionDemoEye = { drugCode: string; siteCode: string; anaestheticCode: string; injectionNumber: number; plannedDate: string; postCheck: 'demo_not_recorded' | 'demo_clear' | 'demo_review' }
export type IntravitrealInjectionDemoDraftPayload = { recordMode: 'demo_intravitreal_injection'; profileCode: 'demo_intravitreal_injection_v1'; eyeMode: 'right' | 'left' | 'both'; rightEye: IntravitrealInjectionDemoEye | null; leftEye: IntravitrealInjectionDemoEye | null; note: string }
export type LaserDemoDraftPayload = { recordMode: 'demo_laser'; profileCode: 'demo_laser_v1'; eyeMode: 'right' | 'left' | 'both'; rightEye: { procedureCode: string } | null; leftEye: { procedureCode: string } | null; siteCode: string; laserCode: string; operatorCode: string; treatmentDate: string; comment: string }
export type OperationChecklistDemoDraftPayload = { recordMode: 'demo_operation_checklist'; profileCode: 'demo_operation_checklist_v1'; eyeMode: 'right' | 'left' | 'both'; questions: Array<{ code: 'identity_check' | 'procedure_confirmed' | 'escort_discussed'; answer: 'demo_yes' | 'demo_no' | 'demo_not_recorded' }>; note: string }
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
export type AdminRoleAssignment = { id: number; roleId: number; roleName: string; scope: string; institutionId: number; active: boolean }
export type AdminUser = { id: string; username: string; displayName: string; role: string; active: boolean; version: number; permissions: string[]; sites: AdminReference[]; firms: AdminReference[]; roles: AdminRoleAssignment[] }
export type AdminRole = { id: number; name: string; description: string; scope: string; permissions: string[] }
export type AdminReference = { id: number; name: string; active?: boolean; version?: number }
export type AdminCatalogueItem = { id: number; category: 'medication' | 'route' | 'frequency' | 'duration' | 'laterality' | 'procedure' | 'biometry' | 'laser' | 'intravitreal' | 'lab' | 'genetics' | 'dna' | 'consent' | 'examination'; code: string; displayName: string; active: boolean; displayOrder: number; version: number }
export type AdminContexts = { institution: AdminReference; sites: AdminReference[]; firms: AdminReference[] }
export type AdminSetting = { key: string; value: string; version: number; scope: 'system' | 'institution' | 'site' | 'firm'; source: string }
export type AdminCapability = { key: string; displayName: string; description: string; enabled: boolean; version: number }
export type AdminAuditEvent = { actorUserId: number; actorDisplayName: string; command: string; targetType: string; targetPublicId?: string; targetKey?: string; targetDisplayName?: string; changedFields: string[]; before?: Record<string, unknown>; after?: Record<string, unknown>; scope: string; outcome: string; correlationId: string; createdAt: string }
export type AdminIntegration = { key: string; displayName: string; description: string; status: string; readOnly: boolean }
export type BrandingProfile = brandingComponents['schemas']['BrandingProfile']
export type BrandingState = brandingComponents['schemas']['BrandingState']
export type BrandingDraft = brandingComponents['schemas']['BrandingDraft']
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
type EpisodeResponse = episodePaths['/patients/{patientId}/episodes']['post']['responses']['201']['content']['application/json']
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

export const adminAPI = {
  users: () => request<AdminUser[]>('/admin/users'),
  roles: () => request<AdminRole[]>('/admin/roles'),
  assignRole: (userId: string, roleId: number, csrfToken: string) => request<AdminUser>(`/admin/users/${userId}/roles`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ roleId }) }),
  revokeRole: (userId: string, roleId: number, csrfToken: string) => request<AdminUser>(`/admin/users/${userId}/roles/${roleId}/revoke`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: '{}' }),
  contexts: () => request<AdminContexts>('/admin/contexts'),
  settings: () => request<AdminSetting[]>('/admin/settings'),
  audit: (filter?: { command?: string; outcome?: string; targetType?: string; actor?: string }) => { const query = new URLSearchParams(); Object.entries(filter ?? {}).forEach(([key, value]) => { if (value) query.set(key, value) }); return request<AdminAuditEvent[]>(`/admin/audit${query.toString() ? `?${query}` : ''}`) },
  integrations: () => request<AdminIntegration[]>('/admin/integrations'),
  capabilities: () => request<AdminCapability[]>('/admin/capabilities'),
  setCapability: (key: string, enabled: boolean, expectedVersion: number, csrfToken: string) => request<AdminCapability>(`/admin/capabilities/${key}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ enabled, expectedVersion }) }),
  prescriptionCatalogue: () => request<AdminCatalogueItem[]>('/admin/catalogues/prescription'),
  createPrescriptionCatalogue: (body: Omit<AdminCatalogueItem, 'id' | 'version'>, csrfToken: string) => request<AdminCatalogueItem>('/admin/catalogues/prescription', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  updatePrescriptionCatalogue: (id: number, body: Omit<AdminCatalogueItem, 'id' | 'version'> & { expectedVersion: number }, csrfToken: string) => request<AdminCatalogueItem>(`/admin/catalogues/prescription/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  theatreProcedureCatalogue: () => request<AdminCatalogueItem[]>('/admin/catalogues/theatre-procedure'),
  createTheatreProcedureCatalogue: (body: Omit<AdminCatalogueItem, 'id' | 'version'>, csrfToken: string) => request<AdminCatalogueItem>('/admin/catalogues/theatre-procedure', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  updateTheatreProcedureCatalogue: (id: number, body: Omit<AdminCatalogueItem, 'id' | 'version'> & { expectedVersion: number }, csrfToken: string) => request<AdminCatalogueItem>(`/admin/catalogues/theatre-procedure/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  clinicalReferenceCatalogue: () => request<AdminCatalogueItem[]>('/admin/catalogues/clinical'),
  createClinicalReferenceCatalogue: (body: Omit<AdminCatalogueItem, 'id' | 'version'>, csrfToken: string) => request<AdminCatalogueItem>('/admin/catalogues/clinical', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  updateClinicalReferenceCatalogue: (id: number, body: Omit<AdminCatalogueItem, 'id' | 'version'> & { expectedVersion: number }, csrfToken: string) => request<AdminCatalogueItem>(`/admin/catalogues/clinical/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  setUserActive: (id: string, expectedVersion: number, active: boolean, csrfToken: string) => request<AdminUser>(`/admin/users/${id}/${active ? 'reactivate' : 'deactivate'}`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ expectedVersion }) }),
  createUser: (body: { username: string; displayName: string; password: string; role: string; siteIds: number[]; firmIds: number[] }, csrfToken: string) => request<AdminUser>('/admin/users', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  updateUser: (id: string, body: { username: string; displayName: string; password?: string | undefined; role: string; siteIds: number[]; firmIds: number[]; expectedVersion: number }, csrfToken: string) => request<AdminUser>(`/admin/users/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify(body) }),
  updateSetting: (key: string, value: string, expectedVersion: number, csrfToken: string) => request<AdminSetting>('/admin/settings', { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ key, value, expectedVersion }) }),
  createSite: (name: string, csrfToken: string) => request<AdminReference>('/admin/sites', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ name, active: true }) }),
  updateSite: (id: number, name: string, active: boolean, version: number, csrfToken: string) => request<AdminReference>(`/admin/sites/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ name, active, expectedVersion: version }) }),
  setSiteActive: (id: number, name: string, version: number, active: boolean, csrfToken: string) => request<AdminReference>(`/admin/sites/${id}/${active ? 'reactivate' : 'deactivate'}`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ name, expectedVersion: version }) }),
  createFirm: (name: string, csrfToken: string) => request<AdminReference>('/admin/firms', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ name, active: true }) }),
  updateFirm: (id: number, name: string, active: boolean, version: number, csrfToken: string) => request<AdminReference>(`/admin/firms/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ name, active, expectedVersion: version }) }),
  setFirmActive: (id: number, name: string, version: number, active: boolean, csrfToken: string) => request<AdminReference>(`/admin/firms/${id}/${active ? 'reactivate' : 'deactivate'}`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ name, expectedVersion: version }) }),
}

export const brandingAPI = {
  publicProfile: (institutionId?: string) => {
    const query = institutionId ? `?institutionId=${encodeURIComponent(institutionId)}` : ''
    return request<BrandingProfile>(`/presentation/branding${query}`)
  },
  state: () => request<BrandingState>('/admin/branding'),
  saveDraft: (body: BrandingDraft, csrfToken: string) => request<BrandingState>('/admin/branding/draft', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(body),
  }),
  publish: (expectedVersion: number, csrfToken: string) => request<BrandingState>('/admin/branding/publish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ expectedVersion }),
  }),
  rollback: (targetProfileVersion: number, expectedVersion: number, csrfToken: string) => request<BrandingState>('/admin/branding/rollback', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ targetProfileVersion, expectedVersion }),
  }),
}

export const patientSearchAPI = {
  recent: (limit = 25, csrfToken = '') => request<PatientSearchResponse>(`/patients/recent?limit=${limit}`, { headers: { 'X-CSRF-Token': csrfToken } }),
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
  create: (patientId: string, csrfToken: string, status: 'open' | 'active' = 'active') =>
    request<EpisodeResponse>(`/patients/${encodeURIComponent(patientId)}/episodes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ status }),
    }),
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
  createBiometryDemoDraft: (episodeId: string, csrfToken: string, payload: BiometryDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.biometry_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  updateBiometryDemoDraft: (draftId: string, csrfToken: string, expectedVersion: number, payload: BiometryDemoDraftPayload) =>
    request<EventDraft>(`/event-drafts/${encodeURIComponent(draftId)}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ expectedVersion, schemaVersion: 1, payload }) }),
  updateEyeDrawDemoDraft: (draftId: string, csrfToken: string, expectedVersion: number, payload: EyeDrawDemoDraftPayload) =>
    request<EventDraft>(`/event-drafts/${encodeURIComponent(draftId)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ expectedVersion, schemaVersion: 1, payload }),
    }),
  createIntravitrealInjectionDemoDraft: (episodeId: string, csrfToken: string, payload: IntravitrealInjectionDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.intravitreal_injection_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createLaserDemoDraft: (episodeId: string, csrfToken: string, payload: LaserDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.laser_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createOperationChecklistDemoDraft: (episodeId: string, csrfToken: string, payload: OperationChecklistDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.operation_checklist_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  updateOperationChecklistDemoDraft: (draftId: string, csrfToken: string, expectedVersion: number, payload: OperationChecklistDemoDraftPayload) =>
    request<EventDraft>(`/event-drafts/${encodeURIComponent(draftId)}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ expectedVersion, schemaVersion: 1, payload }) }),
  createDidNotAttendDemoDraft: (episodeId: string, csrfToken: string, payload: DidNotAttendDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.did_not_attend_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createDocumentDemoDraft: (episodeId: string, csrfToken: string, payload: DocumentDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.document_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createMessagingDemoDraft: (episodeId: string, csrfToken: string, payload: MessagingDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.messaging_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createIOPPhasingDemoDraft: (episodeId: string, csrfToken: string, payload: IOPPhasingDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.iop_phasing_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createCatpromDemoDraft: (episodeId: string, csrfToken: string, payload: CatpromDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.catprom_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createDNAExtractionDemoDraft: (episodeId: string, csrfToken: string, payload: DNAExtractionDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.dna_extraction_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createDNASampleDemoDraft: (episodeId: string, csrfToken: string, payload: DNASampleDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.dna_sample_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createCviDemoDraft: (episodeId: string, csrfToken: string, payload: CviDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.cvi_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createTherapyIntentDemoDraft: (episodeId: string, csrfToken: string, payload: TherapyIntentDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.therapy_intent_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createPGDPSDGuidanceDemoDraft: (episodeId: string, csrfToken: string, payload: PGDPSDGuidanceDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.pgd_psd_guidance_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createGeneticResultDemoDraft: (episodeId: string, csrfToken: string, payload: GeneticResultDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.genetic_result_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
  createAnaestheticFeedbackDemoDraft: (episodeId: string, csrfToken: string, payload: AnaestheticFeedbackDemoDraftPayload) =>
    request<EventDraft>(`/episodes/${encodeURIComponent(episodeId)}/event-drafts`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }, body: JSON.stringify({ eventTypeCode: 'ophthalmology.anaesthetic_feedback_demo', intent: 'create', mode: 'manual', schemaVersion: 1, payload }) }),
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
