import type { Character } from './types'
import { RACE_LABEL } from './sprites'
import { PortraitThumb, RaceGlyph } from './ui'

interface Props {
  character: Character
  selected?: boolean
  onSelect: (id: string) => void
}

export default function QuestToken({ character, selected, onSelect }: Props) {
  return (
    <button
      type="button"
      className={`quest-token${selected ? ' selected' : ''}`}
      aria-label={`${RACE_LABEL[character.race]}, working`}
      aria-pressed={selected}
      onClick={() => onSelect(character.id)}
    >
      <div className="mini-sprite">
        <PortraitThumb characterId={character.id} race={character.race} />
      </div>
      <div className="mini-text">
        <div className="mini-provider">
          <RaceGlyph race={character.race} className="race-rune" />
          <span>{RACE_LABEL[character.race]}</span>
        </div>
        <div className="mini-repo" title={character.territory.path}>
          {character.repo || character.territory.path}
        </div>
      </div>
    </button>
  )
}
