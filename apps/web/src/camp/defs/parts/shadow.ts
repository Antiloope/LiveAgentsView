import type { PixelBuffer } from '../../bake/buffer'
import { CHAR_SIZE } from '../types'
import { bodyLayout, rect, type RoleIndices } from './drawHelpers'

export function paintShadow(buf: PixelBuffer, r: RoleIndices, small: boolean) {
  const { footY } = bodyLayout(small)
  const fy = Math.min(CHAR_SIZE.h - 3, footY + 4)
  rect(buf, 14, fy, 20, 3, r.cs)
  rect(buf, 17, fy + 1, 14, 2, r.o)
}
