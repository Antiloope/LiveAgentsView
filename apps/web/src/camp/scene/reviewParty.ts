import type { Character } from '../../types'

/** Synthetic party for finish-review captures (`?review=1`). Labeled demo, not real sessions. */
export const REVIEW_PARTY: Character[] = [
  {
    id: 'review-cursor-mage',
    session_id: 'review-1',
    race: 'cursor',
    class: 'opus',
    activity: 'ready',
    presence: 'awake',
    unread: false,
    territory: { mode: 'own', path: '/demo/camp', source: '', branch: 'main' },
    repo: 'demo-camp',
    archived: false,
    last_message: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'review-claude-warrior',
    session_id: 'review-2',
    race: 'claude-code',
    class: 'sonnet',
    activity: 'waiting',
    presence: 'awake',
    unread: true,
    territory: { mode: 'own', path: '/demo/quest-a', source: '', branch: 'feat' },
    repo: 'demo-quest-a',
    archived: false,
    last_message: 'Needs approval',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:01Z',
  },
  {
    id: 'review-claude-rogue',
    session_id: 'review-3',
    race: 'claude-code',
    class: 'haiku',
    activity: 'failed',
    presence: 'awake',
    unread: true,
    territory: { mode: 'shared', path: '/demo/quest-b', source: '', branch: 'fix' },
    repo: 'demo-quest-b',
    archived: false,
    last_message: 'Tool error',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:02Z',
  },
  {
    id: 'review-cursor-druid',
    session_id: 'review-4',
    race: 'cursor',
    class: 'composer',
    activity: 'ready',
    presence: 'awake',
    unread: false,
    territory: { mode: 'own', path: '/demo/out', source: '', branch: 'wip' },
    repo: 'demo-out',
    archived: false,
    last_message: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:03Z',
  },
]

/** Same party but one agent out on quest — for dual review captures. */
export const REVIEW_PARTY_WITH_QUEST: Character[] = REVIEW_PARTY.map((c, i) =>
  i === 3 ? { ...c, activity: 'working' as const, repo: 'demo-out' } : c,
)

export function reviewModeEnabled(): boolean {
  if (typeof window === 'undefined') return false
  return new URLSearchParams(window.location.search).has('review')
}

export function reviewPartyForQuery(): Character[] {
  if (typeof window === 'undefined') return REVIEW_PARTY
  const q = new URLSearchParams(window.location.search)
  if (!q.has('review')) return REVIEW_PARTY
  if (q.get('review') === 'quest') return REVIEW_PARTY_WITH_QUEST
  return REVIEW_PARTY
}
