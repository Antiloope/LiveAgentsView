import type { HTMLAttributes, ReactNode } from 'react'
import './Type.css'

type TitleProps = HTMLAttributes<HTMLHeadingElement> & {
  as?: 'h1' | 'h2' | 'h3'
  children: ReactNode
}

/** Cinzel Decorative display title (brand / panel heads). */
export function DisplayTitle({ as: Tag = 'h1', className, children, ...rest }: TitleProps) {
  const classes = ['ui-display', className].filter(Boolean).join(' ')
  return (
    <Tag className={classes} {...rest}>
      {children}
    </Tag>
  )
}

type TextProps = HTMLAttributes<HTMLElement> & {
  as?: 'span' | 'div' | 'p' | 'label' | 'h2'
  children: ReactNode
}

/** Pixelify Sans HUD label. */
export function HudLabel({ as: Tag = 'span', className, children, ...rest }: TextProps) {
  const classes = ['ui-hud', className].filter(Boolean).join(' ')
  return (
    <Tag className={classes} {...rest}>
      {children}
    </Tag>
  )
}

/** Cormorant Garamond body copy. */
export function BodyText({ as: Tag = 'p', className, children, ...rest }: TextProps) {
  const classes = ['ui-body', className].filter(Boolean).join(' ')
  return (
    <Tag className={classes} {...rest}>
      {children}
    </Tag>
  )
}
