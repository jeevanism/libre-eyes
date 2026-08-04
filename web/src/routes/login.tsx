import { createFileRoute } from '@tanstack/react-router'

import { LoginPage } from '../features/auth/LoginPage'

interface LoginSearch {
  redirect: string
}

function validateSearch(search: Record<string, unknown>): LoginSearch {
  const value = typeof search.redirect === 'string' ? search.redirect : '/'
  const redirect = value.startsWith('/') && !value.startsWith('//') ? value : '/'
  return { redirect }
}

export const Route = createFileRoute('/login')({
  validateSearch,
  component: LoginRoute,
})

function LoginRoute() {
  const search = Route.useSearch()
  return <LoginPage redirectTo={search.redirect} />
}
