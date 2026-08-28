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
  organizationName: 'VisionOpus Development Hospital',
  shortName: 'VisionOpus',
  browserTitle: 'VisionOpus',
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
})
