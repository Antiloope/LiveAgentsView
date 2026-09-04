import type { PixelBuffer } from '../../bake/buffer'
import { bodyLayout, outlineBox, rect, type RoleIndices } from './drawHelpers'

/** Torso robe / armor base. */
export function paintClothes(buf: PixelBuffer, r: RoleIndices, small: boolean) {
  const { torsoY } = bodyLayout(small)
  outlineBox(buf, 14, torsoY, 20, 18, r.c, r.o)
  rect(buf, 15, torsoY + 1, 18, 4, r.ch)
  rect(buf, 14, torsoY + 14, 20, 3, r.a)
}
