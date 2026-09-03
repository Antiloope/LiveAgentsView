import { useMemo } from 'react'
import type { Session } from './types'
import { PROVIDER_LABEL, seededPercent } from './sprites'
import Portrait from './Portrait'
import { ProviderRune } from './Glyphs'

interface Props {
  session: Session
  selected?: boolean
  onSelect: (id: string) => void
}

export default function QuestToken({ session, selected, onSelect }: Props) {
  const hp = useMemo(() => seededPercent(session.id, 'hp'), [session.id])
  const mp = useMemo(() => seededPercent(session.id, 'mp'), [session.id])

  return (
    <button
      type="button"
      className={`quest-token${selected ? ' selected' : ''}`}
      aria-label={`${PROVIDER_LABEL[session.provider]}, working`}
      aria-pressed={selected}
      onClick={() => onSelect(session.id)}
    >
      <div className="mini-sprite">
        <Portrait sessionId={session.id} model={session.model} />
      </div>
      <div className="mini-text">
        <div className="mini-provider">
          <ProviderRune provider={session.provider} className="provider-rune" />
          <span>{PROVIDER_LABEL[session.provider]}</span>
        </div>
        <div className="mini-repo" title={session.repo || session.cwd}>
          {session.repo || session.cwd}
        </div>
        <div className="mini-bars">
          <div className="bar-track">
            <div className="bar-fill" style={{ width: `${hp}%` }} />
          </div>
          <div className="bar-track mp">
            <div className="bar-fill mp" style={{ width: `${mp}%` }} />
          </div>
        </div>
      </div>
    </button>
  )
}
