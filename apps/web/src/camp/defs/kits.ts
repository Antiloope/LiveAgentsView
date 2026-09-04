import type { KitDef } from './types'

export interface KitColors {
  skin: string
  skinShadow: string
  cloth: string
  clothShadow: string
  clothHighlight: string
  metal?: string
  accent?: string
  small?: boolean
}

export const KIT_COLORS: Record<string, KitColors> = {
  'human-mage': {
    skin: '#e8b088',
    skinShadow: '#c08050',
    cloth: '#2f4d7a',
    clothShadow: '#1f2f52',
    clothHighlight: '#4a6fa0',
    accent: '#c5a059',
    metal: '#5fc7e8',
  },
  'dragonkin-warrior': {
    skin: '#4a8c52',
    skinShadow: '#2e5c34',
    cloth: '#7a8290',
    clothShadow: '#4a5058',
    clothHighlight: '#9aa2b0',
    accent: '#8c2a2a',
    metal: '#c8ccd0',
  },
  'dwarf-druid': {
    skin: '#d99b6c',
    skinShadow: '#b07848',
    cloth: '#3f6b3f',
    clothShadow: '#2a4a2a',
    clothHighlight: '#5a8a5a',
    accent: '#7a5230',
    metal: '#d6d6d6',
  },
  'elf-rogue': {
    skin: '#f0d9b5',
    skinShadow: '#d0b090',
    cloth: '#33322e',
    clothShadow: '#1a1a18',
    clothHighlight: '#4a4840',
    accent: '#242320',
    metal: '#c8ccd0',
  },
  'halfling-cleric': {
    skin: '#e0b285',
    skinShadow: '#c09060',
    cloth: '#ede3c8',
    clothShadow: '#c8b890',
    clothHighlight: '#fff8e0',
    accent: '#d4af37',
    metal: '#d4af37',
    small: true,
  },
}

export const KITS: Record<string, KitDef> = {
  'human-mage': { id: 'human-mage', label: 'Human Mage', layers: ['shadow', 'body', 'clothes', 'head', 'weapon'], animations: { idle: 1, walk: 4 } },
  'dragonkin-warrior': { id: 'dragonkin-warrior', label: 'Dragonkin Warrior', layers: ['shadow', 'body', 'clothes', 'head', 'weapon'], animations: { idle: 1, walk: 4 } },
  'dwarf-druid': { id: 'dwarf-druid', label: 'Dwarf Druid', layers: ['shadow', 'body', 'clothes', 'head', 'weapon'], animations: { idle: 1, walk: 4 } },
  'elf-rogue': { id: 'elf-rogue', label: 'Elf Rogue', layers: ['shadow', 'body', 'clothes', 'head', 'weapon'], animations: { idle: 1, walk: 4 } },
  'halfling-cleric': { id: 'halfling-cleric', label: 'Halfling Cleric', layers: ['shadow', 'body', 'clothes', 'head', 'weapon'], animations: { idle: 1, walk: 4 } },
}

export const KIT_IDS = Object.keys(KITS)
