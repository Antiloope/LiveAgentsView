import type { Graphics } from 'pixi.js'

type Px = (g: Graphics, x: number, y: number, w: number, h: number, color: number) => void

/** Campfire ring + animated flame. */
export function drawFire(g: Graphics, cx: number, cy: number, t: number, px: Px) {
  g.clear()
  for (let y = -2; y <= 2; y++) {
    for (let x = -4; x <= 4; x++) {
      if (Math.abs(x) + Math.abs(y) > 5) continue
      px(g, cx + x * 10, cy + 10 + y * 8, 10, 8, 0x4a4a48)
    }
  }
  px(g, cx - 20, cy + 6, 16, 8, 0x5a3820)
  px(g, cx + 4, cy + 8, 16, 8, 0x6a4828)
  const flicker = Math.floor(Math.sin(t * 10) * 2)
  px(g, cx - 6, cy - 4 + flicker, 12, 14, 0xe84820)
  px(g, cx - 4, cy - 12 + flicker, 8, 12, 0xf07830)
  px(g, cx - 2, cy - 18 + flicker, 4, 10, 0xf8d060)
  px(g, cx - 1, cy - 22 + flicker, 2, 6, 0xfff8d0)
}
