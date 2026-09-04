/** Fixed grid and kit contracts for the procedural camp. Sizes lock at design time. */

export type PaletteRole =
  | 'transparent'
  | 'outline'
  | 'skin'
  | 'skinShadow'
  | 'cloth'
  | 'clothShadow'
  | 'clothHighlight'
  | 'metal'
  | 'wood'
  | 'accent'
  | 'fire'
  | 'ground'
  | 'groundAlt'
  | 'ink'

export type KitLayerId = 'shadow' | 'body' | 'clothes' | 'head' | 'weapon' | 'fx'

export type AnimId = 'idle' | 'walk'

/** Placeholder until design locks canvas size. */
export const CHAR_SIZE = { w: 48, h: 64 } as const
export const TILE_SIZE = { w: 32, h: 32 } as const

export interface Palette {
  /** Index 0 must be transparent. */
  roles: PaletteRole[]
  /** Hex per role index; length matches roles. */
  colors: string[]
}

export interface PixelSheet {
  w: number
  h: number
  /** Palette indices per frame, row-major, length w*h each. */
  frames: Uint8Array[]
  palette: Palette
}

export interface KitDef {
  id: string
  label: string
  /** Layer draw order, bottom → top. */
  layers: KitLayerId[]
  animations: Partial<Record<AnimId, number>>
}
