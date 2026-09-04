import type { ButtonHTMLAttributes, ReactNode } from 'react'
import './Button.css'

export type ButtonVariant = 'primary' | 'secondary' | 'danger'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  children: ReactNode
}

/** Craft Pixel HUD button — gold primary, navy secondary, danger accent. */
export default function Button({ variant = 'primary', className, type = 'button', children, ...rest }: Props) {
  const classes = ['ui-btn', `ui-btn--${variant}`, className].filter(Boolean).join(' ')
  return (
    <button type={type} className={classes} {...rest}>
      {children}
    </button>
  )
}
