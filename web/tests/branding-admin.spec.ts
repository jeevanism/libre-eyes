import { expect, test } from '@playwright/test'

import type { BrandingDraft, BrandingProfile, BrandingState } from '../src/api/client'

const session = {
  user: { id: '1', displayName: 'Demo Institution Administrator' },
  context: {
    institution: { id: '3', name: 'LibreEyes Development Hospital' },
    site: { id: '3', name: 'Development Eye Clinic' },
    firm: { id: '3', name: 'Development Ophthalmology' },
  },
  permissions: ['admin.development.read', 'admin.development.manage'],
  capabilities: ['administration'],
  csrfToken: 'branding-csrf-token',
  idleExpiresAt: '2026-08-28T16:00:00Z',
  absoluteExpiresAt: '2026-08-28T20:00:00Z',
  contextVersion: 1,
}

const baseProfile: BrandingProfile = {
  institutionId: 3,
  profileVersion: 1,
  rowVersion: 1,
  status: 'published',
  source: 'published',
  organizationName: 'LibreEyes Development Hospital',
  shortName: 'LibreEyes',
  browserTitle: 'LibreEyes',
  colors: { primary: '#116466', primaryHover: '#0c5355', selectedSurface: '#deefee', focus: '#0b6fcc' },
}

test('previews, saves, and publishes an institution branding profile', async ({ page }, testInfo) => {
  let publicProfile = baseProfile
  let brandingState: BrandingState = {
    effective: baseProfile,
    published: baseProfile,
    history: [{ profileVersion: 1, rowVersion: 1, status: 'published', updatedAt: '2026-08-28T10:00:00Z', publishedAt: '2026-08-28T10:00:00Z' }],
    audit: [],
  }

  await page.route('**/api/v1/auth/session', (route) => route.fulfill({ json: session }))
  await page.route('**/api/v1/presentation/branding**', (route) => route.fulfill({ json: publicProfile }))
  await page.route('**/api/v1/admin/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (path === '/api/v1/admin/branding' && request.method() === 'GET') {
      await route.fulfill({ json: brandingState })
      return
    }
    if (path === '/api/v1/admin/branding/draft' && request.method() === 'PUT') {
      expect(request.headers()['x-csrf-token']).toBe('branding-csrf-token')
      const body = request.postDataJSON() as BrandingDraft
      expect(body).toMatchObject({ shortName: 'Velox EyeCare', expectedVersion: 0 })
      const draft: BrandingProfile = {
        ...baseProfile,
        organizationName: body.organizationName,
        shortName: body.shortName,
        browserTitle: body.browserTitle,
        colors: body.colors,
        profileVersion: 2,
        rowVersion: 1,
        status: 'draft',
        source: 'draft',
      }
      brandingState = { ...brandingState, draft, history: [{ profileVersion: 2, rowVersion: 1, status: 'draft', updatedAt: '2026-08-28T11:00:00Z' }, ...brandingState.history] }
      await route.fulfill({ json: brandingState })
      return
    }
    if (path === '/api/v1/admin/branding/publish' && request.method() === 'POST') {
      expect(request.headers()['x-csrf-token']).toBe('branding-csrf-token')
      expect(request.postDataJSON()).toEqual({ expectedVersion: 1 })
      publicProfile = { ...brandingState.draft!, rowVersion: 2, status: 'published', source: 'published' }
      brandingState = {
        effective: publicProfile,
        published: publicProfile,
        history: [
          { profileVersion: 2, rowVersion: 2, status: 'published', updatedAt: '2026-08-28T11:01:00Z', publishedAt: '2026-08-28T11:01:00Z' },
          { profileVersion: 1, rowVersion: 2, status: 'superseded', updatedAt: '2026-08-28T11:01:00Z', publishedAt: '2026-08-28T10:00:00Z' },
        ],
        audit: [],
      }
      await route.fulfill({ json: brandingState })
      return
    }
    if (path === '/api/v1/admin/contexts') {
      await route.fulfill({ json: { institution: { id: 3, name: 'LibreEyes Development Hospital' }, sites: [], firms: [] } })
      return
    }
    await route.fulfill({ json: [] })
  })

  await page.goto('/admin')
  await page.getByRole('button', { name: 'Branding' }).click()
  await expect(page.getByRole('heading', { name: 'Branding' })).toBeVisible()

  await page.getByLabel('Primary action colour').fill('#e4e651')
  await page.getByLabel('Primary hover colour').fill('#f03891')
  await page.getByRole('button', { name: 'Preview' }).click()
  await expect(page.locator('html')).not.toHaveAttribute('data-branding-source', 'preview')
  await page.getByRole('button', { name: 'Save draft' }).click()
  await expect(page.getByRole('alert')).toContainText('Primary action is 1.33:1 and requires 4.5:1')
  await expect(page.getByText('Contrast 3.70:1 against white text; minimum 4.5:1.')).toBeVisible()
  await expect(page.getByLabel('Primary action colour')).toHaveAttribute('aria-invalid', 'true')
  await page.screenshot({ path: testInfo.outputPath('branding-contrast-errors.png'), fullPage: true })
  await page.getByLabel('Primary action colour').fill(baseProfile.colors.primary)
  await page.getByLabel('Primary hover colour').fill(baseProfile.colors.primaryHover)

  await page.getByLabel('Application short name').fill('Velox EyeCare')
  await page.getByRole('button', { name: 'Preview' }).click()
  await expect(page.getByRole('link', { name: 'Velox EyeCare home' })).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('data-branding-source', 'preview')

  await page.getByRole('button', { name: 'Save draft' }).click()
  await expect(page.getByRole('status')).toContainText('live presentation has not changed')
  await page.getByRole('button', { name: 'Publish draft' }).click()
  await expect(page.getByRole('status')).toContainText('published successfully')
  await expect(page.getByText('Published v2')).toBeVisible()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.screenshot({ path: testInfo.outputPath('branding-admin.png'), fullPage: true })
})
