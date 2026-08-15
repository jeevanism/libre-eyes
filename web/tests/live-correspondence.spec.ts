import { expect, test } from '@playwright/test'

const liveE2E = process.env.VISIONOPUS_LIVE_E2E === 'true'

test('saves a multiline synthetic correspondence draft through the real Go API', async ({ page }, testInfo) => {
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
  await page.goto('/patients/11111111-1111-4111-8111-111111111111/examination?tool=correspondence')
  const demo = page.getByRole('region', { name: /Correspondence draft/i })
  await demo.getByLabel('Subject').fill('Demo clinic update')
  await demo.getByLabel('Plain-text body').fill('First paragraph.\nSecond paragraph.')
  await demo.getByLabel('Demo footer').fill('VisionOpus demonstration')
  await demo.getByRole('button', { name: 'Save demo correspondence draft' }).click()
  await expect(demo.getByRole('status')).toHaveText('Demo correspondence draft saved. It remains uncommitted.')
  await page.screenshot({ path: testInfo.outputPath('correspondence-examination-workspace.png'), fullPage: true })
  expect(failedResponses).toEqual([])
})
