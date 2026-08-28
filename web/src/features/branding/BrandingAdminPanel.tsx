import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Eye, History, Palette, RotateCcw, Save, Send } from 'lucide-react'
import { useState, type FormEvent } from 'react'

import { brandingAPI, type BrandingDraft, type BrandingProfile } from '../../api/client'
import { useBranding } from './BrandingProvider'

interface BrandingAdminPanelProps {
  csrfToken: string
}

const brandingStateKey = ['admin', 'branding'] as const

export function BrandingAdminPanel({ csrfToken }: BrandingAdminPanelProps) {
  const queryClient = useQueryClient()
  const { preview, cancelPreview, isPreviewing } = useBranding()
  const stateQuery = useQuery({ queryKey: brandingStateKey, queryFn: brandingAPI.state })
  const [message, setMessage] = useState<string>()
  const [errorMessage, setErrorMessage] = useState<string>()

  function acceptState(messageText: string) {
    return (state: NonNullable<typeof stateQuery.data>) => {
      queryClient.setQueryData(brandingStateKey, state)
      setMessage(messageText)
      setErrorMessage(undefined)
    }
  }

  function reject(error: Error) {
    setMessage(undefined)
    setErrorMessage(error.message)
  }

  const save = useMutation({
    mutationFn: (draft: BrandingDraft) => brandingAPI.saveDraft(draft, csrfToken),
    onSuccess: acceptState('Branding draft saved. The live presentation has not changed.'),
    onError: reject,
  })
  const publish = useMutation({
    mutationFn: (expectedVersion: number) => brandingAPI.publish(expectedVersion, csrfToken),
    onSuccess: (state) => {
      acceptState('Branding profile published successfully.')(state)
      cancelPreview()
      void queryClient.invalidateQueries({ queryKey: ['branding', 'published'] })
    },
    onError: reject,
  })
  const rollback = useMutation({
    mutationFn: (targetProfileVersion: number) => brandingAPI.rollback(targetProfileVersion, stateQuery.data?.published?.rowVersion ?? 0, csrfToken),
    onSuccess: (state) => {
      acceptState('A previous profile was republished as a new version.')(state)
      cancelPreview()
      void queryClient.invalidateQueries({ queryKey: ['branding', 'published'] })
    },
    onError: reject,
  })

  if (stateQuery.isPending) {
    return <section className="admin-panel branding-admin-panel" aria-busy="true"><h2><Palette size={18} /> Branding</h2><p role="status">Loading presentation profile</p></section>
  }
  if (stateQuery.isError) {
    return <section className="admin-panel branding-admin-panel"><h2><Palette size={18} /> Branding</h2><p className="inline-error" role="alert">The branding profile could not be loaded.</p></section>
  }

  const state = stateQuery.data
  const editableProfile = state.draft ?? state.effective
  const editorKey = `${editableProfile.profileVersion}-${editableProfile.rowVersion}-${state.draft ? 'draft' : 'published'}`

  return (
    <section className="admin-panel branding-admin-panel" aria-labelledby="branding-admin-heading">
      <div className="branding-admin-heading">
        <div>
          <h2 id="branding-admin-heading"><Palette size={18} /> Branding</h2>
          <p className="muted">Institution presentation only. Clinical status colours, layouts, permissions, and recorded data are unchanged.</p>
        </div>
        <span className={`branding-status branding-status-${state.draft ? 'draft' : 'published'}`}>
          {state.draft ? `Draft v${state.draft.profileVersion}` : `Published v${state.effective.profileVersion}`}
        </span>
      </div>

      <BrandingEditor
        key={editorKey}
        profile={editableProfile}
        expectedVersion={state.draft?.rowVersion ?? 0}
        pending={save.isPending}
        onSave={(draft) => save.mutate(draft)}
        onPreview={preview}
      />

      <div className="branding-publish-actions">
        <button className="primary-button" type="button" disabled={!state.draft || publish.isPending} onClick={() => state.draft && publish.mutate(state.draft.rowVersion)}>
          <Send size={16} aria-hidden="true" /> Publish draft
        </button>
        {isPreviewing && <button className="secondary-button" type="button" onClick={cancelPreview}>Cancel preview</button>}
        <span className="muted">Only publication changes the profile seen by other users.</span>
      </div>

      {errorMessage && <p className="inline-error" role="alert">{errorMessage}</p>}
      {message && <p className="inline-success" role="status">{message}</p>}

      <div className="branding-history">
        <h3><History size={16} /> Version history</h3>
        {state.history.map((item) => (
          <div className="branding-history-row" key={item.profileVersion}>
            <span><strong>Version {item.profileVersion}</strong><small>{new Date(item.updatedAt).toLocaleString()}</small></span>
            <span className={`branding-status branding-status-${item.status}`}>{item.status}</span>
            {item.status === 'superseded'
              ? <button className="small-action" type="button" disabled={rollback.isPending} onClick={() => rollback.mutate(item.profileVersion)}><RotateCcw size={14} aria-hidden="true" /> Republish</button>
              : <span />}
          </div>
        ))}
      </div>

      <details className="branding-audit">
        <summary>Recent branding activity</summary>
        {state.audit.length === 0
          ? <p className="muted">No branding changes have been recorded.</p>
          : state.audit.map((event) => <p key={`${event.command}-${event.createdAt}`}><strong>{event.command.replaceAll('.', ' ')}</strong> by {event.actorDisplayName} · version {event.profileVersion}<small>{new Date(event.createdAt).toLocaleString()} · {event.correlationId}</small></p>)}
      </details>
    </section>
  )
}

interface BrandingEditorProps {
  profile: BrandingProfile
  expectedVersion: number
  pending: boolean
  onSave: (draft: BrandingDraft) => void
  onPreview: (profile: BrandingProfile) => void
}

function BrandingEditor({ profile, expectedVersion, pending, onSave, onPreview }: BrandingEditorProps) {
  const [organizationName, setOrganizationName] = useState(profile.organizationName)
  const [shortName, setShortName] = useState(profile.shortName)
  const [browserTitle, setBrowserTitle] = useState(profile.browserTitle)
  const [primary, setPrimary] = useState(profile.colors.primary)
  const [primaryHover, setPrimaryHover] = useState(profile.colors.primaryHover)
  const [selectedSurface, setSelectedSurface] = useState(profile.colors.selectedSurface)
  const [focus, setFocus] = useState(profile.colors.focus)

  function values(): BrandingDraft {
    return {
      organizationName,
      shortName,
      browserTitle,
      colors: { primary, primaryHover, selectedSurface, focus },
      expectedVersion,
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSave(values())
  }

  function showPreview() {
    onPreview({
      ...profile,
      status: 'draft',
      source: 'draft',
      organizationName,
      shortName,
      browserTitle,
      colors: { primary, primaryHover, selectedSurface, focus },
    })
  }

  return (
    <form className="branding-form" onSubmit={submit}>
      <fieldset>
        <legend>Identity</legend>
        <label>Organization name<input required minLength={2} maxLength={120} value={organizationName} onChange={(event) => setOrganizationName(event.target.value)} /></label>
        <label>Application short name<input required minLength={2} maxLength={60} value={shortName} onChange={(event) => setShortName(event.target.value)} /></label>
        <label>Browser title<input required minLength={2} maxLength={80} value={browserTitle} onChange={(event) => setBrowserTitle(event.target.value)} /></label>
      </fieldset>
      <fieldset>
        <legend>Accessible presentation colours</legend>
        <ColorField label="Primary action" value={primary} onChange={setPrimary} />
        <ColorField label="Primary hover" value={primaryHover} onChange={setPrimaryHover} />
        <ColorField label="Selected surface" value={selectedSurface} onChange={setSelectedSurface} />
        <ColorField label="Focus indicator" value={focus} onChange={setFocus} />
      </fieldset>
      <div className="branding-form-actions">
        <button className="secondary-button" type="button" onClick={showPreview}><Eye size={16} aria-hidden="true" /> Preview</button>
        <button className="primary-button" type="submit" disabled={pending}><Save size={16} aria-hidden="true" /> {pending ? 'Saving draft' : 'Save draft'}</button>
      </div>
    </form>
  )
}

function ColorField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="branding-color-field"><span>{label}<small>{value}</small></span><input aria-label={`${label} colour`} type="color" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}
