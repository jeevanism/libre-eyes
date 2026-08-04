import { createFileRoute } from '@tanstack/react-router'

import { SessionShell } from '../../features/auth/SessionShell'

export const Route = createFileRoute('/_authenticated/')({
  component: AuthenticatedHome,
})

function AuthenticatedHome() {
  const { session } = Route.useRouteContext()
  return <SessionShell session={session} />
}
