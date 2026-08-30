import { fireEvent, render, screen } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ThemeControl } from './ThemeControl'
import { ThemeProvider } from './ThemeProvider'

const listeners = new Set<(event: MediaQueryListEvent) => void>()
let systemDark = false

function installMatchMedia() {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' && systemDark,
      media: query,
      onchange: null,
      addEventListener: (_type: string, listener: (event: MediaQueryListEvent) => void) => listeners.add(listener),
      removeEventListener: (_type: string, listener: (event: MediaQueryListEvent) => void) => listeners.delete(listener),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

describe('ThemeControl', () => {
  beforeEach(() => {
    localStorage.clear()
    listeners.clear()
    systemDark = false
    installMatchMedia()
    delete document.documentElement.dataset.theme
  })

  it('persists an explicit theme and can return to the system preference', () => {
    render(<ThemeProvider><ThemeControl /></ThemeProvider>)

    expect(document.documentElement.dataset.theme).toBe('light')
    expect(screen.getByRole('button', { name: 'Use system theme' })).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(screen.getByRole('button', { name: 'Use dark theme' }))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(localStorage.getItem('libreeyes-theme')).toBe('dark')

    systemDark = true
    listeners.forEach((listener) => listener({ matches: true } as MediaQueryListEvent))
    fireEvent.click(screen.getByRole('button', { name: 'Use system theme' }))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(localStorage.getItem('libreeyes-theme')).toBe('system')
  })

  it('has no automated accessibility violations', async () => {
    const { container } = render(<ThemeProvider><ThemeControl /></ThemeProvider>)
    expect((await axe(container)).violations).toEqual([])
  })
})
