import type { components, paths } from './schema'
import type {
	components as patientSearchComponents,
	paths as patientSearchPaths,
} from './patient-search-schema'
import type {
  components as episodeComponents,
  paths as episodePaths,
} from './episodes-schema'

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
}
