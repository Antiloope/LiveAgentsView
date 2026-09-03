import type { Provider, State } from './types'
import { PROVIDER_RUNE, STATUS_ICON_CELLS, STATUS_ICON_BG } from './sprites'

// Provider mark: one flat glyph per vendor, same stroke weight — echoes the
// party sprites' own pixel-block language rather than a stock icon font.
export function ProviderRune({ provider, className }: { provider: Provider; className?: string }) {
  const rune = PROVIDER_RUNE[provider]
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'provider-rune'} aria-hidden="true">
      {rune.shape === 'leaf' && (
        <path d="M8 1c3 3 5 6 5 9a5 5 0 1 1-10 0c0-3 2-6 5-9z" fill={rune.color} />
      )}
      {rune.shape === 'diamond' && <path d="M8 1 15 8 8 15 1 8z" fill={rune.color} />}
      {rune.shape === 'triangle' && <path d="M8 1 15 14 1 14z" fill={rune.color} />}
    </svg>
  )
}

// Recruit-panel icons — same flat pixel-block grammar as ProviderRune and
// StatusIcon (axis-aligned rects, one glyph per idea), not a unicode/emoji
// stand-in. Shared 16x16 grid so they drop into the same .rune sizing.
export function ShieldRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={4} y={2} width={8} height={2} fill="#b98a52" />
      <rect x={3} y={4} width={10} height={6} fill="#9aa2b0" />
      <rect x={4} y={10} width={8} height={3} fill="#9aa2b0" />
      <rect x={6} y={13} width={4} height={2} fill="#9aa2b0" />
      <rect x={7} y={4} width={2} height={9} fill="#8c2a2a" />
    </svg>
  )
}

export function HoodRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={6} y={1} width={4} height={2} fill="#33322e" />
      <rect x={4} y={3} width={8} height={3} fill="#33322e" />
      <rect x={3} y={6} width={10} height={5} fill="#33322e" />
      <rect x={6} y={9} width={2} height={1} fill="#f0d9b5" />
      <rect x={9} y={9} width={2} height={1} fill="#f0d9b5" />
      <rect x={2} y={11} width={12} height={3} fill="#242320" />
    </svg>
  )
}

export function SatchelRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={3} y={3} width={2} height={9} fill="#452712" />
      <rect x={5} y={6} width={6} height={7} fill="#7a5230" />
      <rect x={6} y={4} width={4} height={3} fill="#d4af37" />
      <rect x={7} y={8} width={2} height={2} fill="#452712" />
    </svg>
  )
}

export function MapRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={2} y={3} width={12} height={10} fill="#f2e2c4" />
      <rect x={6} y={3} width={1} height={10} fill="#452712" />
      <rect x={10} y={3} width={1} height={10} fill="#452712" />
      <rect x={4} y={8} width={2} height={1} fill="#2a1a0c" />
      <rect x={9} y={6} width={2} height={1} fill="#2a1a0c" />
      <rect x={7} y={9} width={2} height={2} fill="#e2703a" />
    </svg>
  )
}

export function SignpostRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={7} y={6} width={2} height={9} fill="#452712" />
      <rect x={1} y={3} width={2} height={3} fill="#452712" />
      <rect x={3} y={3} width={7} height={3} fill="#7a4a26" />
      <rect x={7} y={7} width={7} height={3} fill="#7a4a26" />
      <rect x={13} y={7} width={2} height={3} fill="#452712" />
      <rect x={4} y={14} width={8} height={1} fill="#152410" />
    </svg>
  )
}

export function SparkRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={7} y={1} width={2} height={4} fill="#2a1a0c" />
      <rect x={7} y={11} width={2} height={4} fill="#2a1a0c" />
      <rect x={1} y={7} width={4} height={2} fill="#2a1a0c" />
      <rect x={11} y={7} width={4} height={2} fill="#2a1a0c" />
      <rect x={7} y={7} width={2} height={2} fill="#fff8e8" />
    </svg>
  )
}

export function SearchRune({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className ?? 'rune'} shapeRendering="crispEdges" aria-hidden="true">
      <rect x={3} y={3} width={7} height={2} fill="#63492c" />
      <rect x={3} y={3} width={2} height={7} fill="#63492c" />
      <rect x={8} y={3} width={2} height={7} fill="#63492c" />
      <rect x={3} y={8} width={7} height={2} fill="#63492c" />
      <rect x={10} y={10} width={2} height={2} fill="#63492c" />
      <rect x={12} y={12} width={2} height={2} fill="#63492c" />
    </svg>
  )
}

// Drawn pixel-block status glyph (question mark, exclamation, check, cross,
// zzz) instead of a unicode/emoji stand-in — same 2-unit-cell grammar as
// the party sprites so it reads as this world's own icon system.
export function StatusIcon({ state, className }: { state: State; className?: string }) {
  const cells = STATUS_ICON_CELLS[state]
  if (!cells) return null
  const bg = STATUS_ICON_BG[state]
  const ink = state === 'waiting' ? '#2a1a0c' : '#fff8e8'
  return (
    <div className={className ?? 'status-icon'} style={{ background: bg }}>
      <svg viewBox="0 0 10 10">
        {cells.map(([col, row], i) => (
          <rect key={i} x={col * 2} y={row * 2} width={2} height={2} fill={ink} />
        ))}
      </svg>
    </div>
  )
}
