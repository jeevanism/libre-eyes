import { expect, test } from '@playwright/test'

const session = {
  user: { id: '10', displayName: 'Synthetic Clinician' },
  context: {
    institution: { id: '1', name: 'Vision Hospital' },
    site: { id: '2', name: 'Main Clinic' },
    firm: { id: '3', name: 'Ophthalmology' },
  },
  permissions: ['patient.search', 'patient.duplicate_check'],
  csrfToken: 'synthetic-csrf-token',
  idleExpiresAt: '2026-08-06T12:00:00Z',
  absoluteExpiresAt: '2026-08-06T18:00:00Z',
  contextVersion: 1,
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/presentation/branding**', (route) => route.fulfill({ json: {
    profileVersion: 1, rowVersion: 1, status: 'published', source: 'institution',
    institutionId: 1, organizationName: 'Vision Hospital', shortName: 'LibreEyes', browserTitle: 'LibreEyes',
    colors: { primary: '#116466', primaryHover: '#0c5355', selectedSurface: '#deefee', focus: '#0b6fcc' },
  } }))
  await page.route('**/api/v1/auth/session', (route) => route.fulfill({ json: session }))
})

test('keeps criteria private and renders a responsive minimum-disclosure result', async ({ page }, testInfo) => {
  const consoleErrors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })

  await page.route('**/api/v1/patients/searches', async (route) => {
    expect(route.request().method()).toBe('POST')
    expect(route.request().headers()['x-csrf-token']).toBe('synthetic-csrf-token')
    expect(route.request().postDataJSON()).toEqual({
      criteria: {
        kind: 'demographic',
        familyName: 'Synthetic',
        dateOfBirth: '1980-02-03',
        gender: 'female',
      },
      limit: 25,
    })
    await route.fulfill({
      json: {
        items: [{
          patientId: '11111111-1111-4111-8111-111111111111',
          fullName: '<img src=x onerror=alert(1)> Synthetic',
          givenName: '<img src=x onerror=alert(1)>',
          familyName: 'Synthetic',
          dateOfBirth: '1980-02-03',
          gender: 'female',
          primaryIdentifier: { typeId: '7', label: 'Hospital number', value: 'SYN 001' },
          deceased: false,
          dateOfDeath: null,
        }],
        page: { hasMore: false, nextCursor: null },
      },
    })
  })

  await page.goto('/patients/search')
  await page.getByLabel('Family name').fill('Synthetic')
  await page.getByLabel('Date of birth').fill('1980-02-03')
  await page.getByLabel('Gender').selectOption('female')
  await page.getByRole('button', { name: 'Search patients' }).click()

  await expect(page).toHaveURL('/patients/search')
  await expect(page.getByText('<img src=x onerror=alert(1)> Synthetic')).toBeVisible()
  await expect(page.getByText('SYN 001')).toBeVisible()
  await expect(page.getByText('11111111-1111-4111-8111-111111111111')).toHaveCount(0)
  await expect(page.locator('.results-section img')).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.screenshot({ path: testInfo.outputPath('patient-search-light.png'), fullPage: true })

  await page.getByRole('button', { name: 'Use dark theme' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.screenshot({ path: testInfo.outputPath('patient-search-dark.png'), fullPage: true })
  await page.reload()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  expect(consoleErrors).toEqual([])
})
