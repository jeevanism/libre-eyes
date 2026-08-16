import { expect, test } from '@playwright/test'

const liveE2E = process.env.VISIONOPUS_LIVE_E2E === 'true'

test('saves and reloads an approved EyeDraw development draft through the real Go API', async ({ page }, testInfo) => {
  test.skip(!liveE2E, 'Set VISIONOPUS_LIVE_E2E=true to run against the local Go API and PostgreSQL')

  const failedResponses: string[] = []
  const pageErrors: string[] = []
  page.on('response', (response) => {
    if (response.status() >= 400) failedResponses.push(`${response.status()} ${response.url()}`)
  })
  page.on('pageerror', (error) => pageErrors.push(error.message))

  await page.goto('/login')
  await page.getByLabel('Username').fill(process.env.VISIONOPUS_DEV_USERNAME ?? 'clinician')
  await page.getByLabel('Password').fill(process.env.VISIONOPUS_DEV_PASSWORD ?? '')
  await page.getByLabel('Institution').selectOption({ label: 'VisionOpus Development Hospital' })
  await page.getByLabel('Site').selectOption({ label: 'Development Eye Clinic' })
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL('/')

  await page.goto('/patients/11111111-1111-4111-8111-111111111111/examination?tool=drawing')
  const demo = page.locator('.eyedraw-draft-demo')
  await expect(demo.getByRole('button', { name: 'Add anterior segment' })).toBeEnabled()
  await demo.getByRole('button', { name: 'Add anterior segment' }).click()
  await demo.getByLabel('Anterior segment pupil size').selectOption('Medium')
  await demo.getByRole('button', { name: 'Add PCIOL' }).click()
  await demo.getByRole('button', { name: 'Add phako incision' }).click()
  await demo.getByRole('button', { name: 'Add side port' }).click()
  await demo.getByRole('button', { name: 'Save demo draft' }).click()
  await expect(demo.getByRole('status')).toHaveText('Demo drawing draft saved. It remains uncommitted.')
  await demo.getByRole('button', { name: 'Reload saved draft' }).click()
  await expect(demo.getByRole('button', { name: 'Update demo draft' })).toBeVisible()
  await expect(demo.getByRole('alert')).not.toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('eyedraw-examination-workspace.png'), fullPage: true })
  expect(failedResponses).toEqual([])
  expect(pageErrors).toEqual([])
})
