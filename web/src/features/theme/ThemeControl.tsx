import { Monitor, Moon, Sun } from 'lucide-react'

import { useTheme, type ThemePreference } from './ThemeProvider'

const options = [
  { value: 'light', label: 'Use light theme', Icon: Sun },
  { value: 'dark', label: 'Use dark theme', Icon: Moon },
  { value: 'system', label: 'Use system theme', Icon: Monitor },
] satisfies Array<{ value: ThemePreference; label: string; Icon: typeof Sun }>

export function ThemeControl() {
  const { preference, setPreference } = useTheme()

  return (
    <div className="theme-control" role="group" aria-label="Color theme">
      {options.map(({ value, label, Icon }) => (
        <button
          key={value}
          className="theme-option"
          type="button"
          title={label}
          aria-label={label}
          aria-pressed={preference === value}
          onClick={() => setPreference(value)}
        >
          <Icon size={16} aria-hidden="true" />
        </button>
      ))}
    </div>
  )
}
