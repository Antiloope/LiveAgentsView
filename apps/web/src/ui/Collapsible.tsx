import { useId, useState, type ReactNode } from 'react'
import { ChevronRune } from '../Glyphs'
import './Collapsible.css'

export type CollapsibleTone = 'tool' | 'thinking' | 'raw'

type Props = {
  /** Short HUD label for the row — what this block is. */
  label: string
  /** One-line preview of the content, shown on the row itself. */
  summary?: string
  tone?: CollapsibleTone
  children: ReactNode
}

/**
 * A transcript row that keeps its detail out of the way: the header reads as
 * one line, and the chevron opens the full block underneath it.
 */
export default function Collapsible({ label, summary, tone = 'tool', children }: Props) {
  const [open, setOpen] = useState(false)
  const bodyId = useId()
  return (
    <div className={`ui-fold ui-fold--${tone}${open ? ' is-open' : ''}`}>
      <button
        type="button"
        className="ui-fold__row"
        aria-expanded={open}
        aria-controls={bodyId}
        onClick={() => setOpen((v) => !v)}
      >
        <ChevronRune className="ui-fold__chevron" />
        <span className="ui-fold__label">{label}</span>
        {summary ? <span className="ui-fold__summary">{summary}</span> : null}
      </button>
      <div id={bodyId} className="ui-fold__body" hidden={!open}>
        {children}
      </div>
    </div>
  )
}
