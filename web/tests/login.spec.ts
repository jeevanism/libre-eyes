import { expect, test } from '@playwright/test'

const loginOptions = {
  institutions: [
    { id: '1', name: 'Vision Hospital', sites: [{ id: '2', name: 'Main Clinic' }] },
  ],
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/auth/login-options', (route) => route.fulfill({ json: loginOptions }))
  await page.route('**/api/v1/auth/session', (route) => route.fulfill({
    status: 401,
    contentType: 'application/problem+json',
    body: JSON.stringify({ type: 'about:blank', title: 'Authentication required', status: 401, code: 'unauthenticated', correlationId: 'test' }),
  }))
})

test('renders an operable login form without console errors', async ({ page }, testInfo) => {
  const errors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error') errors.push(message.text())
  })

  await page.goto('/login')
  await expect(page.getByText('VisionOpus')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible()
  await expect(page.getByLabel('Institution')).toHaveValue('1')
  await expect(page.getByLabel('Site')).toHaveValue('2')
  await page.getByLabel('Username').fill('clinician')
  await page.getByLabel('Password').fill('synthetic-password')
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeEnabled()
  await page.screenshot({ path: testInfo.outputPath('login.png'), fullPage: true })
  expect(errors).toEqual([])
})
