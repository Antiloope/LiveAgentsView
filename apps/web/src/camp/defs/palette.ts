import type { Palette } from './types'

/**
 * Craft Pixel camp palette — navy night + gold trim + parchment ink.
 * Index 0 is always transparent.
 */
export const CAMP_PALETTE: Palette = {
  roles: [
    'transparent',
    'outline',
    'skin',
    'skinShadow',
    'cloth',
    'clothShadow',
    'clothHighlight',
    'metal',
    'wood',
    'accent',
    'fire',
    'ground',
    'groundAlt',
    'ink',
  ],
  colors: [
    '#00000000',
    '#1a1008',
    '#e8b888',
    '#c08050',
    '#3a5f9a',
    '#243f72',
    '#6a98e0',
    '#c8d0e0',
    '#6a4a28',
    '#c5a059',
    '#f0d078',
    '#1a2618',
    '#243020',
    '#1a140c',
  ],
}

/** @deprecated Use CAMP_PALETTE — kept as alias during migration. */
export const PLACEHOLDER_PALETTE = CAMP_PALETTE

export function roleIndex(palette: Palette, role: string): number {
  const i = palette.roles.indexOf(role as Palette['roles'][number])
  return i < 0 ? 1 : i
}

export function withKitColors(
  base: Palette,
  overrides: Partial<Record<'skin' | 'skinShadow' | 'cloth' | 'clothShadow' | 'clothHighlight' | 'metal' | 'accent', string>>,
): Palette {
  const colors = [...base.colors]
  for (const [role, hex] of Object.entries(overrides)) {
    // A kit's color record also carries non-color fields (proportions), and
    // a name the palette has no role for is not a color either — both would
    // otherwise land in the color table and break every later parse of it.
    if (typeof hex !== 'string' || !hex) continue
    const i = base.roles.indexOf(role as Palette['roles'][number])
    if (i < 0) continue
    colors[i] = hex
  }
  return { roles: base.roles, colors }
}
