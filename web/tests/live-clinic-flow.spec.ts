import { expect, test } from '@playwright/test'

const liveE2E = process.env.LIBREEYES_LIVE_E2E === 'true'

test('operates only the synthetic development clinic-flow queue through the real Go API', async ({ page }, testInfo) => {
  test.skip(!liveE2E, 'Set LIBREEYES_LIVE_E2E=true to run against the local Go API and PostgreSQL')

  const failedResponses: string[] = []
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

  await page.getByRole('link', { name: 'Clinic flow' }).click()

  await expect(page.getByRole('heading', { name: 'Clinic flow queue' })).toBeVisible()
  const patientLabel = testInfo.project.name === 'tablet-chromium'
    ? 'Synthetic queue patient B'
    : 'Synthetic queue patient C'
  const ticket = page.getByRole('listitem').filter({ hasText: patientLabel })

  await ticket.getByRole('button', { name: `Mark arrived for ${patientLabel}` }).click()
  await expect(ticket.getByRole('button', { name: `Claim ticket for ${patientLabel}` })).toBeVisible()
  await ticket.getByRole('button', { name: `Claim ticket for ${patientLabel}` }).click()
  await expect(ticket.getByRole('button', { name: `Complete ticket for ${patientLabel}` })).toBeVisible()
  await ticket.getByRole('button', { name: `Complete ticket for ${patientLabel}` }).click()
  await expect(ticket.getByText('Completed')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('development-clinic-flow.png'), fullPage: true })
  expect(failedResponses).toEqual([])
})
