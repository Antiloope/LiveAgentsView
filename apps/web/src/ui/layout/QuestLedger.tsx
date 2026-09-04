import type { ReactNode } from 'react'
import OrnateCorners from '../../OrnateCorners'
import { HudLabel } from '../Type'

type Props = {
  count: number
  children: ReactNode
}

export default function QuestLedger({ count, children }: Props) {
  return (
    <aside className="sidebar" aria-label="Quest ledger">
      <OrnateCorners />
      <div className="sidebar-header">
        <HudLabel>Quest Ledger</HudLabel>
        <span className="count">{count}</span>
      </div>
      <div className="sidebar-list">{children}</div>
    </aside>
  )
}
