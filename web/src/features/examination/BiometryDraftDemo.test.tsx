import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { BiometryDraftDemo } from './BiometryDraftDemo'

describe('BiometryDraftDemo', () => {
  it('renders synthetic bilateral measurements and no-side-effect disclosure', () => {
    vi.stubGlobal('fetch', vi.fn())
    render(<QueryClientProvider client={new QueryClient()}><BiometryDraftDemo csrfToken="synthetic" episodeId="episode" /></QueryClientProvider>)
    expect(screen.getByRole('heading', { name: 'Biometry draft' })).toBeVisible()
    expect(screen.getByText(/No device import, IOL calculation/)).toBeVisible()
    expect(screen.getAllByLabelText('Axial length (mm)')[0]).toHaveValue('23.50')
    expect(screen.getByRole('button', { name: 'Save demo draft' })).toBeEnabled()
  })
})
