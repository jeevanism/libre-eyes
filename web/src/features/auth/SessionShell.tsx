import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { CalendarDays, ClipboardList, LogOut, Search } from 'lucide-react'
import type { ReactNode } from 'react'

import { authAPI, type Session } from '../../api/client'
import { ThemeControl } from '../theme/ThemeControl'
import { authKeys } from './queries'

interface SessionShellProps {
  session: Session
  children: ReactNode
}

export function SessionShell({ session, children }: SessionShellProps) {
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
        <Link className="app-brand" to="/" aria-label="VisionOpus home">VisionOpus</Link>
        <div className="header-context" aria-label="Current clinical context">
          <strong title={session.context.institution.name}>{session.context.institution.name}</strong>
          <span title={session.context.site.name}>{session.context.site.name}</span>
          <span title={session.context.firm.name}>{session.context.firm.name}</span>
        </div>
        <div className="user-menu">
          <span>{session.user.displayName}</span>
          <ThemeControl />
          <button
            className="icon-button"
            type="button"
            title="Sign out"
            aria-label="Sign out"
            disabled={logout.isPending}
            onClick={() => logout.mutate()}
          >
            <LogOut size={18} aria-hidden="true" />
          </button>
        </div>
      </header>
      <nav className="app-nav" aria-label="Primary navigation">
        <Link to="/" activeOptions={{ exact: true }}>Home</Link>
        <Link to="/patients/search"><Search size={16} aria-hidden="true" />Patient search</Link>
        <Link to="/clinic-flow"><ClipboardList size={16} aria-hidden="true" />Clinic flow</Link>
        <Link to="/theatre-booking"><CalendarDays size={16} aria-hidden="true" />Theatre schedule</Link>
      </nav>
      <main className="workspace">{children}</main>
    </div>
  )
}
