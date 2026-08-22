import { fireEvent, render, screen } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { describe, expect, it, vi } from 'vitest'

import { ExaminationToolNavigation } from './ExaminationToolNavigation'

describe('ExaminationToolNavigation', () => {
  it('renders grouped tools and exposes the selected route tool', async () => {
    const { container } = render(<ExaminationToolNavigation onSelect={vi.fn()} selectedTool="consent" />)

    expect(screen.getByRole('navigation', { name: 'Examination tool navigation' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Consent form' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('Measurements')).toBeVisible()
    expect(screen.getByText('Findings')).toBeVisible()
    expect(screen.getByText('Documentation')).toBeVisible()
    expect((await axe(container)).violations).toEqual([])
  })

  it('selects a tool and closes the tablet drawer with Escape', () => {
    const onSelect = vi.fn()
    render(<ExaminationToolNavigation onSelect={onSelect} selectedTool="acuity" />)

    const toggle = screen.getByRole('button', { name: 'Examination tools' })
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')

    fireEvent.click(screen.getByRole('button', { name: 'Documentation' }))
    fireEvent.click(screen.getByRole('button', { name: 'Consent form' }))
    expect(onSelect).toHaveBeenCalledWith('consent')
    expect(toggle).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(toggle)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(toggle).toHaveFocus()
  })
})
