import { useEffect, useRef } from 'react'
import { Application, Container, Graphics, Sprite, Text, Texture } from 'pixi.js'
import type { Character } from '../../types'
import {
  ACTIVITY_COLOR,
  NEEDS_ATTENTION,
  RACE_LABEL,
  STATUS_ICON_CELLS,
  STATUS_ICON_BG,
  archetypeFor,
} from '../../sprites'
import { bakeKitBuffer } from '../defs/assembleKit'
import { bakeTexture } from '../bake'
import { pixelWalkFrame, createPatrolBrain, stepPatrol, type PatrolBrain } from './patrol'
import { drawBackdrop } from './backdrop'
import { drawFire } from './fire'

export interface CampFigure {
  character: Character
  calm?: boolean
}

interface Props {
  className?: string
  figures: CampFigure[]
  selectedId: string | null
  onSelect: (id: string) => void
}

interface FigureNode {
  root: Container
  sprite: Sprite
  label: Text
  ring: Graphics
  status: Graphics
  shadow: Graphics
  brain: PatrolBrain
  kitId: string
  textures: Texture[]
  calm: boolean
  activity: string
}

function activityCueColor(activity: string): number {
  const hex = (
    STATUS_ICON_BG[activity as keyof typeof STATUS_ICON_BG] ??
    ACTIVITY_COLOR[activity as keyof typeof ACTIVITY_COLOR] ??
    '#c5a059'
  ).replace('#', '')
  return Number.parseInt(hex, 16)
}

function buildKitTextures(kitId: string): Texture[] {
  return [0, 1, 2, 3].map((frame) => {
    const { buf, palette } = bakeKitBuffer(kitId, frame === 0 ? 'idle' : 'walk', frame)
    return bakeTexture(buf, palette, 3)
  })
}

function layoutWaypoints(
  index: number,
  total: number,
  calm: boolean,
  cx: number,
  cy: number,
  w: number,
): { x: number; y: number }[] {
  const rowY = calm ? cy - 28 : cy + 36
  const spread = Math.min(w * 0.5, 70 + total * 36)
  const t = total <= 1 ? 0.5 : index / (total - 1)
  const x = cx - spread / 2 + t * spread
  const amp = 14 + (index % 3) * 6
  return [
    { x, y: rowY },
    { x: x + amp, y: rowY - 4 },
    { x: x - amp * 0.35, y: rowY + 6 },
    { x, y: rowY + 2 },
  ]
}

function px(g: Graphics, x: number, y: number, w: number, h: number, color: number) {
  g.rect(Math.round(x), Math.round(y), Math.round(w), Math.round(h))
  g.fill(color)
}

function drawStatusGlyph(g: Graphics, activity: string, ox: number, oy: number) {
  const cells = STATUS_ICON_CELLS[activity as keyof typeof STATUS_ICON_CELLS]
  const bg = activityCueColor(activity)
  const cell = 3
  const size = 5 * cell + 4
  px(g, ox, oy, size, size, 0x1a1008)
  px(g, ox + 1, oy + 1, size - 2, size - 2, bg)
  if (!cells) return
  for (const [cx, cy] of cells) {
    px(g, ox + 2 + cx * cell, oy + 2 + cy * cell, cell, cell, 0xfbe9d4)
  }
}

/** Pixi camp: night craft-pixel clearing, hard fire, procedural party kits with patrol. */
export default function CampCanvas({ className, figures, selectedId, onSelect }: Props) {
  const hostRef = useRef<HTMLDivElement>(null)
  const onSelectRef = useRef(onSelect)
  onSelectRef.current = onSelect
  const selectedRef = useRef(selectedId)
  selectedRef.current = selectedId
  const figuresRef = useRef(figures)
  figuresRef.current = figures
  const syncRef = useRef<(() => void) | null>(null)

  useEffect(() => {
    const host = hostRef.current
    if (!host) return
    let destroyed = false
    let app: Application | null = null
    let partyLayer: Container | null = null
    let nodes = new Map<string, FigureNode>()
    let raf = 0
    let last = performance.now()
    const textureCache = new Map<string, Texture[]>()

    const getTextures = (kitId: string) => {
      let t = textureCache.get(kitId)
      if (!t) {
        t = buildKitTextures(kitId)
        textureCache.set(kitId, t)
      }
      return t
    }

    const syncFigures = (w: number, h: number) => {
      if (!partyLayer || !app) return
      const list = figuresRef.current
      const cx = w * 0.5
      const cy = h * 0.7
      const calm = list.filter((f) => f.calm)
      const urgent = list.filter((f) => !f.calm)
      const seen = new Set<string>()

      const place = (fig: CampFigure, index: number, row: CampFigure[], isCalm: boolean) => {
        const { character } = fig
        seen.add(character.id)
        const kitId = archetypeFor(character.id, character.race)
        let node = nodes.get(character.id)
        const waypoints = layoutWaypoints(index, row.length, isCalm, cx, cy, w)
        if (!node) {
          const textures = getTextures(kitId)
          const root = new Container()
          root.eventMode = 'static'
          root.cursor = 'pointer'
          root.on('pointertap', () => onSelectRef.current(character.id))
          const shadow = new Graphics()
          const ring = new Graphics()
          const sprite = new Sprite(textures[0])
          sprite.anchor.set(0.5, 1)
          const label = new Text({
            text: RACE_LABEL[character.race],
            style: {
              fontFamily: 'Pixelify Sans, Cinzel Decorative, sans-serif',
              fontSize: 12,
              fill: 0xf0e8d0,
              stroke: { color: 0x0a1020, width: 3 },
            },
          })
          label.anchor.set(0.5, 1)
          label.y = -sprite.height - 6
          const status = new Graphics()
          root.addChild(shadow, ring, sprite, status, label)
          partyLayer!.addChild(root)
          const brain = createPatrolBrain(character.id, waypoints, index)
          node = {
            root,
            sprite,
            label,
            ring,
            status,
            shadow,
            brain,
            kitId,
            textures,
            calm: isCalm,
            activity: character.activity,
          }
          nodes.set(character.id, node)
        } else {
          node.brain.waypoints = waypoints
          node.calm = isCalm
          node.activity = character.activity
          if (node.kitId !== kitId) {
            node.kitId = kitId
            node.textures = getTextures(kitId)
            node.sprite.texture = node.textures[0]
          }
        }
      }

      calm.forEach((f, i) => place(f, i, calm, true))
      urgent.forEach((f, i) => place(f, i, urgent, false))

      for (const [id, node] of nodes) {
        if (!seen.has(id)) {
          partyLayer.removeChild(node.root)
          node.root.destroy({ children: true })
          nodes.delete(id)
        }
      }
    }

    const tick = (now: number) => {
      if (destroyed || !app) return
      const dt = Math.min(0.05, (now - last) / 1000)
      last = now
      const w = app.screen.width
      const h = app.screen.height
      const t = now / 1000
      const fireX = w * 0.5
      const fireY = h * 0.66

      const fire = app.stage.getChildByLabel('fire', true) as Graphics | null
      if (fire) drawFire(fire, fireX, fireY, t, px)

      for (const [id, node] of nodes) {
        stepPatrol(node.brain, dt, id)
        const walking = node.brain.mode === 'walk'
        const frame = pixelWalkFrame(node.brain.phase, walking)
        node.sprite.texture = node.textures[frame] ?? node.textures[0]
        node.sprite.scale.x = node.brain.facing
        node.root.x = node.brain.pos.x
        node.root.y = node.brain.pos.y
        node.root.zIndex = node.brain.pos.y
        node.root.alpha = node.calm ? 0.82 : 1

        const dx = node.root.x - fireX
        const dy = node.root.y - fireY
        const dist = Math.hypot(dx, dy) || 1
        const sx = (dx / dist) * 10
        const sy = (dy / dist) * 4 + 2
        node.shadow.clear()
        px(node.shadow, sx - 14, sy, 28, 6, 0x0a0806)

        const selected = selectedRef.current === id
        node.ring.clear()
        if (selected) {
          px(node.ring, -22, 0, 44, 3, 0xe8c878)
          px(node.ring, -20, 3, 40, 2, 0xc5a059)
        } else if (NEEDS_ATTENTION.includes(node.activity as 'waiting' | 'failed')) {
          px(node.ring, -18, 2, 36, 3, activityCueColor(node.activity))
        }

        node.status.clear()
        if (node.activity === 'waiting' || node.activity === 'failed') {
          drawStatusGlyph(node.status, node.activity, 14, -node.sprite.height - 2)
        }
      }

      partyLayer?.sortChildren()
      raf = requestAnimationFrame(tick)
    }

    const boot = async () => {
      const next = new Application()
      await next.init({
        backgroundAlpha: 0,
        antialias: false,
        resolution: Math.min(window.devicePixelRatio || 1, 2),
        autoDensity: true,
        resizeTo: host,
        preference: 'webgl',
        powerPreference: 'low-power',
      })
      if (destroyed) {
        next.destroy(true)
        return
      }
      app = next
      host.appendChild(next.canvas)
      next.canvas.style.display = 'block'
      next.canvas.style.width = '100%'
      next.canvas.style.height = '100%'
      next.stage.sortableChildren = true

      const backdrop = new Graphics()
      backdrop.label = 'backdrop'
      const fire = new Graphics()
      fire.label = 'fire'
      partyLayer = new Container()
      partyLayer.label = 'party'
      partyLayer.sortableChildren = true
      next.stage.addChild(backdrop, fire, partyLayer)

      const layout = () => {
        if (!app) return
        drawBackdrop(backdrop, app.screen.width, app.screen.height, px)
        syncFigures(app.screen.width, app.screen.height)
      }
      layout()
      syncRef.current = layout
      next.renderer.on('resize', layout)
      raf = requestAnimationFrame(tick)
    }

    void boot()

    return () => {
      destroyed = true
      syncRef.current = null
      cancelAnimationFrame(raf)
      for (const node of nodes.values()) node.root.destroy({ children: true })
      nodes.clear()
      for (const textures of textureCache.values()) {
        for (const tex of textures) tex.destroy(true)
      }
      textureCache.clear()
      if (app) {
        if (host.contains(app.canvas)) host.removeChild(app.canvas)
        app.destroy(true, { children: true })
        app = null
      }
    }
  }, [])

  useEffect(() => {
    figuresRef.current = figures
    syncRef.current?.()
  }, [figures])

  return <div className={className} ref={hostRef} style={{ width: '100%', height: '100%', minHeight: 240 }} />
}
