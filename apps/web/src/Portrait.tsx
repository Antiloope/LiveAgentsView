import { useMemo } from 'react'
import { ARCHETYPES, archetypeFor } from './sprites'

interface Props {
  sessionId: string
  className?: string
}

// Renders the party sprite assigned to this session id — see sprites.ts for
// how the archetype is picked and why it varies per session, not provider.
export default function Portrait({ sessionId, className }: Props) {
  const archetype = useMemo(() => ARCHETYPES[archetypeFor(sessionId)], [sessionId])

  return (
    <svg viewBox="0 0 96 128" className={className ? `party-sprite ${className}` : 'party-sprite'} shapeRendering="crispEdges">
      {archetype.rects.map((rect, i) => (
        <rect key={i} x={rect.x} y={rect.y} width={rect.w} height={rect.h} fill={rect.fill} />
      ))}
    </svg>
  )
}
