import { useMemo } from 'react'
import type { Race } from './types'
import { ARCHETYPES, archetypeFor } from './sprites'

interface Props {
  characterId: string
  race: Race
  className?: string
}

// Renders the party sprite assigned to this character — see sprites.ts for
// how the archetype is picked: a pool per race, hashed by character id so
// individuals of the same race stay visually distinguishable.
export default function Portrait({ characterId, race, className }: Props) {
  const archetype = useMemo(() => ARCHETYPES[archetypeFor(characterId, race)], [characterId, race])

  return (
    <svg viewBox="0 0 96 128" className={className ? `party-sprite ${className}` : 'party-sprite'} shapeRendering="crispEdges">
      {archetype.rects.map((rect, i) => (
        <rect key={i} x={rect.x} y={rect.y} width={rect.w} height={rect.h} fill={rect.fill} />
      ))}
    </svg>
  )
}
