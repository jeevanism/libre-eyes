import { expect, test } from '@playwright/test'

const session = {
  user: { id: '1', displayName: 'Synthetic Clinician' },
  context: {
    institution: { id: '1', name: 'VisionOpus Development Hospital' },
    site: { id: '1', name: 'Development Eye Clinic' },
    firm: { id: '1', name: 'Development Ophthalmology' },
  },
  permissions: ['patient.search', 'patient.summary.read', 'patient.clinical_summary.read'],
  csrfToken: 'synthetic-csrf-token',
  idleExpiresAt: '2026-08-06T12:00:00Z',
  absoluteExpiresAt: '2026-08-06T18:00:00Z',
  contextVersion: 1,
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/auth/session', (route) => route.fulfill({ json: session }))
  await page.route('**/api/v1/patients/11111111-1111-4111-8111-111111111111/summary-header', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    expect(route.request().headers()['x-context-version']).toBe('1')
    await route.fulfill({ json: {
      patientId: '11111111-1111-4111-8111-111111111111', givenName: 'Alice', familyName: 'Patient',
      dateOfBirth: '1985-04-12', ageYears: 41, gender: 'female', deceased: false,
      dateOfDeath: null, clinicalDisclosure: 'authorized', patientVersion: 1, warningVersion: 1,
    } })
  })
  await page.route('**/api/v1/patients/11111111-1111-4111-8111-111111111111/summary-header/warnings', async (route) => {
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    await route.fulfill({ json: {
      patientId: '11111111-1111-4111-8111-111111111111',
      allergies: { status: 'present' }, alerts: { status: 'none_known' }, complete: true, warningVersion: 1,
      items: [{ kind: 'allergy', code: 'peanuts', label: 'Peanut allergy', reaction: 'Urticaria', comment: null }],
    } })
  })
})

test('renders the selected patient identity and warning details', async ({ page }) => {
  await page.goto('/patients/11111111-1111-4111-8111-111111111111')
  await expect(page.getByRole('heading', { name: 'Alice Patient' })).toBeVisible()
  await expect(page.getByText('12 Apr 1985')).toBeVisible()
  await expect(page.getByText('41 years')).toBeVisible()
  await expect(page.getByText('Peanut allergy')).toBeVisible()
  await expect(page.getByText('Urticaria')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
