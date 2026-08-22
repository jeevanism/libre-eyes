import { ChevronDown, ChevronRight, Menu, Stethoscope, X } from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'

import { examinationToolGroups, type ExaminationTool } from './examinationNavigation'

interface ExaminationToolNavigationProps {
  selectedTool: ExaminationTool
  onSelect: (tool: ExaminationTool) => void
}

export function ExaminationToolNavigation({ selectedTool, onSelect }: ExaminationToolNavigationProps) {
  const [open, setOpen] = useState(false)
  const selectedGroup = examinationToolGroups.find((group) => group.items.some((tool) => tool.id === selectedTool))?.id ?? 'measurements'
  const [expandedGroups, setExpandedGroups] = useState<string[]>([selectedGroup])
  const drawerID = useId().replace(/[:]/g, '')
  const toggleRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
        toggleRef.current?.focus()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open])

  function selectTool(tool: ExaminationTool) {
    const groupID = examinationToolGroups.find((group) => group.items.some((item) => item.id === tool))?.id
    if (groupID) {
      setExpandedGroups((current) => current.includes(groupID) ? current : [...current, groupID])
    }
    onSelect(tool)
    setOpen(false)
  }

  function toggleGroup(groupID: string) {
    setExpandedGroups((current) => current.includes(groupID)
      ? current.filter((id) => id !== groupID)
      : [...current, groupID])
  }

  return (
    <>
      <button
        aria-controls={drawerID}
        aria-expanded={open}
        className="examination-tool-drawer-toggle"
        onClick={() => setOpen((current) => !current)}
        ref={toggleRef}
        type="button"
      >
        {open ? <X size={17} aria-hidden="true" /> : <Menu size={17} aria-hidden="true" />}
        Examination tools
      </button>
      <aside aria-label="Examination tools" className={`examination-tool-rail${open ? ' examination-tool-rail-open' : ''}`} id={drawerID}>
        <div className="examination-tool-rail-heading">
          <Stethoscope size={17} aria-hidden="true" />
          <span>Tools</span>
        </div>
        <nav aria-label="Examination tool navigation">
          {examinationToolGroups.map((group) => (
            <section className="examination-tool-group" key={group.id}>
              <button
                aria-controls={`${drawerID}-${group.id}`}
                aria-expanded={expandedGroups.includes(group.id)}
                className="examination-tool-group-toggle"
                onClick={() => toggleGroup(group.id)}
                type="button"
              >
                <span>{group.label}</span>
                <ChevronDown className="examination-tool-group-chevron" size={14} aria-hidden="true" />
              </button>
              {expandedGroups.includes(group.id) && <ul id={`${drawerID}-${group.id}`}>
                {group.items.map(({ id, label, Icon }) => (
                  <li key={id}>
                    <button
                      aria-current={selectedTool === id ? 'page' : undefined}
                      className="examination-tool-link"
                      onClick={() => selectTool(id)}
                      type="button"
                    >
                      <Icon size={16} aria-hidden="true" />
                      <span>{label}</span>
                      {selectedTool === id && <ChevronRight className="examination-tool-link-indicator" size={15} aria-hidden="true" />}
                    </button>
                  </li>
                ))}
              </ul>}
            </section>
          ))}
        </nav>
      </aside>
    </>
  )
}
