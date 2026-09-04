import type { Graphics } from 'pixi.js'

type Px = (g: Graphics, x: number, y: number, w: number, h: number, color: number) => void

/** Night sky, trees, ground, clearing, tents — Craft Pixel camp backdrop. */
export function drawBackdrop(g: Graphics, w: number, h: number, px: Px) {
  g.clear()
  const tile = 16
  for (let y = 0; y < h * 0.48; y += tile) {
    for (let x = 0; x < w; x += tile) {
      const band = y < h * 0.18 ? 0x060a16 : y < h * 0.32 ? 0x0c1428 : 0x152038
      px(g, x, y, tile, tile, band)
      if ((x * 13 + y * 7) % 97 < 3) px(g, x + 4, y + 5, 1, 1, 0xdce4f0)
      if ((x * 17 + y * 11) % 131 === 0) px(g, x + 8, y + 3, 2, 1, 0xffffff)
    }
  }
  const mx = w * 0.82
  const my = h * 0.1
  px(g, mx, my, 14, 14, 0xd8d4c0)
  px(g, mx + 2, my + 2, 10, 10, 0xf4f0e0)
  px(g, mx + 8, my + 5, 3, 3, 0xb8b4a0)

  const treeBase = h * 0.48
  for (let i = 0; i < Math.ceil(w / 22); i++) {
    const tx = i * 22 + (i % 2) * 5
    const th = 40 + (i % 5) * 12
    px(g, tx + 9, treeBase - 8, 4, 10, 0x2a1a10)
    px(g, tx + 2, treeBase - th, 18, 10, 0x0e1a12)
    px(g, tx + 4, treeBase - th + 10, 14, 10, 0x142418)
    px(g, tx + 6, treeBase - th + 20, 10, 12, 0x1a3020)
  }
  for (let y = h * 0.5; y < h; y += tile) {
    for (let x = 0; x < w; x += tile) {
      const parity = (Math.floor(x / tile) + Math.floor(y / tile)) % 2
      px(g, x, y, tile, tile, parity ? 0x152018 : 0x1a2618)
      if ((x + y) % 7 === 0) px(g, x + 3, y + 4, 2, 1, 0x2a3820)
    }
  }
  const cx = w * 0.5
  const cy = h * 0.72
  for (let y = -5; y <= 5; y++) {
    for (let x = -9; x <= 9; x++) {
      if ((x * x) / 81 + (y * y) / 25 > 1) continue
      const shade = Math.abs(x) + Math.abs(y) < 5 ? 0x4a3824 : 0x3a3020
      px(g, cx + x * tile * 0.65, cy + y * tile * 0.5, tile * 0.65, tile * 0.5, shade)
    }
  }
  const tent = (ox: number, left: number, right: number) => {
    for (let row = 0; row < 12; row++) {
      const half = 5 + row
      px(g, ox - half * 2.5, h * 0.42 + row * 5, half * 2.5, 5, left)
      px(g, ox, h * 0.42 + row * 5, half * 2.5, 5, right)
    }
    px(g, ox - 6, h * 0.42 + 28, 12, 32, 0x0a0810)
    px(g, ox - 1, h * 0.4, 3, 10, 0x6a4a28)
  }
  tent(w * 0.16, 0x1a3a68, 0x2a5088)
  tent(w * 0.84, 0x5a2830, 0x7a3848)
  px(g, w * 0.22, h * 0.5, 2, 10, 0x6a4a28)
  px(g, w * 0.21, h * 0.51, 6, 5, 0xf0c060)
}
