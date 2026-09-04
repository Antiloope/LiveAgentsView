import { useMemo } from 'react'
import type { Race } from './types'
import { archetypeFor } from './sprites'
import { bakeKitBuffer } from './camp/defs/assembleKit'
import { bufferToCanvas } from './camp/bake/sheet'

interface Props {
  characterId: string
  race: Race
  className?: string
}

/** Party portrait from the procedural kit baker (shared with the Pixi camp). */
export default function Portrait({ characterId, race, className }: Props) {
  const src = useMemo(() => {
    const kitId = archetypeFor(characterId, race)
    const { buf, palette } = bakeKitBuffer(kitId, 'idle', 0)
    return bufferToCanvas(buf, palette, 2).toDataURL()
  }, [characterId, race])

  return (
    <img
      src={src}
      alt=""
      width={96}
      height={128}
      className={className ? `party-sprite ${className}` : 'party-sprite'}
      draggable={false}
      style={{ imageRendering: 'pixelated' }}
    />
  )
}
