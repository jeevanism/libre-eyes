import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Eye, History, Palette, RotateCcw, Save, Send } from 'lucide-react'
import { useState, type FormEvent } from 'react'

import { brandingAPI, type BrandingDraft, type BrandingProfile } from '../../api/client'
import { evaluateBrandingContrast, type BrandingContrastCheck } from './contrast'
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
  const [showContrastErrors, setShowContrastErrors] = useState(false)
  const contrastChecks = {
    primary: evaluateBrandingContrast('primary', primary),
    primaryHover: evaluateBrandingContrast('primaryHover', primaryHover),
    selectedSurface: evaluateBrandingContrast('selectedSurface', selectedSurface),
    focus: evaluateBrandingContrast('focus', focus),
  }
  const invalidContrastChecks = Object.values(contrastChecks).filter((check) => !check.valid)

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
    if (!contrastActionsAllowed()) return
    onSave(values())
  }

  function showPreview() {
    if (!contrastActionsAllowed()) return
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

  function contrastActionsAllowed() {
    const valid = invalidContrastChecks.length === 0
    setShowContrastErrors(!valid)
    return valid
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
        <ColorField check={contrastChecks.primary} value={primary} onChange={setPrimary} />
        <ColorField check={contrastChecks.primaryHover} value={primaryHover} onChange={setPrimaryHover} />
        <ColorField check={contrastChecks.selectedSurface} value={selectedSurface} onChange={setSelectedSurface} />
        <ColorField check={contrastChecks.focus} value={focus} onChange={setFocus} />
      </fieldset>
      {showContrastErrors && invalidContrastChecks.length > 0 && (
        <p className="branding-contrast-summary inline-error" role="alert">
          {formatInvalidContrastSummary(invalidContrastChecks)} Correct the highlighted colours before saving this draft.
        </p>
      )}
      <div className="branding-form-actions">
        <button className="secondary-button" type="button" onClick={showPreview}><Eye size={16} aria-hidden="true" /> Preview</button>
        <button className="primary-button" type="submit" disabled={pending}><Save size={16} aria-hidden="true" /> {pending ? 'Saving draft' : 'Save draft'}</button>
      </div>
    </form>
  )
}

function ColorField({ check, value, onChange }: { check: BrandingContrastCheck; value: string; onChange: (value: string) => void }) {
  const descriptionID = `branding-${check.field}-contrast`
  return (
    <label className={`branding-color-field ${check.valid ? '' : 'branding-color-field-invalid'}`}>
      <span>
        {check.label}
        <small>{value}</small>
        <small id={descriptionID} className={check.valid ? 'branding-contrast-valid' : 'branding-contrast-invalid'}>
          Contrast {check.ratio.toFixed(2)}:1 against {check.against}; minimum {check.minimum.toFixed(1)}:1.
        </small>
      </span>
      <input
        aria-describedby={descriptionID}
        aria-invalid={!check.valid}
        aria-label={`${check.label} colour`}
        type="color"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  )
}

function formatInvalidContrastSummary(checks: BrandingContrastCheck[]) {
  const details = checks.map((check) => `${check.label} is ${check.ratio.toFixed(2)}:1 and requires ${check.minimum.toFixed(1)}:1`)
  return details.join('; ') + '.'
}
