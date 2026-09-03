import { useMemo } from 'react'
import type { Character } from './types'
import { RACE_LABEL, seededPercent } from './sprites'
import Portrait from './Portrait'
import { RaceRune } from './Glyphs'

interface Props {
  character: Character
  selected?: boolean
  onSelect: (id: string) => void
}

export default function QuestToken({ character, selected, onSelect }: Props) {
  const hp = useMemo(() => seededPercent(character.id, 'hp'), [character.id])
  const mp = useMemo(() => seededPercent(character.id, 'mp'), [character.id])

  return (
    <button
      type="button"
      className={`quest-token${selected ? ' selected' : ''}`}
      aria-label={`${RACE_LABEL[character.race]}, working`}
      aria-pressed={selected}
      onClick={() => onSelect(character.id)}
    >
      <div className="mini-sprite">
        <Portrait characterId={character.id} race={character.race} />
      </div>
      <div className="mini-text">
        <div className="mini-provider">
          <RaceRune race={character.race} className="race-rune" />
          <span>{RACE_LABEL[character.race]}</span>
        </div>
        <div className="mini-repo" title={character.territory.path}>
          {character.repo || character.territory.path}
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
