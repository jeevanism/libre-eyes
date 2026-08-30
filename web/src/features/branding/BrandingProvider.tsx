import { useQuery } from '@tanstack/react-query'
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'

import { brandingAPI, type BrandingProfile } from '../../api/client'

export const defaultBrandingProfile: BrandingProfile = {
  profileVersion: 1,
  rowVersion: 1,
  status: 'default',
  source: 'default',
  organizationName: 'LibreEyes',
  shortName: 'LibreEyes',
  browserTitle: 'LibreEyes',
  colors: {
    primary: '#116466',
    primaryHover: '#0c5355',
    selectedSurface: '#deefee',
    focus: '#0b6fcc',
  },
}

interface BrandingContextValue {
  profile: BrandingProfile
  isFallback: boolean
  selectInstitution: (institutionId: string | undefined) => void
  preview: (profile: BrandingProfile) => void
  cancelPreview: () => void
  isPreviewing: boolean
}

const fallbackContext: BrandingContextValue = {
  profile: defaultBrandingProfile,
  isFallback: true,
  selectInstitution: () => undefined,
  preview: () => undefined,
  cancelPreview: () => undefined,
  isPreviewing: false,
}

const BrandingContext = createContext<BrandingContextValue>(fallbackContext)

export function BrandingProvider({ children }: { children: ReactNode }) {
  const [institutionId, setInstitutionId] = useState<string>()
  const [previewProfile, setPreviewProfile] = useState<BrandingProfile>()
  const query = useQuery({
    queryKey: ['branding', 'published', institutionId ?? 'default'],
    queryFn: () => brandingAPI.publicProfile(institutionId),
  })
  const published = query.data ?? defaultBrandingProfile
  const profile = previewProfile ?? published
  const selectInstitution = useCallback((next: string | undefined) => {
    setPreviewProfile(undefined)
    setInstitutionId(next)
  }, [])
  const preview = useCallback((next: BrandingProfile) => setPreviewProfile(next), [])
  const cancelPreview = useCallback(() => setPreviewProfile(undefined), [])

  useEffect(() => {
    const root = document.documentElement
    root.style.setProperty('--action-primary', profile.colors.primary)
    root.style.setProperty('--action-primary-hover', profile.colors.primaryHover)
    root.style.setProperty('--surface-selected', profile.colors.selectedSurface)
    root.style.setProperty('--focus-ring', profile.colors.focus)
    root.dataset.brandingSource = previewProfile ? 'preview' : profile.source
    document.title = profile.browserTitle
  }, [previewProfile, profile])

  const value = useMemo<BrandingContextValue>(() => ({
    profile,
    isFallback: query.isError || query.data === undefined,
    selectInstitution,
    preview,
    cancelPreview,
    isPreviewing: previewProfile !== undefined,
  }), [cancelPreview, preview, profile, previewProfile, query.data, query.isError, selectInstitution])

  return <BrandingContext.Provider value={value}>{children}</BrandingContext.Provider>
}

export function useBranding() {
  return useContext(BrandingContext)
}
