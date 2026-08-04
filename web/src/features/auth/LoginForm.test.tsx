import { fireEvent, render, screen } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { describe, expect, it, vi } from 'vitest'

import { LoginForm } from './LoginForm'

const options = {
  institutions: [
    {
      id: '1',
      name: 'Vision Hospital',
      sites: [
        { id: '2', name: 'Main Clinic' },
        { id: '3', name: 'Satellite Clinic' },
      ],
    },
  ],
}

describe('LoginForm', () => {
  it('submits credentials and selected context', () => {
    const onSubmit = vi.fn()
    render(<LoginForm options={options} pending={false} errorMessage={undefined} onSubmit={onSubmit} />)

    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'clinician' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'secret' } })
    fireEvent.change(screen.getByLabelText('Site'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(onSubmit).toHaveBeenCalledWith({
      username: 'clinician',
      password: 'secret',
      institutionId: '1',
      siteId: '3',
    })
  })

  it('has no automated accessibility violations', async () => {
    const { container } = render(
      <LoginForm options={options} pending={false} errorMessage={undefined} onSubmit={vi.fn()} />,
    )
    const results = await axe(container)
    expect(results.violations).toEqual([])
  })
})
