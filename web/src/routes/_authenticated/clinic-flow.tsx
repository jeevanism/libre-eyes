import { createFileRoute } from '@tanstack/react-router'

import { DevelopmentClinicFlowBoard } from '../../features/worklist/DevelopmentClinicFlowBoard'

export const Route = createFileRoute('/_authenticated/clinic-flow')({
  component: ClinicFlowRoute,
})

function ClinicFlowRoute() {
  const { session } = Route.useRouteContext()
  return (
    <section className="clinic-flow-page" aria-labelledby="clinic-flow-page-title">
      <div className="workspace-title">
        <p>Demo</p>
        <h1 id="clinic-flow-page-title">Clinic flow</h1>
      </div>
      <DevelopmentClinicFlowBoard
        allowed={session.permissions.includes('worklist.development_flow.manage')}
        contextVersion={session.contextVersion}
        csrfToken={session.csrfToken}
      />
    </section>
  )
}
