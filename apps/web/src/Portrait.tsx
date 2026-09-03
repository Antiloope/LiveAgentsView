import { useMemo } from 'react'
import { ARCHETYPES, archetypeFor } from './sprites'

interface Props {
  sessionId: string
  model?: string
  className?: string
}

// Renders the party sprite assigned to this session id — see sprites.ts for
// how the archetype is picked: a fixed one for Claude Code's three recruit
// classes, a random per-session-id pick for everything else.
export default function Portrait({ sessionId, model, className }: Props) {
  const archetype = useMemo(() => ARCHETYPES[archetypeFor(sessionId, model)], [sessionId, model])

  return (
    <svg viewBox="0 0 96 128" className={className ? `party-sprite ${className}` : 'party-sprite'} shapeRendering="crispEdges">
      {archetype.rects.map((rect, i) => (
        <rect key={i} x={rect.x} y={rect.y} width={rect.w} height={rect.h} fill={rect.fill} />
      ))}
    </svg>
  )
}
