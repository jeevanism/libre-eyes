import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { Building2, History, Settings, Users } from 'lucide-react'

import { adminAPI } from '../../api/client'
import { sessionQuery } from '../../features/auth/queries'

export const Route = createFileRoute('/_authenticated/admin')({ component: AdminPage })

function AdminPage() {
  const { data: users = [], isError: usersError } = useQuery({ queryKey: ['admin', 'users'], queryFn: adminAPI.users })
  const { data: contexts, isError: contextsError } = useQuery({ queryKey: ['admin', 'contexts'], queryFn: adminAPI.contexts })
  const { data: settings = [], isError: settingsError } = useQuery({ queryKey: ['admin', 'settings'], queryFn: adminAPI.settings })
  const { data: audit = [], isError: auditError } = useQuery({ queryKey: ['admin', 'audit'], queryFn: adminAPI.audit })
  const { data: session } = useQuery(sessionQuery)
  const queryClient = useQueryClient()
  const userMutation = useMutation({ mutationFn: ({ id, version, active }: { id: string; version: number; active: boolean }) => adminAPI.setUserActive(id, version, active, session?.csrfToken ?? ''), onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin'] }) })
  const settingMutation = useMutation({ mutationFn: ({ key, value, version }: { key: string; value: string; version: number }) => adminAPI.updateSetting(key, value, version, session?.csrfToken ?? ''), onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin'] }) })
  if (!session?.permissions.includes('admin.development.read') || usersError || contextsError || settingsError || auditError) {
    return <div className="admin-workspace"><div className="workspace-title"><p>Administration demonstration</p><h1>Access restricted</h1><span>Your current role cannot view administration settings.</span></div></div>
  }

  return <div className="admin-workspace">
    <div className="workspace-title"><p>Administration demonstration</p><h1>Admin &amp; configuration</h1><span>Seeded synthetic settings and users only. No production records are changed.</span></div>
    <div className="admin-grid">
      <section className="admin-panel"><h2><Users size={18} /> Users</h2>{users.map(user => <div className="admin-row" key={user.id}><strong>{user.displayName}</strong><span>{user.role.replaceAll('_', ' ')}</span><span>{user.active ? 'Active' : 'Inactive'} <button className="small-action" type="button" onClick={() => userMutation.mutate({ id: user.id, version: user.version, active: !user.active })}>{user.active ? 'Deactivate' : 'Reactivate'}</button></span></div>)}</section>
      <section className="admin-panel"><h2><Building2 size={18} /> Organization</h2><p><strong>{contexts?.institution.name}</strong></p><p>{contexts?.sites.length ?? 0} seeded sites · {contexts?.firms.length ?? 0} seeded firms</p><p className="muted">Current context: {session?.context.site.name} / {session?.context.firm.name}</p></section>
      <section className="admin-panel"><h2><Settings size={18} /> Demo settings</h2>{settings.map(setting => <div className="admin-row" key={setting.key}><span>{setting.key.replaceAll('_', ' ')}</span><span><input aria-label={setting.key} defaultValue={setting.value} onBlur={event => { const value = event.currentTarget.value; if (value !== setting.value) settingMutation.mutate({ key: setting.key, value, version: setting.version }) }} /></span></div>)}</section>
      <section className="admin-panel"><h2><History size={18} /> Audit summary</h2>{audit.length === 0 ? <p className="muted">No administration events yet.</p> : audit.map((event, index) => <div className="admin-row" key={`${event.command}-${index}`}><span>{event.command}</span><span>{event.targetType}</span><span>{event.changedFields.join(', ') || 'No fields recorded'}</span><span>{event.outcome}</span></div>)}</section>
    </div>
  </div>
}
