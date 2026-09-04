import type { ReactNode } from 'react'
import OrnateCorners from '../../OrnateCorners'
import Button from '../Button'
import { DisplayTitle, HudLabel } from '../Type'
import './TopBar.css'

type Props = {
  characterCount: number
  archivedCount: number
  onRecruit: () => void
  onShowArchived: () => void
}

export default function TopBar({ characterCount, archivedCount, onRecruit, onShowArchived }: Props) {
  return (
    <header className="topbar">
      <OrnateCorners large />
      <div className="brand">
        <div className="crest" aria-hidden="true" />
        <div>
          <DisplayTitle className="topbar-title">LiveAgentsView</DisplayTitle>
          <HudLabel as="span" className="subtitle">
            {characterCount} character{characterCount === 1 ? '' : 's'} known
          </HudLabel>
        </div>
      </div>
      <div className="topbar-actions">
        <Button className="recruit-btn" onClick={onRecruit}>
          + Recruit
        </Button>
        <Button variant="secondary" onClick={onShowArchived}>
          Archived ({archivedCount})
        </Button>
      </div>
    </header>
  )
}
