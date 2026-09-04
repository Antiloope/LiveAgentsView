import type { ReactNode, Ref } from 'react'

type Props = {
  open: boolean
  drawerRef: Ref<HTMLDivElement>
  width: number
  dragging: boolean
  onHandleMouseDown: (e: React.MouseEvent) => void
  label?: string
  children: ReactNode
}

/** Presentational drawer frame/scrim/handle — data/API stay in SessionDrawer. */
export default function SessionChrome({
  open,
  drawerRef,
  width,
  dragging,
  onHandleMouseDown,
  label,
  children,
}: Props) {
  return (
    <>
      <div className={`scrim${open ? ' open' : ''}`} />
      <div
        ref={drawerRef}
        className={`drawer${open ? ' open' : ''}`}
        style={{ width }}
        role="dialog"
        aria-label={label}
        aria-hidden={!open}
      >
        <div
          className={`drawer-handle${dragging ? ' dragging' : ''}`}
          onMouseDown={onHandleMouseDown}
          title="Drag to resize"
        />
        <div className="drawer-body">{children}</div>
      </div>
    </>
  )
}
