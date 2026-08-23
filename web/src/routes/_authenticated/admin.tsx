import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
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
  const [editing, setEditing] = useState<string | null>(null)
  const [username, setUsername] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('clinical_user')
  const [siteId, setSiteId] = useState('')
  const [firmId, setFirmId] = useState('')
  const [userMessage, setUserMessage] = useState<string | null>(null)
  const [userError, setUserError] = useState<string | null>(null)
  const userMutation = useMutation({ mutationFn: ({ id, version, active }: { id: string; version: number; active: boolean }) => adminAPI.setUserActive(id, version, active, session?.csrfToken ?? ''), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['admin'] }) } })
  const settingMutation = useMutation({ mutationFn: ({ key, value, version }: { key: string; value: string; version: number }) => adminAPI.updateSetting(key, value, version, session?.csrfToken ?? ''), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['admin'] }) } })
  const userSave = useMutation({ mutationFn: () => editing ? adminAPI.updateUser(editing, { username, displayName, password: password || undefined, role, siteIds: siteId ? [Number(siteId)] : [], firmIds: firmId ? [Number(firmId)] : [], expectedVersion: users.find(user => user.id === editing)?.version ?? 0 }, session?.csrfToken ?? '') : adminAPI.createUser({ username, displayName, password, role, siteIds: siteId ? [Number(siteId)] : [], firmIds: firmId ? [Number(firmId)] : [] }, session?.csrfToken ?? ''), onMutate: () => { setUserMessage(null); setUserError(null) }, onSuccess: () => { setEditing(null); setUsername(''); setDisplayName(''); setPassword(''); setRole('clinical_user'); setSiteId(''); setFirmId(''); setUserMessage(editing ? 'Synthetic user updated successfully.' : 'Synthetic user created successfully.'); void queryClient.invalidateQueries({ queryKey: ['admin'] }) }, onError: (error) => { setUserError(error instanceof Error ? error.message : 'The user could not be saved.') } })
  if (!session?.permissions.includes('admin.development.read') || usersError || contextsError || settingsError || auditError) {
    return <div className="admin-workspace"><div className="workspace-title"><p>Administration demonstration</p><h1>Access restricted</h1><span>Your current role cannot view administration settings.</span></div></div>
  }

  return <div className="admin-workspace">
    <div className="workspace-title"><p>Administration demonstration</p><h1>Admin &amp; configuration</h1><span>Seeded synthetic settings and users only. No production records are changed.</span></div>
    <div className="admin-grid">
      <section className="admin-panel"><h2><Users size={18} /> Users</h2>
        <form className="admin-user-form" onSubmit={event => { event.preventDefault(); userSave.mutate() }}>
          <strong>{editing ? 'Edit synthetic user' : 'Create synthetic user'}</strong>
          <label>Username<input required pattern="[a-z0-9._-]+" value={username} onChange={event => setUsername(event.target.value)} /></label>
          <label>Display name<input required value={displayName} onChange={event => setDisplayName(event.target.value)} /></label>
          <label>Password{editing ? ' (optional)' : ''}<input required={!editing} minLength={6} type="password" value={password} onChange={event => setPassword(event.target.value)} /></label>
          <label>Role<select value={role} onChange={event => setRole(event.target.value)}><option value="clinical_user">Clinical user</option><option value="institution_administrator">Institution administrator</option></select></label>
          <label>Site<select value={siteId} onChange={event => setSiteId(event.target.value)}><option value="">No site membership</option>{contexts?.sites.map(site => <option key={site.id} value={site.id}>{site.name}</option>)}</select></label>
          <label>Firm<select value={firmId} onChange={event => setFirmId(event.target.value)}><option value="">No firm membership</option>{contexts?.firms.map(firm => <option key={firm.id} value={firm.id}>{firm.name}</option>)}</select></label>
          <span><button className="small-action" type="submit" disabled={userSave.isPending}>{userSave.isPending ? 'Saving…' : editing ? 'Update user' : 'Create user'}</button>{editing && <button className="small-action" type="button" onClick={() => setEditing(null)}>Cancel</button>}</span>
          {userError && <p className="inline-error" role="alert">{userError}</p>}
          {userMessage && <p className="inline-success" role="status">{userMessage}</p>}
        </form>
        {users.map(user => <div className="admin-row" key={user.id}><strong>{user.displayName}<small>{user.username}</small></strong><span>{user.role.replaceAll('_', ' ')}<small>{(user.permissions ?? []).join(', ') || 'No permissions'}</small></span><span>{user.active ? 'Active' : 'Inactive'} <button className="small-action" type="button" onClick={() => userMutation.mutate({ id: user.id, version: user.version, active: !user.active })}>{user.active ? 'Deactivate' : 'Reactivate'}</button> <button className="small-action" type="button" onClick={() => { setEditing(user.id); setUsername(user.username); setDisplayName(user.displayName); setRole(user.role); setPassword(''); setSiteId(String(user.sites?.[0]?.id ?? '')); setFirmId(String(user.firms?.[0]?.id ?? '')) }}>Edit</button></span></div>)}
      </section>
      <section className="admin-panel"><h2><Building2 size={18} /> Organization</h2><p><strong>{contexts?.institution.name}</strong></p><p>{contexts?.sites.length ?? 0} seeded sites · {contexts?.firms.length ?? 0} seeded firms</p><p className="muted">Current context: {session?.context.site.name} / {session?.context.firm.name}</p></section>
      <section className="admin-panel"><h2><Settings size={18} /> Demo settings</h2>{settings.map(setting => <div className="admin-row" key={setting.key}><span>{setting.key.replaceAll('_', ' ')}</span><span><input aria-label={setting.key} defaultValue={setting.value} onBlur={event => { const value = event.currentTarget.value; if (value !== setting.value) settingMutation.mutate({ key: setting.key, value, version: setting.version }) }} /></span></div>)}</section>
      <section className="admin-panel"><h2><History size={18} /> Audit summary</h2>{audit.length === 0 ? <p className="muted">No administration events yet.</p> : audit.map((event, index) => <div className="admin-row" key={`${event.command}-${index}`}><span>{event.command}</span><span>{event.targetType}</span><span>{event.changedFields.join(', ') || 'No fields recorded'}</span><span>{event.outcome}</span></div>)}</section>
    </div>
  </div>
}
