import { Texture } from 'pixi.js'
import type { Palette } from '../defs/types'
import { PixelBuffer } from './buffer'
import { bufferToCanvas } from './sheet'

/** Bake a pixel buffer into a Pixi texture with nearest-neighbor sampling. */
export function bakeTexture(buf: PixelBuffer, palette: Palette, scale = 1): Texture {
  const canvas = bufferToCanvas(buf, palette, scale)
  const tex = Texture.from(canvas)
  tex.source.scaleMode = 'nearest'
  return tex
}

export { PixelBuffer } from './buffer'
export { bufferToCanvas, bufferToImageData } from './sheet'
