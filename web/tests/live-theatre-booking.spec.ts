import { expect, test } from '@playwright/test'

const liveE2E = process.env.VISIONOPUS_LIVE_E2E === 'true'

test('operates only the synthetic development theatre board through the real Go API', async ({ page }, testInfo) => {
  test.skip(!liveE2E, 'Set VISIONOPUS_LIVE_E2E=true to run against the local Go API and PostgreSQL')
  const failedResponses: string[] = []
  page.on('response', (response) => { if (response.status() >= 400) failedResponses.push(`${response.status()} ${response.url()}`) })

  await page.goto('/login')
  await page.getByLabel('Username').fill(process.env.VISIONOPUS_DEV_USERNAME ?? 'clinician')
  await page.getByLabel('Password').fill(process.env.VISIONOPUS_DEV_PASSWORD ?? '')
  await page.getByLabel('Institution').selectOption({ label: 'VisionOpus Development Hospital' })
  await page.getByLabel('Site').selectOption({ label: 'Development Eye Clinic' })
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL('/')

  await page.getByRole('link', { name: 'Theatre schedule' }).click()
  await expect(page.getByRole('heading', { name: 'Theatre booking' })).toBeVisible()
  const label = testInfo.project.name === 'tablet-chromium' ? 'Synthetic booking request B' : 'Synthetic booking request A'
  const request = page.getByRole('listitem').filter({ hasText: label })
  await request.getByRole('button', { name: 'Schedule' }).click()
  await expect(request.getByText('scheduled')).toBeVisible()
  await request.getByRole('button', { name: 'Cancel' }).click()
  await expect(request.getByText('cancelled')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('development-theatre-booking.png'), fullPage: true })
  expect(failedResponses).toEqual([])
})
