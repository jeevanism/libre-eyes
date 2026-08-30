import { expect, test } from '@playwright/test'

const liveE2E = process.env.LIBREEYES_LIVE_E2E === 'true'

test('authenticates and signs out through the real Go API', async ({ page }, testInfo) => {
  test.skip(!liveE2E, 'Set LIBREEYES_LIVE_E2E=true to run against the local Go API and PostgreSQL')

  const consoleErrors: string[] = []
  const failedResponses: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })
  page.on('response', (response) => {
    if (response.status() >= 400) failedResponses.push(`${response.status()} ${response.url()}`)
  })

  await page.goto('/login')
  await page.getByLabel('Username').fill(process.env.LIBREEYES_DEV_USERNAME ?? 'clinician')
  await page.getByLabel('Password').fill(process.env.LIBREEYES_DEV_PASSWORD ?? '')
  await page.getByLabel('Institution').selectOption({ label: 'LibreEyes Development Hospital' })
  await page.getByLabel('Site').selectOption({ label: 'Development Eye Clinic' })
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page).toHaveURL('/')
  await expect(page.getByRole('heading', { name: 'Home' })).toBeVisible()
  await expect(page.getByText('LibreEyes Development Hospital')).toBeVisible()
  await expect(page.getByText('Development Eye Clinic')).toBeVisible()
  await expect(page.getByText('Development Ophthalmology')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('authenticated-home.png'), fullPage: true })

  await page.getByRole('link', { name: 'Patient search' }).click()
  await page.getByLabel('Family name').fill('NoSuchSyntheticPatient')
  await page.getByLabel('Date of birth').fill('1901-01-01')
  await page.getByLabel('Gender').selectOption('unknown')
  await page.getByRole('button', { name: 'Search patients' }).click()
  await expect(page.getByText('No matching patients found in current institution.')).toBeVisible()
  await expect(page).toHaveURL('/patients/search')
  await page.screenshot({ path: testInfo.outputPath('patient-search-empty.png'), fullPage: true })

  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/login(?:\?|$)/)
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
  expect(failedResponses).toEqual([])
  expect(consoleErrors).toEqual([])
})
