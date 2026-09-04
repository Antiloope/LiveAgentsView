import type { AnimId, KitLayerId, Palette } from './types'
import { CHAR_SIZE } from './types'
import { PixelBuffer } from '../bake/buffer'
import { CAMP_PALETTE, withKitColors } from './palette'
import { KIT_COLORS, KITS } from './kits'
import { paintBody, paintClothes, paintShadow, roles } from './parts'
import { ITEMS, resolveItems, type ItemId } from './items'

export interface AssembleOpts {
  kitId: string
  /** Override default race/class loadout. */
  items?: ItemId[]
  walkFrame?: number
  anim?: AnimId
}

/**
 * Draw kit layers in order: shadow → body → clothes (+armor items) →
 * head items → weapon items → fx.
 */
export function assembleKit(opts: AssembleOpts): { buf: PixelBuffer; palette: Palette } {
  const { kitId, items: itemOverride, walkFrame = 0 } = opts
  const colors = KIT_COLORS[kitId] ?? KIT_COLORS['human-mage']
  const palette = withKitColors(CAMP_PALETTE, colors)
  const buf = new PixelBuffer(CHAR_SIZE.w, CHAR_SIZE.h)
  const r = roles(palette)
  const small = Boolean(colors.small)
  const kit = KITS[kitId] ?? KITS['human-mage']
  const items = resolveItems(kitId, itemOverride).map((id) => ITEMS[id]).filter(Boolean)

  const paintItems = (slot: 'head' | 'armor' | 'weapon') => {
    for (const item of items) {
      if (item.slot === slot) item.paint(buf, r, small)
    }
  }

  for (const layer of kit.layers as KitLayerId[]) {
    switch (layer) {
      case 'shadow':
        paintShadow(buf, r, small)
        break
      case 'body':
        paintBody(buf, r, small, walkFrame)
        break
      case 'clothes':
        paintClothes(buf, r, small)
        paintItems('armor')
        break
      case 'head':
        paintItems('head')
        break
      case 'weapon':
        paintItems('weapon')
        break
      case 'fx':
        break
    }
  }

  return { buf, palette }
}

/** Bake a kit frame via layered assembleKit (Portrait + Pixi share this path). */
export function bakeKitBuffer(kitId: string, anim: AnimId = 'idle', walkFrame = 0, items?: ItemId[]) {
  return assembleKit({ kitId, anim, walkFrame, items })
}
