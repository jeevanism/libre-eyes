import { Outlet, createFileRoute, redirect } from '@tanstack/react-router'

import { sessionQuery } from '../features/auth/queries'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ context, location }) => {
    try {
      const session = await context.queryClient.ensureQueryData(sessionQuery)
      return { session }
    } catch {
      // TanStack Router redirects are intentionally thrown control-flow values.
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw redirect({ to: '/login', search: { redirect: location.href } })
    }
  },
  component: () => <Outlet />,
})
