import { useMemo } from 'react'
import type { Character } from './types'
import { ARCHETYPES, NEEDS_ATTENTION, RACE_LABEL, ACTIVITY_COLOR, archetypeFor, seededPercent } from './sprites'
import Portrait from './Portrait'
import { RaceRune, StatusIcon, UnreadDot } from './Glyphs'

interface Props {
  character: Character
  calm?: boolean
  selected?: boolean
  onSelect: (id: string) => void
}

export default function PartyStand({ character, calm, selected, onSelect }: Props) {
  const small = ARCHETYPES[archetypeFor(character.id, character.race)].small
  const hp = useMemo(() => seededPercent(character.id, 'hp'), [character.id])
  const mp = useMemo(() => seededPercent(character.id, 'mp'), [character.id])
  const needsAttention = NEEDS_ATTENTION.includes(character.activity)

  return (
    <button
      type="button"
      className={`stand${calm ? ' calm' : ''}${needsAttention ? ' needs-attention' : ''}${selected ? ' selected' : ''}`}
      data-small={small ? 'true' : undefined}
      aria-label={`${RACE_LABEL[character.race]}, ${character.activity}${character.unread ? ', unread' : ''}`}
      aria-pressed={selected}
      onClick={() => onSelect(character.id)}
    >
      <div className="stand-flag">
        <RaceRune race={character.race} />
        <span>{RACE_LABEL[character.race]}</span>
        {character.unread && <UnreadDot />}
      </div>
      <div
        className="rune-glow"
        style={{ background: `radial-gradient(ellipse at center, ${ACTIVITY_COLOR[character.activity]}, transparent 70%)` }}
      />
      <div className="sprite-wrap">
        <StatusIcon activity={character.activity} />
        <Portrait characterId={character.id} race={character.race} />
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
