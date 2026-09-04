import type { Palette } from '../defs/types'
import { PixelBuffer } from './buffer'

function parseHex(hex: string): [number, number, number, number] {
  const h = hex.replace('#', '')
  if (h.length === 8) {
    return [
      parseInt(h.slice(0, 2), 16),
      parseInt(h.slice(2, 4), 16),
      parseInt(h.slice(4, 6), 16),
      parseInt(h.slice(6, 8), 16),
    ]
  }
  if (h.length === 6) {
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16), 255]
  }
  return [0, 0, 0, 0]
}

/** Buffer + palette → ImageData (index 0 = fully transparent). */
export function bufferToImageData(buf: PixelBuffer, palette: Palette): ImageData {
  const rgba = new Uint8ClampedArray(buf.w * buf.h * 4)
  for (let i = 0; i < buf.cells.length; i++) {
    const idx = buf.cells[i]
    const o = i * 4
    if (idx === 0) {
      rgba[o + 3] = 0
      continue
    }
    const hex = palette.colors[idx] ?? '#000000'
    const [r, g, b, a] = parseHex(hex)
    rgba[o] = r
    rgba[o + 1] = g
    rgba[o + 2] = b
    rgba[o + 3] = a
  }
  return new ImageData(rgba, buf.w, buf.h)
}

/** Paint a buffer to a canvas at integer scale (nearest via imageSmoothingEnabled=false). */
export function bufferToCanvas(buf: PixelBuffer, palette: Palette, scale = 1): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = buf.w * scale
  canvas.height = buf.h * scale
  const ctx = canvas.getContext('2d')!
  ctx.imageSmoothingEnabled = false
  const img = bufferToImageData(buf, palette)
  if (scale === 1) {
    ctx.putImageData(img, 0, 0)
    return canvas
  }
  const tmp = document.createElement('canvas')
  tmp.width = buf.w
  tmp.height = buf.h
  tmp.getContext('2d')!.putImageData(img, 0, 0)
  ctx.drawImage(tmp, 0, 0, canvas.width, canvas.height)
  return canvas
}
