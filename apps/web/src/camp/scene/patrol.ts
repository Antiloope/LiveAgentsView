/** Walk / idle patrol brain for camp figures. Speeds are scene-unit/sec. */

export type PatrolMode = 'walk' | 'idle'

export interface Vec2 {
  x: number
  y: number
}

export interface PatrolBrain {
  mode: PatrolMode
  timer: number
  speed: number
  walkCadence: number
  phase: number
  pos: Vec2
  target: Vec2
  waypoints: Vec2[]
  wp: number
  facing: 1 | -1
}

function hash(seed: string): number {
  let h = 2166136261
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

function rand01(h: number, salt: number): number {
  const x = Math.sin(h * 0.0001 + salt * 12.9898) * 43758.5453
  return x - Math.floor(x)
}

function idleDuration(h: number, salt: number): number {
  return 0.9 + rand01(h, salt) * 2.4
}

function nextIdle(brain: PatrolBrain, h: number, salt: number): void {
  brain.mode = 'idle'
  brain.timer = idleDuration(h, salt)
}

function nextWalk(brain: PatrolBrain): void {
  brain.mode = 'walk'
  brain.wp = (brain.wp + 1) % brain.waypoints.length
  brain.target = { ...brain.waypoints[brain.wp] }
  const dx = brain.target.x - brain.pos.x
  if (Math.abs(dx) > 2) brain.facing = dx >= 0 ? 1 : -1
}

export function createPatrolBrain(id: string, waypoints: Vec2[], startIndex = 0): PatrolBrain {
  const h = hash(id)
  const wp = startIndex % waypoints.length
  const brain: PatrolBrain = {
    mode: 'idle',
    timer: idleDuration(h, 1) * (0.3 + rand01(h, 2)),
    speed: 22 + (h % 48),
    walkCadence: 5.5 + rand01(h, 3) * 4.5,
    phase: rand01(h, 4) * Math.PI * 2,
    pos: { ...waypoints[wp] },
    target: { ...waypoints[(wp + 1) % waypoints.length] },
    waypoints,
    wp,
    facing: 1,
  }
  if (rand01(h, 5) > 0.45) nextWalk(brain)
  return brain
}

export function stepPatrol(brain: PatrolBrain, dt: number, id: string): void {
  const h = hash(id)
  brain.timer -= dt

  if (brain.mode === 'idle') {
    if (brain.timer <= 0) nextWalk(brain)
    return
  }

  const dx = brain.target.x - brain.pos.x
  const dy = brain.target.y - brain.pos.y
  const dist = Math.hypot(dx, dy)
  if (dist < 4) {
    brain.pos.x = brain.target.x
    brain.pos.y = brain.target.y
    nextIdle(brain, h, (brain.wp + 7) * 3)
    return
  }

  if (Math.abs(dx) > 1) brain.facing = dx >= 0 ? 1 : -1
  const step = Math.min(dist, brain.speed * dt)
  brain.pos.x += (dx / dist) * step
  brain.pos.y += (dy / dist) * step
  brain.phase += dt * brain.walkCadence
}

export function walkSwing(phase: number): number {
  return Math.sin(phase)
}

export function pixelWalkFrame(phase: number, walking: boolean): number {
  if (!walking) return 0
  const cycle = ((phase % (Math.PI * 2)) + Math.PI * 2) % (Math.PI * 2)
  const t = cycle / (Math.PI * 2)
  if (t < 0.25) return 1
  if (t < 0.5) return 2
  if (t < 0.75) return 3
  return 2
}
