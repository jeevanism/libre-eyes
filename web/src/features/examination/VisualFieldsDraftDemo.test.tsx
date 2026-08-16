import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import { VisualFieldsDraftDemo } from './VisualFieldsDraftDemo'

describe('VisualFieldsDraftDemo', () => {
  it('renders the bounded synthetic strategy and both-eye result controls', () => {
    vi.stubGlobal('fetch', vi.fn())
    render(<QueryClientProvider client={new QueryClient()}><VisualFieldsDraftDemo csrfToken="synthetic" episodeId="episode" /></QueryClientProvider>)
    expect(screen.getByRole('heading', { name: 'Visual fields draft' })).toBeVisible()
    expect(screen.getByLabelText('Demo strategy')).toHaveValue('Demo SITA Standard')
    expect(screen.getByLabelText('Demo pattern')).toHaveValue('Demo 24-2')
    expect(screen.getByLabelText('Left-eye result')).toBeVisible()
  })
})
