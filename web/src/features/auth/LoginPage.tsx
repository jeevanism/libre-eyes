import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'

import { ApiError, authAPI, type LoginRequest } from '../../api/client'
import { LoginForm } from './LoginForm'
import { authKeys, loginOptionsQuery } from './queries'

interface LoginPageProps {
  redirectTo: string
}

export function LoginPage({ redirectTo }: LoginPageProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const options = useQuery(loginOptionsQuery)
  const login = useMutation({
    mutationFn: (request: LoginRequest) => authAPI.login(request),
    onSuccess: async (session) => {
      queryClient.setQueryData(authKeys.session(), session)
      await navigate({ to: redirectTo })
    },
  })

  if (options.isPending) {
    return <div className="route-status" role="status">Loading</div>
  }
  if (options.isError) {
    return <div className="route-status route-error" role="alert">Unable to load sign-in options.</div>
  }

  const errorMessage = login.error instanceof ApiError && login.error.status === 401
    ? 'The sign-in details could not be verified.'
    : login.isError
      ? 'Sign in is temporarily unavailable.'
      : undefined

  return (
    <LoginForm
      options={options.data}
      pending={login.isPending}
      errorMessage={errorMessage}
      onSubmit={(request) => login.mutate(request)}
    />
  )
}
