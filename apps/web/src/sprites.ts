import type { Provider, State } from './types'

export interface PixelRect {
  x: number
  y: number
  w: number
  h: number
  fill: string
}

export interface Archetype {
  label: string
  small?: boolean
  rects: PixelRect[]
}

function r(x: number, y: number, w: number, h: number, fill: string): PixelRect {
  return { x, y, w, h, fill }
}

// Shared chibi base: head, torso, belt, legs, boots. Archetypes override
// skin/garment colors and add their own headgear/prop rects on top.
function baseBody(skin: string, main: string, trim: string, boot: string): PixelRect[] {
  return [
    r(32, 16, 32, 32, skin),
    r(40, 32, 6, 8, '#1a1a1a'),
    r(50, 32, 6, 8, '#1a1a1a'),
    r(24, 48, 48, 40, main),
    r(24, 76, 48, 8, trim),
    r(32, 88, 16, 32, main),
    r(48, 88, 16, 32, main),
    r(32, 112, 16, 8, boot),
    r(48, 112, 16, 8, boot),
  ]
}

// Hand-authored on an 8px grid (96x128 canvas per character) — no image
// generation is available for this build, so the party art is code-drawn.
export const ARCHETYPES: Record<string, Archetype> = {
  'human-mage': {
    label: 'Human Mage',
    rects: [
      ...baseBody('#e8b088', '#2f4d7a', '#e0b84a', '#241a12'),
      r(20, 8, 56, 8, '#1f2f52'),
      r(40, 0, 16, 12, '#1f2f52'),
      r(44, 2, 8, 4, '#e0b84a'),
      r(78, 26, 6, 56, '#6b4a2a'),
      r(70, 12, 20, 18, '#5fc7e8'),
    ],
  },
  'dragonkin-warrior': {
    label: 'Dragonkin Warrior',
    rects: [
      ...baseBody('#4a8c52', '#7a8290', '#8c2a2a', '#2a2a2a'),
      r(30, 4, 8, 16, '#caa46a'),
      r(58, 4, 8, 16, '#caa46a'),
      r(14, 48, 14, 18, '#9aa2b0'),
      r(68, 48, 14, 18, '#9aa2b0'),
      r(78, 20, 8, 56, '#c8ccd0'),
      r(76, 76, 12, 12, '#6b4a2a'),
    ],
  },
  'dwarf-druid': {
    label: 'Dwarf Druid',
    rects: [
      r(32, 16, 32, 32, '#d99b6c'),
      r(40, 32, 6, 8, '#1a1a1a'),
      r(50, 32, 6, 8, '#1a1a1a'),
      r(16, 52, 64, 36, '#3f6b3f'),
      r(16, 76, 64, 8, '#7a5230'),
      r(26, 88, 22, 28, '#3f6b3f'),
      r(48, 88, 22, 28, '#3f6b3f'),
      r(26, 108, 22, 8, '#2e2015'),
      r(48, 108, 22, 8, '#2e2015'),
      r(26, 38, 44, 22, '#d6d6d6'),
      r(78, 24, 6, 56, '#6b4a2a'),
      r(68, 14, 24, 16, '#4f8f4f'),
    ],
  },
  'elf-rogue': {
    label: 'Elf Rogue',
    rects: [
      ...baseBody('#f0d9b5', '#33322e', '#242320', '#1a1a1a'),
      r(20, 24, 8, 10, '#f0d9b5'),
      r(68, 24, 8, 10, '#f0d9b5'),
      r(26, 10, 44, 30, '#242320'),
      r(40, 26, 16, 16, '#f0d9b5'),
      r(42, 32, 5, 6, '#1a1a1a'),
      r(49, 32, 5, 6, '#1a1a1a'),
      r(10, 64, 6, 20, '#c8ccd0'),
      r(80, 64, 6, 20, '#c8ccd0'),
      r(10, 82, 6, 6, '#6b4a2a'),
      r(80, 82, 6, 6, '#6b4a2a'),
    ],
  },
  'halfling-cleric': {
    label: 'Halfling Cleric',
    small: true,
    rects: [
      ...baseBody('#e0b285', '#ede3c8', '#d4af37', '#7a6a4a'),
      r(36, 14, 24, 8, '#8a6a3a'),
      r(72, 46, 18, 6, '#d4af37'),
      r(79, 38, 4, 20, '#d4af37'),
    ],
  },
}

export const ARCHETYPE_KEYS = Object.keys(ARCHETYPES)

export const STATE_COLOR: Record<State, string> = {
  working: '#3ec27a',
  waiting: '#f0a83c',
  blocked: '#e5473b',
  done: '#5b9bd8',
  failed: '#a41f2f',
  idle: '#8a8f85',
}

export const STATE_LABEL: Record<State, string> = {
  working: 'Working',
  waiting: 'Waiting',
  blocked: 'Blocked',
  done: 'Done',
  failed: 'Failed',
  idle: 'Idle',
}

export const PROVIDER_RUNE: Record<Provider, { shape: 'leaf' | 'diamond' | 'triangle'; color: string }> = {
  'claude-code': { shape: 'leaf', color: '#d98a3d' },
  codex: { shape: 'diamond', color: '#6fa8dc' },
  cursor: { shape: 'triangle', color: '#c77dd8' },
}

export const PROVIDER_LABEL: Record<Provider, string> = {
  'claude-code': 'Claude Code',
  codex: 'Codex',
  cursor: 'Cursor',
}

export const NEEDS_ATTENTION: State[] = ['waiting', 'blocked', 'failed']

// Deterministic pool pick: an archetype stays stable for a given session id
// across re-renders, but varies session to session, even for the same
// provider — several agents from the same vendor don't read as clones.
function hash(seed: string): number {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  return h
}

export function archetypeFor(sessionId: string): string {
  return ARCHETYPE_KEYS[hash(sessionId) % ARCHETYPE_KEYS.length]
}

// No token/context field exists on Session yet (see types.ts), so HP/mana
// are seeded placeholders — stable per session id, not real telemetry.
export function seededPercent(sessionId: string, salt: string): number {
  const h = hash(sessionId + salt)
  return 15 + (h % 86)
}

// 5x5 block-icon glyphs (2-unit cells on a 10x10 grid) drawn in the same
// pixel grammar as the party sprites, replacing unicode/emoji status marks.
export const STATUS_ICON_CELLS: Partial<Record<State, [number, number][]>> = {
  waiting: [
    [1, 0], [2, 0], [3, 0],
    [0, 1], [4, 1],
    [3, 2], [2, 3],
    [2, 4],
  ],
  blocked: [
    [2, 0], [2, 1], [2, 2],
    [2, 4],
  ],
  done: [
    [0, 2], [1, 3], [2, 4], [3, 2], [4, 0],
  ],
  failed: [
    [0, 0], [1, 1], [2, 2], [3, 3], [4, 4],
    [4, 0], [3, 1], [1, 3], [0, 4],
  ],
  idle: [
    [0, 0], [1, 0], [2, 0], [3, 0],
    [2, 1], [1, 2],
    [0, 3], [1, 3], [2, 3], [3, 3],
  ],
}

export const STATUS_ICON_BG: Partial<Record<State, string>> = {
  waiting: '#f0a83c',
  blocked: '#e5473b',
  done: '#5b9bd8',
  failed: '#7a1620',
  idle: '#8a8f85',
}
