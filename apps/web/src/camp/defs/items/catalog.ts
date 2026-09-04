import type { PixelBuffer } from '../../bake/buffer'
import { bodyLayout, outlineBox, rect, type RoleIndices } from '../parts/drawHelpers'

export type ItemSlot = 'head' | 'armor' | 'weapon'

export type ItemId =
  | 'hood'
  | 'cowl'
  | 'cap'
  | 'horns'
  | 'ears'
  | 'beard'
  | 'pauldron'
  | 'staff'
  | 'leaf-staff'
  | 'sword'
  | 'dagger'
  | 'mace'

export interface ItemDef {
  id: ItemId
  slot: ItemSlot
  paint: (buf: PixelBuffer, r: RoleIndices, small: boolean) => void
}

function paintHood(buf: PixelBuffer, r: RoleIndices, small: boolean) {
  const { y0 } = bodyLayout(small)
  outlineBox(buf, 14, 2 + y0, 20, 8, r.cs, r.o)
  rect(buf, 20, y0, 8, 6, r.cs)
  rect(buf, 22, 1 + y0, 4, 2, r.a)
}

function paintCowl(buf: PixelBuffer, r: RoleIndices, small: boolean) {
  const { y0 } = bodyLayout(small)
  outlineBox(buf, 14, 4 + y0, 20, 14, r.cs, r.o)
  rect(buf, 18, 12 + y0, 12, 8, r.sk)
  rect(buf, 20, 14 + y0, 3, 3, r.o)
  rect(buf, 25, 14 + y0, 3, 3, r.o)
}

function paintCap(buf: PixelBuffer, r: RoleIndices, small: boolean) {
  const { y0 } = bodyLayout(small)
  outlineBox(buf, 18, 10 + y0, 12, 5, r.w, r.o)
}

function paintHorns(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  rect(buf, 14, 2, 4, 10, r.a)
  rect(buf, 30, 2, 4, 10, r.a)
  buf.set(14, 2, r.o)
  buf.set(33, 2, r.o)
}

function paintEars(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  rect(buf, 12, 10, 4, 6, r.sk)
  rect(buf, 32, 10, 4, 6, r.sk)
  buf.set(12, 10, r.o)
  buf.set(35, 10, r.o)
}

function paintBeard(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  outlineBox(buf, 15, 16, 18, 12, r.m, r.o)
  rect(buf, 16, 17, 16, 10, r.m)
}

function paintPauldron(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  outlineBox(buf, 8, 22, 8, 10, r.m, r.o)
  outlineBox(buf, 32, 22, 8, 10, r.m, r.o)
}

function paintStaff(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  rect(buf, 36, 14, 3, 36, r.w)
  for (let y = 14; y < 50; y++) {
    buf.set(36, y, r.o)
    buf.set(38, y, r.o)
  }
  outlineBox(buf, 33, 8, 9, 9, r.m, r.o)
}

function paintLeafStaff(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  rect(buf, 37, 14, 3, 34, r.w)
  outlineBox(buf, 33, 8, 11, 8, r.ch, r.o)
}

function paintSword(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  rect(buf, 38, 12, 3, 32, r.m)
  outlineBox(buf, 36, 40, 7, 5, r.w, r.o)
}

function paintDagger(buf: PixelBuffer, r: RoleIndices, _small: boolean) {
  rect(buf, 8, 30, 3, 12, r.m)
  rect(buf, 37, 30, 3, 12, r.m)
}

function paintMace(buf: PixelBuffer, r: RoleIndices, small: boolean) {
  const { y0 } = bodyLayout(small)
  rect(buf, 34, 24 + y0, 3, 18, r.a)
  outlineBox(buf, 31, 20 + y0, 9, 6, r.m, r.o)
}

export const ITEMS: Record<ItemId, ItemDef> = {
  hood: { id: 'hood', slot: 'head', paint: paintHood },
  cowl: { id: 'cowl', slot: 'head', paint: paintCowl },
  cap: { id: 'cap', slot: 'head', paint: paintCap },
  horns: { id: 'horns', slot: 'head', paint: paintHorns },
  ears: { id: 'ears', slot: 'head', paint: paintEars },
  beard: { id: 'beard', slot: 'head', paint: paintBeard },
  pauldron: { id: 'pauldron', slot: 'armor', paint: paintPauldron },
  staff: { id: 'staff', slot: 'weapon', paint: paintStaff },
  'leaf-staff': { id: 'leaf-staff', slot: 'weapon', paint: paintLeafStaff },
  sword: { id: 'sword', slot: 'weapon', paint: paintSword },
  dagger: { id: 'dagger', slot: 'weapon', paint: paintDagger },
  mace: { id: 'mace', slot: 'weapon', paint: paintMace },
}

/** Default loadout per race–class kit id. */
export const KIT_DEFAULT_ITEMS: Record<string, ItemId[]> = {
  'human-mage': ['hood', 'staff'],
  'dragonkin-warrior': ['horns', 'pauldron', 'sword'],
  'dwarf-druid': ['beard', 'leaf-staff'],
  'elf-rogue': ['ears', 'cowl', 'dagger'],
  'halfling-cleric': ['cap', 'mace'],
}

export function resolveItems(kitId: string, items?: ItemId[]): ItemId[] {
  return items ?? KIT_DEFAULT_ITEMS[kitId] ?? KIT_DEFAULT_ITEMS['human-mage']
}
