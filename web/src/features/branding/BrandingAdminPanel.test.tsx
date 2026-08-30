import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { brandingAPI, type BrandingState } from '../../api/client'
import { BrandingAdminPanel } from './BrandingAdminPanel'
import { BrandingProvider } from './BrandingProvider'

const profile = {
  institutionId: 3,
  profileVersion: 1,
  rowVersion: 1,
  status: 'published' as const,
  source: 'published' as const,
  organizationName: 'LibreEyes Development Hospital',
  shortName: 'LibreEyes',
  browserTitle: 'LibreEyes',
  colors: { primary: '#116466', primaryHover: '#0c5355', selectedSurface: '#deefee', focus: '#0b6fcc' },
}

const state: BrandingState = {
  effective: profile,
  published: profile,
  history: [{ profileVersion: 1, rowVersion: 1, status: 'published', updatedAt: '2026-08-28T10:00:00Z', publishedAt: '2026-08-28T10:00:00Z' }],
  audit: [],
}

function renderPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><BrandingProvider><BrandingAdminPanel csrfToken="csrf-token" /></BrandingProvider></QueryClientProvider>)
}

afterEach(() => vi.restoreAllMocks())

describe('BrandingAdminPanel', () => {
  it('saves a versioned draft and keeps publication explicit', async () => {
    vi.spyOn(brandingAPI, 'publicProfile').mockResolvedValue(profile)
    vi.spyOn(brandingAPI, 'state').mockResolvedValue(state)
    const draftState: BrandingState = { ...state, draft: { ...profile, profileVersion: 2, rowVersion: 1, status: 'draft', source: 'draft', shortName: 'Velox EyeCare' } }
    const save = vi.spyOn(brandingAPI, 'saveDraft').mockResolvedValue(draftState)
    renderPanel()

    fireEvent.change(await screen.findByLabelText('Application short name'), { target: { value: 'Velox EyeCare' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    await waitFor(() => expect(save).toHaveBeenCalledWith(expect.objectContaining({ shortName: 'Velox EyeCare', expectedVersion: 0 }), 'csrf-token'))
    expect(await screen.findByRole('status')).toHaveTextContent('live presentation has not changed')
  })

  it('has no automated accessibility violations', async () => {
    vi.spyOn(brandingAPI, 'publicProfile').mockResolvedValue(profile)
    vi.spyOn(brandingAPI, 'state').mockResolvedValue(state)
    const { container } = renderPanel()
    await screen.findByRole('heading', { name: 'Branding' })
    const results = await axe(container)
    expect(results.violations).toEqual([])
  })

  it('identifies every inaccessible colour before submitting', async () => {
    vi.spyOn(brandingAPI, 'publicProfile').mockResolvedValue(profile)
    vi.spyOn(brandingAPI, 'state').mockResolvedValue(state)
    const save = vi.spyOn(brandingAPI, 'saveDraft')
    renderPanel()

    fireEvent.change(await screen.findByLabelText('Primary action colour'), { target: { value: '#e4e651' } })
    fireEvent.change(screen.getByLabelText('Primary hover colour'), { target: { value: '#f03891' } })
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }))

    expect(save).not.toHaveBeenCalled()
    expect(document.documentElement).not.toHaveAttribute('data-branding-source', 'preview')
    expect(screen.getByLabelText('Primary action colour')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('Primary hover colour')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByText('Contrast 1.33:1 against white text; minimum 4.5:1.')).toBeVisible()
    expect(screen.getByText('Contrast 3.70:1 against white text; minimum 4.5:1.')).toBeVisible()
    expect(screen.getByRole('alert')).toHaveTextContent('Primary action is 1.33:1 and requires 4.5:1; Primary hover is 3.70:1 and requires 4.5:1.')
  })
})
