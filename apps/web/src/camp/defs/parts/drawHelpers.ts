import type { Palette } from '../types'
import { roleIndex } from '../palette'
import type { PixelBuffer } from '../../bake/buffer'

export type RoleIndices = {
  o: number
  sk: number
  ss: number
  c: number
  cs: number
  ch: number
  m: number
  w: number
  a: number
}

export function roles(p: Palette): RoleIndices {
  return {
    o: roleIndex(p, 'outline'),
    sk: roleIndex(p, 'skin'),
    ss: roleIndex(p, 'skinShadow'),
    c: roleIndex(p, 'cloth'),
    cs: roleIndex(p, 'clothShadow'),
    ch: roleIndex(p, 'clothHighlight'),
    m: roleIndex(p, 'metal'),
    w: roleIndex(p, 'wood'),
    a: roleIndex(p, 'accent'),
  }
}

export function rect(buf: PixelBuffer, x: number, y: number, w: number, h: number, i: number) {
  buf.fill(x, y, w, h, i)
}

export function outlineBox(buf: PixelBuffer, x: number, y: number, w: number, h: number, fill: number, o: number) {
  rect(buf, x, y, w, h, fill)
  for (let xx = x; xx < x + w; xx++) {
    buf.set(xx, y, o)
    buf.set(xx, y + h - 1, o)
  }
  for (let yy = y; yy < y + h; yy++) {
    buf.set(x, yy, o)
    buf.set(x + w - 1, yy, o)
  }
}

/** Walk leg offset: 0 idle, 1–3 stride. */
export function legOffset(frame: number): [number, number] {
  if (frame === 1) return [-1, 1]
  if (frame === 2) return [0, 0]
  if (frame === 3) return [1, -1]
  return [0, 0]
}

export function bodyLayout(small: boolean) {
  const y0 = small ? 8 : 0
  return {
    y0,
    headY: 6 + y0,
    torsoY: 22 + y0,
    legY: 40 + y0,
    footY: 54 + y0,
  }
}
