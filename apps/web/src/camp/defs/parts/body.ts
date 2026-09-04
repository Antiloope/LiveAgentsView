import type { PixelBuffer } from '../../bake/buffer'
import { bodyLayout, legOffset, outlineBox, rect, type RoleIndices } from './drawHelpers'

/** Legs, feet, and bare head (face). Torso cloth is `clothes`. */
export function paintBody(buf: PixelBuffer, r: RoleIndices, small: boolean, walkFrame: number) {
  const [lOff, rOff] = legOffset(walkFrame)
  const { headY, legY, footY } = bodyLayout(small)

  outlineBox(buf, 16, legY + lOff, 7, 14, r.c, r.o)
  outlineBox(buf, 25, legY + rOff, 7, 14, r.c, r.o)
  rect(buf, 16, footY + lOff, 7, 4, r.w)
  rect(buf, 25, footY + rOff, 7, 4, r.w)
  for (let x = 16; x < 23; x++) buf.set(x, footY + lOff, r.o)
  for (let x = 25; x < 32; x++) buf.set(x, footY + rOff, r.o)

  outlineBox(buf, 16, headY, 16, 16, r.sk, r.o)
  rect(buf, 17, headY + 1, 3, 12, r.ss)
  rect(buf, 28, headY + 1, 3, 10, r.sk)
  rect(buf, 19, headY + 5, 4, 4, r.m)
  rect(buf, 25, headY + 5, 4, 4, r.m)
  rect(buf, 20, headY + 6, 2, 2, r.cs)
  rect(buf, 26, headY + 6, 2, 2, r.cs)
  buf.set(21, headY + 6, r.o)
  buf.set(27, headY + 6, r.o)
  buf.set(20, headY + 5, r.ch)
  buf.set(26, headY + 5, r.ch)
  rect(buf, 19, headY + 4, 4, 1, r.o)
  rect(buf, 25, headY + 4, 4, 1, r.o)
  rect(buf, 23, headY + 8, 2, 2, r.ss)
  buf.set(18, headY + 9, r.a)
  buf.set(29, headY + 9, r.a)
  rect(buf, 20, headY + 12, 8, 1, r.ss)
  rect(buf, 21, headY + 12, 6, 1, r.a)
  rect(buf, 18, headY + 13, 12, 2, r.ss)
}
