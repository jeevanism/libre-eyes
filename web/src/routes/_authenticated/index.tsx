import { createFileRoute } from '@tanstack/react-router'

import { HomeDashboard } from '../../features/home/HomeDashboard'

export const Route = createFileRoute('/_authenticated/')({
  component: AuthenticatedHome,
})

function AuthenticatedHome() {
  const { session } = Route.useRouteContext()
  return <HomeDashboard session={session} />
}
