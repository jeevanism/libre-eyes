import { queryOptions } from '@tanstack/react-query'

import { authAPI } from '../../api/client'

export const authKeys = {
  all: ['auth'] as const,
  loginOptions: () => [...authKeys.all, 'login-options'] as const,
  session: () => [...authKeys.all, 'session'] as const,
}

export const loginOptionsQuery = queryOptions({
  queryKey: authKeys.loginOptions(),
  queryFn: authAPI.loginOptions,
  staleTime: 5 * 60_000,
})

export const sessionQuery = queryOptions({
  queryKey: authKeys.session(),
  queryFn: authAPI.currentSession,
  staleTime: 30_000,
})
