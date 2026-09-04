import type { Activity, Race } from '../types'
import {
  RaceRune,
  StatusIcon,
  UnreadDot,
  ShieldRune,
  HoodRune,
  SatchelRune,
  MapRune,
  SignpostRune,
  SparkRune,
  SearchRune,
} from '../Glyphs'

export type StatusGlyphKind = 'waiting' | 'failed' | 'working' | 'ready'
export type RaceGlyphKind = Race
export type DecorGlyphKind =
  | 'shield'
  | 'hood'
  | 'satchel'
  | 'map'
  | 'signpost'
  | 'spark'
  | 'search'
  | 'unread'

export type GlyphKind = StatusGlyphKind | DecorGlyphKind

type Props = {
  kind: GlyphKind | RaceGlyphKind
  /** When kind is a race id, pass race explicitly via kind. */
  className?: string
  activity?: Activity
}

const RACES: Race[] = ['claude-code', 'cursor']

function isRace(kind: string): kind is Race {
  return (RACES as string[]).includes(kind)
}

/**
 * Unified Craft Pixel glyph entry — status marks, race runes, recruit icons.
 * DOM path only; Pixi status cells stay in CampCanvas via STATUS_ICON_CELLS.
 */
export function Glyph({ kind, className }: Props) {
  if (isRace(kind)) return <RaceRune race={kind} className={className} />
  switch (kind) {
    case 'waiting':
    case 'failed':
    case 'working':
    case 'ready':
      return <StatusIcon activity={kind} className={className} />
    case 'shield':
      return <ShieldRune className={className} />
    case 'hood':
      return <HoodRune className={className} />
    case 'satchel':
      return <SatchelRune className={className} />
    case 'map':
      return <MapRune className={className} />
    case 'signpost':
      return <SignpostRune className={className} />
    case 'spark':
      return <SparkRune className={className} />
    case 'search':
      return <SearchRune className={className} />
    case 'unread':
      return <UnreadDot className={className} />
    default:
      return null
  }
}

export function StatusGlyph({ activity, className }: { activity: Activity; className?: string }) {
  return <StatusIcon activity={activity} className={className} />
}

export function RaceGlyph({ race, className }: { race: Race; className?: string }) {
  return <RaceRune race={race} className={className} />
}

export { UnreadDot }
