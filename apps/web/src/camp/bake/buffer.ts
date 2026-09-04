/** Indexed pixel buffer used by kit/tile bakers. */

export class PixelBuffer {
  readonly w: number
  readonly h: number
  cells: Uint8Array

  constructor(w: number, h: number) {
    this.w = w
    this.h = h
    this.cells = new Uint8Array(w * h)
  }

  index(x: number, y: number): number {
    return y * this.w + x
  }

  inBounds(x: number, y: number): boolean {
    return x >= 0 && y >= 0 && x < this.w && y < this.h
  }

  set(x: number, y: number, colorIndex: number): void {
    if (this.inBounds(x, y) && colorIndex > 0) this.cells[this.index(x, y)] = colorIndex
  }

  get(x: number, y: number): number {
    return this.inBounds(x, y) ? this.cells[this.index(x, y)] : 0
  }

  fill(x: number, y: number, w: number, h: number, colorIndex: number): void {
    for (let yy = y; yy < y + h; yy++) {
      for (let xx = x; xx < x + w; xx++) this.set(xx, yy, colorIndex)
    }
  }

  clear(): void {
    this.cells.fill(0)
  }

  clone(): PixelBuffer {
    const next = new PixelBuffer(this.w, this.h)
    next.cells.set(this.cells)
    return next
  }
}
