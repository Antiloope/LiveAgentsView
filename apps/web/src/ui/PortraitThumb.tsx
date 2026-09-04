import type { Race } from '../types'
import Portrait from '../Portrait'

type Props = {
  characterId: string
  race: Race
  className?: string
}

/** Thin portrait wrapper for ledger / drawer / recruit thumbs. */
export default function PortraitThumb({ characterId, race, className }: Props) {
  return <Portrait characterId={characterId} race={race} className={className} />
}
