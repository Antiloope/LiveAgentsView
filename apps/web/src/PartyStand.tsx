import { useMemo } from 'react'
import type { Session } from './types'
import { ARCHETYPES, NEEDS_ATTENTION, PROVIDER_LABEL, STATE_COLOR, archetypeFor, seededPercent } from './sprites'
import Portrait from './Portrait'
import { ProviderRune, StatusIcon } from './Glyphs'

interface Props {
  session: Session
  calm?: boolean
  selected?: boolean
  onSelect: (id: string) => void
}

export default function PartyStand({ session, calm, selected, onSelect }: Props) {
  const small = ARCHETYPES[archetypeFor(session.id, session.model)].small
  const hp = useMemo(() => seededPercent(session.id, 'hp'), [session.id])
  const mp = useMemo(() => seededPercent(session.id, 'mp'), [session.id])
  const needsAttention = NEEDS_ATTENTION.includes(session.state)

  return (
    <button
      type="button"
      className={`stand${calm ? ' calm' : ''}${needsAttention ? ' needs-attention' : ''}${selected ? ' selected' : ''}`}
      data-small={small ? 'true' : undefined}
      aria-label={`${PROVIDER_LABEL[session.provider]}, ${session.state}`}
      aria-pressed={selected}
      onClick={() => onSelect(session.id)}
    >
      <div className="stand-flag">
        <ProviderRune provider={session.provider} />
        <span>{PROVIDER_LABEL[session.provider]}</span>
      </div>
      <div className="rune-glow" style={{ background: `radial-gradient(ellipse at center, ${STATE_COLOR[session.state]}, transparent 70%)` }} />
      <div className="sprite-wrap">
        <StatusIcon state={session.state} />
        <Portrait sessionId={session.id} model={session.model} />
      </div>
      <div className="bars">
        <div className="bar-track">
          <div className="bar-fill" style={{ width: `${hp}%` }} />
        </div>
        <div className="bar-track mp">
          <div className="bar-fill mp" style={{ width: `${mp}%` }} />
        </div>
      </div>
    </button>
  )
}
