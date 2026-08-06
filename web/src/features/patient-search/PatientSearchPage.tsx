import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowRight,
  BadgeAlert,
  Info,
  LoaderCircle,
  RotateCcw,
  Search,
  ShieldCheck,
  Users,
} from 'lucide-react'
import { useEffect, useRef, useState, type FormEvent } from 'react'

import {
  ApiError,
  patientSearchAPI,
  type DuplicateCandidateRequest,
  type DuplicateCandidateResponse,
  type GenderCode,
  type PatientSearchCriteria,
  type PatientSearchPage as PatientSearchResponse,
  type PatientSearchRequest,
  type PatientSearchResult,
  type Session,
} from '../../api/client'
import { authKeys } from '../auth/queries'

type Operation = 'search' | 'duplicates'
type CriteriaMode = 'demographic' | 'identifier'

type Execution =
  | { kind: 'search'; request: PatientSearchRequest; pageNumber: number; formVersion: number }
  | { kind: 'duplicates'; request: DuplicateCandidateRequest; formVersion: number }

type ExecutionResult =
  | { kind: 'search'; response: PatientSearchResponse; pageNumber: number; formVersion: number }
  | { kind: 'duplicates'; response: DuplicateCandidateResponse; formVersion: number }

interface FormState {
  identifierTypeID: string
  identifierValue: string
  givenName: string
  familyName: string
  dateOfBirth: string
  gender: '' | GenderCode
}

const initialForm: FormState = {
  identifierTypeID: '', identifierValue: '', givenName: '',
  familyName: '', dateOfBirth: '', gender: '',
}

export function PatientSearchPage({ session }: { session: Session }) {
  const queryClient = useQueryClient()
  const [operation, setOperation] = useState<Operation>('search')
  const [criteriaMode, setCriteriaMode] = useState<CriteriaMode>('demographic')
  const [form, setForm] = useState<FormState>(initialForm)
  const [criteriaVersion, setCriteriaVersion] = useState(0)
  const [clientError, setClientError] = useState<string>()
  const [invalidFields, setInvalidFields] = useState<Set<keyof FormState>>(new Set())
  const committedSearch = useRef<PatientSearchRequest | null>(null)
  const clientErrorRef = useRef<HTMLDivElement>(null)
  const serviceErrorRef = useRef<HTMLDivElement>(null)
  const resultsRef = useRef<HTMLDivElement>(null)
  const canCheckDuplicates = session.permissions.includes('patient.duplicate_check')

  const execution = useMutation<ExecutionResult, Error, Execution>({
    mutationFn: async (input) => {
      if (input.kind === 'search') {
        return {
          kind: 'search',
          response: await patientSearchAPI.search(input.request, session.csrfToken),
          pageNumber: input.pageNumber,
          formVersion: input.formVersion,
        }
      }
      return {
        kind: 'duplicates',
        response: await patientSearchAPI.findDuplicateCandidates(input.request, session.csrfToken),
        formVersion: input.formVersion,
      }
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) {
        queryClient.removeQueries({ queryKey: authKeys.all })
      }
    },
  })

  function updateForm<Key extends keyof FormState>(key: Key, value: FormState[Key]) {
    setForm((current) => ({ ...current, [key]: value }))
    setClientError(undefined)
    setInvalidFields(new Set())
    setCriteriaVersion((current) => current + 1)
    committedSearch.current = null
    execution.reset()
  }

  function changeOperation(next: Operation) {
    setOperation(next)
    setClientError(undefined)
    setInvalidFields(new Set())
    setCriteriaVersion((current) => current + 1)
    committedSearch.current = null
    execution.reset()
  }

  function changeCriteriaMode(next: CriteriaMode) {
    setCriteriaMode(next)
    setClientError(undefined)
    setInvalidFields(new Set())
    setCriteriaVersion((current) => current + 1)
    committedSearch.current = null
    execution.reset()
  }

  function buildIdentifierCriteria(): Extract<PatientSearchCriteria, { kind: 'identifier' }> | undefined {
    const invalid = new Set<keyof FormState>()
    if (!/^[1-9][0-9]*$/.test(form.identifierTypeID)) invalid.add('identifierTypeID')
    if (form.identifierValue.trim() === '') invalid.add('identifierValue')
    if (invalid.size > 0) {
      setClientError('Enter a valid identifier type and identifier value.')
      setInvalidFields(invalid)
      return undefined
    }
    return {
      kind: 'identifier', identifierTypeId: form.identifierTypeID,
      value: form.identifierValue,
    }
  }

  function hasRequiredDemographics(): boolean {
    const invalid = new Set<keyof FormState>()
    if (form.familyName.trim() === '') invalid.add('familyName')
    if (form.dateOfBirth === '') invalid.add('dateOfBirth')
    if (invalid.size > 0) {
      setClientError('Enter a family name and date of birth.')
      setInvalidFields(invalid)
      return false
    }
    return true
  }

  function buildSearchCriteria(): PatientSearchCriteria | undefined {
    if (criteriaMode === 'identifier') return buildIdentifierCriteria()
    if (!hasRequiredDemographics()) return undefined
    if (form.gender === '') {
      setClientError('Select a gender for demographic search.')
      setInvalidFields(new Set(['gender']))
      return undefined
    }
    return {
      kind: 'demographic', familyName: form.familyName,
      dateOfBirth: form.dateOfBirth, gender: form.gender,
      ...(form.givenName.trim() === '' ? {} : { givenName: form.givenName }),
    }
  }

  function buildDuplicateCriteria(): DuplicateCandidateRequest['criteria'] | undefined {
    if (criteriaMode === 'identifier') return buildIdentifierCriteria()
    if (!hasRequiredDemographics()) return undefined
    if (form.givenName.trim() === '') {
      setClientError('Enter a given name for an exact demographic duplicate check.')
      setInvalidFields(new Set(['givenName']))
      return undefined
    }
    return {
      kind: 'demographic', givenName: form.givenName,
      familyName: form.familyName, dateOfBirth: form.dateOfBirth,
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setClientError(undefined)
    setInvalidFields(new Set())
    if (operation === 'duplicates') {
      const criteria = buildDuplicateCriteria()
      if (criteria === undefined) return
      execution.mutate({
        kind: 'duplicates', request: { criteria }, formVersion: criteriaVersion,
      })
      return
    }
    const criteria = buildSearchCriteria()
    if (criteria === undefined) return
    const request: PatientSearchRequest = { criteria, limit: 25 }
    committedSearch.current = request
    execution.mutate({
      kind: 'search', request, pageNumber: 1, formVersion: criteriaVersion,
    })
  }

  function nextPage() {
    if (execution.data?.kind !== 'search' || !execution.data.response.page.hasMore) return
    const request = committedSearch.current
    if (request === null) return
    execution.mutate({
      kind: 'search',
      request: { ...request, cursor: execution.data.response.page.nextCursor },
      pageNumber: execution.data.pageNumber + 1,
      formVersion: criteriaVersion,
    })
  }

  function reset() {
    setForm(initialForm)
    setClientError(undefined)
    setInvalidFields(new Set())
    setCriteriaVersion((current) => current + 1)
    committedSearch.current = null
    execution.reset()
  }

  const result = execution.data?.formVersion === criteriaVersion ? execution.data : undefined
  const serviceError = execution.error === null
    || execution.variables?.formVersion !== criteriaVersion
    ? undefined
    : errorMessage(execution.error)

  useEffect(() => {
    if (clientError !== undefined) clientErrorRef.current?.focus()
  }, [clientError])

  useEffect(() => {
    if (serviceError !== undefined) serviceErrorRef.current?.focus()
    else if (result !== undefined) resultsRef.current?.focus()
  }, [result, serviceError])

  return (
    <div className="patient-search-page">
      <header className="page-header">
        <div>
          <p className="page-eyebrow">Patients</p>
          <h1>Patient search</h1>
        </div>
        <div className="context-scope" aria-label={`Current institution: ${session.context.institution.name}`}>
          <ShieldCheck size={17} aria-hidden="true" />
          <span><small>Current institution</small>{session.context.institution.name}</span>
        </div>
      </header>

      {canCheckDuplicates ? (
        <div className="operation-tabs" role="group" aria-label="Patient lookup operation">
          <button aria-pressed={operation === 'search'} onClick={() => changeOperation('search')}>
            <Search size={16} aria-hidden="true" /> Patient search
          </button>
          <button aria-pressed={operation === 'duplicates'} onClick={() => changeOperation('duplicates')}>
            <Users size={16} aria-hidden="true" /> Duplicate check
          </button>
        </div>
      ) : null}

      <section className="search-workbench" aria-labelledby="criteria-heading">
        <form className="patient-search-form" onSubmit={submit} noValidate autoComplete="off">
          <div className="form-toolbar">
            <h2 id="criteria-heading">Search criteria</h2>
            <div className="criteria-switch" role="group" aria-label="Criteria type">
              <button type="button" aria-pressed={criteriaMode === 'demographic'} onClick={() => changeCriteriaMode('demographic')}>
                Demographics
              </button>
              <button type="button" aria-pressed={criteriaMode === 'identifier'} onClick={() => changeCriteriaMode('identifier')}>
                Identifier
              </button>
            </div>
          </div>

          {criteriaMode === 'identifier' ? (
            <div className="criteria-grid identifier-grid">
              <div className="field">
                <label htmlFor="identifier-type">Identifier type ID</label>
                <input
                  id="identifier-type" inputMode="numeric" pattern="[1-9][0-9]*" maxLength={19}
                  autoComplete="off"
                  aria-invalid={invalidFields.has('identifierTypeID') || undefined}
                  aria-describedby={invalidFields.has('identifierTypeID') ? 'patient-search-form-error' : undefined}
                  value={form.identifierTypeID}
                  onChange={(event) => updateForm('identifierTypeID', event.target.value)}
                />
              </div>
              <div className="field field-grow">
                <label htmlFor="identifier-value">Identifier value</label>
                <input
                  id="identifier-value" maxLength={255} autoComplete="off"
                  aria-invalid={invalidFields.has('identifierValue') || undefined}
                  aria-describedby={invalidFields.has('identifierValue') ? 'patient-search-form-error' : undefined}
                  value={form.identifierValue}
                  onChange={(event) => updateForm('identifierValue', event.target.value)}
                />
              </div>
            </div>
          ) : (
            <div className="criteria-grid demographic-grid">
              <div className="field">
                <label htmlFor="given-name">Given name</label>
                <input
                  id="given-name" maxLength={300} autoComplete="off"
                  required={operation === 'duplicates'}
                  aria-invalid={invalidFields.has('givenName') || undefined}
                  aria-describedby={invalidFields.has('givenName') ? 'patient-search-form-error' : undefined}
                  value={form.givenName}
                  onChange={(event) => updateForm('givenName', event.target.value)}
                />
              </div>
              <div className="field">
                <label htmlFor="family-name">Family name</label>
                <input
                  id="family-name" maxLength={100} required autoComplete="off"
                  aria-invalid={invalidFields.has('familyName') || undefined}
                  aria-describedby={invalidFields.has('familyName') ? 'patient-search-form-error' : undefined}
                  value={form.familyName}
                  onChange={(event) => updateForm('familyName', event.target.value)}
                />
              </div>
              <div className="field">
                <label htmlFor="date-of-birth">Date of birth</label>
                <input
                  id="date-of-birth" type="date" required autoComplete="off"
                  aria-invalid={invalidFields.has('dateOfBirth') || undefined}
                  aria-describedby={invalidFields.has('dateOfBirth') ? 'patient-search-form-error' : undefined}
                  value={form.dateOfBirth}
                  onChange={(event) => updateForm('dateOfBirth', event.target.value)}
                />
              </div>
              {operation === 'search' ? (
                <div className="field">
                  <label htmlFor="gender">Gender</label>
                  <select
                    id="gender" required value={form.gender}
                    aria-invalid={invalidFields.has('gender') || undefined}
                    aria-describedby={invalidFields.has('gender') ? 'patient-search-form-error' : undefined}
                    onChange={(event) => updateForm('gender', event.target.value as FormState['gender'])}
                  >
                    <option value="">Select</option>
                    <option value="female">Female</option>
                    <option value="male">Male</option>
                    <option value="other">Other</option>
                    <option value="unknown">Unknown</option>
                  </select>
                </div>
              ) : null}
            </div>
          )}

          {operation === 'duplicates' ? (
            <div className="scope-notice" role="note">
              <Info size={17} aria-hidden="true" />
              <span>Exact matches in the current institution only. No candidates does not exclude a duplicate.</span>
            </div>
          ) : null}

          <div className="form-actions">
            <div
              id="patient-search-form-error" ref={clientErrorRef} className="inline-error"
              role={clientError === undefined ? undefined : 'alert'} aria-live="polite"
              tabIndex={clientError === undefined ? undefined : -1}
            >
              {clientError ?? ''}
            </div>
            <button className="secondary-button" type="button" onClick={reset}>
              <RotateCcw size={16} aria-hidden="true" /> Reset
            </button>
            <button className="primary-button search-submit" type="submit" disabled={execution.isPending}>
              {execution.isPending ? <LoaderCircle className="spinner" size={17} aria-hidden="true" /> : <Search size={17} aria-hidden="true" />}
              {operation === 'duplicates' ? 'Check duplicates' : 'Search patients'}
            </button>
          </div>
        </form>
      </section>

      {serviceError !== undefined ? (
        <div ref={serviceErrorRef} className="service-message service-message-error" role="alert" tabIndex={-1}>
          <AlertTriangle size={18} aria-hidden="true" />
          <span>{serviceError}</span>
          {execution.error instanceof ApiError && execution.error.status === 401 ? <a href="/login">Sign in</a> : null}
        </div>
      ) : null}

      {execution.isPending ? <div className="results-status" role="status">Searching current institution</div> : null}
      {!execution.isPending && result !== undefined ? (
        <div ref={resultsRef} className="result-focus-target" tabIndex={-1}>
          {result.kind === 'search'
            ? <SearchResults result={result.response} pageNumber={result.pageNumber} onNext={nextPage} />
            : <DuplicateResults result={result.response} />}
        </div>
      ) : null}
    </div>
  )
}

function SearchResults({ result, pageNumber, onNext }: {
  result: PatientSearchResponse
  pageNumber: number
  onNext: () => void
}) {
  return (
    <section className="results-section" aria-labelledby="results-heading">
      <div className="results-header">
        <div>
          <h2 id="results-heading">Search results</h2>
          <p role="status">{result.items.length === 0 ? 'No matching patients found in current institution.' : `${result.items.length} ${result.items.length === 1 ? 'patient' : 'patients'} on page ${pageNumber}`}</p>
        </div>
        {result.page.hasMore ? (
          <button className="secondary-button" type="button" onClick={onNext}>
            Next page <ArrowRight size={16} aria-hidden="true" />
          </button>
        ) : null}
      </div>
      {result.items.length > 0 ? <PatientTable patients={result.items} caption={`Patient search results, page ${pageNumber}`} /> : null}
    </section>
  )
}

function DuplicateResults({ result }: { result: DuplicateCandidateResponse }) {
  return (
    <section className="results-section" aria-labelledby="duplicate-results-heading">
      <div className="results-header">
        <div>
          <h2 id="duplicate-results-heading">Duplicate candidates</h2>
          <p role="status">{result.candidates.length === 0 ? 'No exact candidates found.' : `${result.candidates.length} exact candidates found.`}</p>
        </div>
      </div>
      <div className="scope-notice" role="note">
        <Info size={17} aria-hidden="true" />
        <span>Exact matching and current-institution coverage only; no result proves uniqueness.</span>
      </div>
      {result.hardConflict ? (
        <div className="service-message service-message-error" role="alert">
          <BadgeAlert size={18} aria-hidden="true" /> Exact active identifier conflict found.
        </div>
      ) : null}
      {result.truncated ? (
        <div className="service-message service-message-warning" role="alert">
          <AlertTriangle size={18} aria-hidden="true" /> Candidate results are limited to 100 and are not exhaustive.
        </div>
      ) : null}
      {result.candidates.length > 0
        ? <PatientTable patients={result.candidates.map((candidate) => candidate.patient)} caption="Exact duplicate candidates" />
        : null}
    </section>
  )
}

function PatientTable({ patients, caption }: { patients: PatientSearchResult[]; caption: string }) {
  return (
    <div className="table-scroll">
      <table className="patient-table">
        <caption className="visually-hidden">{caption}</caption>
        <thead>
          <tr><th scope="col">Patient</th><th scope="col">Status</th><th scope="col">Primary identifier</th><th scope="col">Date of birth</th><th scope="col">Gender</th></tr>
        </thead>
        <tbody>
          {patients.map((patient) => (
            <tr key={patient.patientId}>
              <td><a className="patient-result-link" href={`/patients/${encodeURIComponent(patient.patientId)}`}><strong>{patient.fullName ?? 'Name not recorded'}</strong></a></td>
              <td>{patient.deceased ? <span className="status-badge status-deceased"><AlertTriangle size={14} aria-hidden="true" />Deceased{patient.dateOfDeath === null ? '' : `, ${formatDate(patient.dateOfDeath)}`}</span> : <span className="status-badge status-current">Current</span>}</td>
              <td>{patient.primaryIdentifier === null ? <span className="not-recorded">Not recorded</span> : <><span className="cell-label">{patient.primaryIdentifier.label}</span><span className="clinical-value">{patient.primaryIdentifier.value}</span></>}</td>
              <td className="clinical-value">{formatDate(patient.dateOfBirth)}</td>
              <td>{genderLabel(patient.gender)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function genderLabel(gender: GenderCode | null | undefined): string {
  if (gender === null || gender === undefined) return 'Not recorded'
  return gender.charAt(0).toUpperCase() + gender.slice(1)
}

function formatDate(value: string | null | undefined): string {
  if (value === null || value === undefined || value === '') return 'Not recorded'
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (match === null) return value
  const [, year, month, day] = match
  const date = new Date(Date.UTC(Number(year), Number(month) - 1, Number(day)))
  return new Intl.DateTimeFormat('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC',
  }).format(date)
}

function errorMessage(error: Error): string {
  if (!(error instanceof ApiError)) return 'Patient search is temporarily unavailable.'
  switch (error.status) {
    case 400: return 'Review the search fields and try again.'
    case 401: return 'Your session has ended. Sign in again.'
    case 403: return 'You do not have permission to perform this operation.'
    case 413: return 'The search request is too large.'
    case 429: return error.retryAfterSeconds === undefined
      ? 'Too many searches. Try again shortly.'
      : `Too many searches. Try again in ${error.retryAfterSeconds} seconds.`
    case 504: return 'Patient search timed out. Try again.'
    default: return 'Patient search is temporarily unavailable.'
  }
}
