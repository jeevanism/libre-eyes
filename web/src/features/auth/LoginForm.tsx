import { Eye, LoaderCircle, LockKeyhole } from 'lucide-react'
import { useEffect, useMemo, useState, type FormEvent } from 'react'

import type { LoginOptions, LoginRequest } from '../../api/client'
import { ThemeControl } from '../theme/ThemeControl'
import { useBranding } from '../branding/BrandingProvider'

interface LoginFormProps {
  options: LoginOptions
  pending: boolean
  errorMessage: string | undefined
  onSubmit: (request: LoginRequest) => void
}

export function LoginForm({ options, pending, errorMessage, onSubmit }: LoginFormProps) {
  const { profile, selectInstitution } = useBranding()
  const firstInstitution = options.institutions[0]
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [institutionID, setInstitutionID] = useState(firstInstitution?.id ?? '')
  const sites = useMemo(
    () => options.institutions.find((institution) => institution.id === institutionID)?.sites ?? [],
    [institutionID, options.institutions],
  )
  const [siteID, setSiteID] = useState(firstInstitution?.sites[0]?.id ?? '')

  useEffect(() => {
    selectInstitution(firstInstitution?.id)
  }, [firstInstitution?.id, selectInstitution])

  function changeInstitution(value: string) {
    setInstitutionID(value)
    selectInstitution(value)
    const institution = options.institutions.find((candidate) => candidate.id === value)
    setSiteID(institution?.sites[0]?.id ?? '')
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (username === '' || password === '' || institutionID === '' || siteID === '') {
      return
    }
    onSubmit({ username, password, institutionId: institutionID, siteId: siteID })
  }

  return (
    <main className="login-page">
      <section className="login-shell" aria-labelledby="login-heading">
        <header className="brand-lockup">
          <div className="brand-identity">
            <span className="brand-mark" aria-hidden="true"><Eye size={27} strokeWidth={2} /></span>
            <span className="brand-name">{profile.shortName}</span>
          </div>
          <ThemeControl />
        </header>

        <div className="login-panel">
          <div className="panel-heading">
            <span className="section-icon" aria-hidden="true"><LockKeyhole size={18} /></span>
            <h1 id="login-heading">Sign in</h1>
          </div>

          <form onSubmit={submit} noValidate>
            <label htmlFor="username">Username</label>
            <input
              id="username"
              name="username"
              autoComplete="username"
              maxLength={255}
              required
              value={username}
              onChange={(event) => setUsername(event.target.value)}
            />

            <label htmlFor="password">Password</label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              maxLength={1024}
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />

            <div className="field-grid">
              <div>
                <label htmlFor="institution">Institution</label>
                <select
                  id="institution"
                  value={institutionID}
                  required
                  onChange={(event) => changeInstitution(event.target.value)}
                >
                  {options.institutions.map((institution) => (
                    <option key={institution.id} value={institution.id}>{institution.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor="site">Site</label>
                <select id="site" value={siteID} required onChange={(event) => setSiteID(event.target.value)}>
                  {sites.map((site) => <option key={site.id} value={site.id}>{site.name}</option>)}
                </select>
              </div>
            </div>

            <div className="form-message" role="alert" aria-live="polite">
              {errorMessage ?? ''}
            </div>

            <button className="primary-button" type="submit" disabled={pending || options.institutions.length === 0}>
              {pending ? <LoaderCircle className="spinner" size={18} aria-hidden="true" /> : null}
              <span>{pending ? 'Signing in' : 'Sign in'}</span>
            </button>
          </form>
        </div>
      </section>
    </main>
  )
}
