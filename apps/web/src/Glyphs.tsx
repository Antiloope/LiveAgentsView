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

// Drawn pixel-block status glyph (question mark, exclamation, check, cross,
// zzz) instead of a unicode/emoji stand-in — same 2-unit-cell grammar as
// the party sprites so it reads as this world's own icon system.
export function StatusIcon({ state, className }: { state: State; className?: string }) {
  const cells = STATUS_ICON_CELLS[state]
  if (!cells) return null
  const bg = STATUS_ICON_BG[state]
  const ink = state === 'waiting' ? '#2a1f14' : '#fff8e8'
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
