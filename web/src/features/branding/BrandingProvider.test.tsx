import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { brandingAPI, type BrandingProfile } from '../../api/client'
import { BrandingProvider, useBranding } from './BrandingProvider'

const published: BrandingProfile = {
  institutionId: 3,
  profileVersion: 2,
  rowVersion: 1,
  status: 'published',
  source: 'published',
  organizationName: 'Velox Group EyeCare',
  shortName: 'Velox EyeCare',
  browserTitle: 'Velox clinical workspace',
  colors: { primary: '#17543f', primaryHover: '#103d2e', selectedSurface: '#dcefe5', focus: '#005fcc' },
}

function Consumer() {
  const { profile, preview, cancelPreview } = useBranding()
  return <div><span>{profile.shortName}</span><button onClick={() => preview({ ...profile, shortName: 'Preview EyeCare', source: 'draft', status: 'draft' })}>Preview</button><button onClick={cancelPreview}>Cancel</button></div>
}

function renderProvider() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><BrandingProvider><Consumer /></BrandingProvider></QueryClientProvider>)
}

afterEach(() => {
  vi.restoreAllMocks()
  document.documentElement.removeAttribute('style')
  document.documentElement.removeAttribute('data-branding-source')
})

describe('BrandingProvider', () => {
  it('applies a published profile through semantic presentation tokens', async () => {
    vi.spyOn(brandingAPI, 'publicProfile').mockResolvedValue(published)
    renderProvider()

    await screen.findByText('Velox EyeCare')
    await waitFor(() => expect(document.documentElement.style.getPropertyValue('--action-primary')).toBe('#17543f'))
    expect(document.documentElement.dataset.brandingSource).toBe('published')
    expect(document.title).toBe('Velox clinical workspace')
  })

  it('supports local preview without publishing and can restore the published profile', async () => {
    vi.spyOn(brandingAPI, 'publicProfile').mockResolvedValue(published)
    renderProvider()
    await screen.findByText('Velox EyeCare')

    fireEvent.click(screen.getByRole('button', { name: 'Preview' }))
    expect(screen.getByText('Preview EyeCare')).toBeInTheDocument()
    expect(document.documentElement.dataset.brandingSource).toBe('preview')

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.getByText('Velox EyeCare')).toBeInTheDocument()
  })
})
