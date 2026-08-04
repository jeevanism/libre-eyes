import { useMutation, useQueryClient } from '@tanstack/react-query'
import { LogOut } from 'lucide-react'

import { authAPI, type Session } from '../../api/client'
import { authKeys } from './queries'

interface SessionShellProps {
  session: Session
}

export function SessionShell({ session }: SessionShellProps) {
  const queryClient = useQueryClient()
  const logout = useMutation({
    mutationFn: () => authAPI.logout(session.csrfToken),
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: authKeys.all })
      window.location.assign('/login')
    },
  })

  return (
    <div className="app-shell">
      <header className="app-header">
        <a className="app-brand" href="/" aria-label="VisionOpus home">VisionOpus</a>
        <div className="header-context" aria-label="Current clinical context">
          <strong>{session.context.institution.name}</strong>
          <span>{session.context.site.name}</span>
          <span>{session.context.firm.name}</span>
        </div>
        <div className="user-menu">
          <span>{session.user.displayName}</span>
          <button
            className="icon-button"
            type="button"
            title="Sign out"
            aria-label="Sign out"
            disabled={logout.isPending}
            onClick={() => logout.mutate()}
          >
            <LogOut size={18} />
          </button>
        </div>
      </header>
      <main className="workspace">
        <div className="workspace-title">
          <p>Clinical workspace</p>
          <h1>Home</h1>
        </div>
        <div className="empty-workspace">
          <span>No patient selected</span>
        </div>
      </main>
    </div>
  )
}
