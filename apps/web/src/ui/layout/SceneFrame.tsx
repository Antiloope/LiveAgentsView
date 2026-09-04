import type { ReactNode } from 'react'
import OrnateCorners from '../../OrnateCorners'

type Props = {
  children: ReactNode
}

export default function SceneFrame({ children }: Props) {
  return (
    <div className="scene">
      <OrnateCorners large />
      {children}
    </div>
  )
}
