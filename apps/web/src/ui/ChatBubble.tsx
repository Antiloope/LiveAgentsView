import type { ReactNode } from 'react'
import './ChatBubble.css'

export type ChatBubbleVariant = 'user' | 'assistant' | 'tool' | 'error' | 'system'

type Props = {
  variant?: ChatBubbleVariant
  label?: string
  children: ReactNode
}

/** Parchment transcript bubble for the session drawer. */
export default function ChatBubble({ variant = 'assistant', label, children }: Props) {
  return (
    <div className={`ui-bubble ui-bubble--${variant}`}>
      {label ? <span className="ui-bubble__label">{label}</span> : null}
      {children}
    </div>
  )
}
