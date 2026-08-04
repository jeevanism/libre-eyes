import type { components, paths } from './schema'

export type LoginOptions = components['schemas']['LoginOptions']
export type Session = components['schemas']['SessionRepresentation']
export type LoginRequest = components['schemas']['LoginRequest']
export type Problem = components['schemas']['Problem']
export type ReplaceContextRequest = components['schemas']['ReplaceContextRequest']

type LoginResponse = paths['/auth/sessions']['post']['responses']['200']['content']['application/json']

export class ApiError extends Error {
  readonly status: number
  readonly problem: Problem | undefined

  constructor(status: number, problem?: Problem) {
    super(problem?.title ?? `Request failed with status ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.problem = problem
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
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
    throw new ApiError(response.status, body)
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
