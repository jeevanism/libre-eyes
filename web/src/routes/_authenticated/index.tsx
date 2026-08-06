import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/')({
  component: AuthenticatedHome,
})

function AuthenticatedHome() {
  return (
    <>
      <div className="workspace-title">
        <p>Clinical workspace</p>
        <h1>Home</h1>
      </div>
      <div className="empty-workspace"><span>No patient selected</span></div>
    </>
  )
}
