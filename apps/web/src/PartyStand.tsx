import type { Character } from './types'
import { NEEDS_ATTENTION, RACE_LABEL, ACTIVITY_COLOR } from './sprites'
import { PortraitThumb, RaceGlyph, StatusGlyph, UnreadDot } from './ui'

interface Props {
  character: Character
  calm?: boolean
  selected?: boolean
  onSelect: (id: string) => void
}

/** Compact stand used where a Pixi scene is not mounted (e.g. drawer header). */
export default function PartyStand({ character, calm, selected, onSelect }: Props) {
  const needsAttention = NEEDS_ATTENTION.includes(character.activity)

  return (
    <button
      type="button"
      className={`stand${calm ? ' calm' : ''}${needsAttention ? ' needs-attention' : ''}${selected ? ' selected' : ''}`}
      aria-label={`${RACE_LABEL[character.race]}, ${character.activity}${character.unread ? ', unread' : ''}`}
      aria-pressed={selected}
      onClick={() => onSelect(character.id)}
    >
      <div className="stand-flag">
        <RaceGlyph race={character.race} />
        <span>{RACE_LABEL[character.race]}</span>
        {character.unread && <UnreadDot />}
      </div>
      <div
        className="rune-glow"
        style={{ background: `radial-gradient(ellipse at center, ${ACTIVITY_COLOR[character.activity]}, transparent 70%)` }}
      />
      <div className="sprite-wrap">
        <StatusGlyph activity={character.activity} />
        <PortraitThumb characterId={character.id} race={character.race} />
      </div>
    </button>
  )
}
