import type { Character, CursorClassOption, PilotEvent, Race, TerritoryMode } from './types'

// The daemon rejects every state-changing request (never GET or SSE) that
// lacks this header. Its only job is to force a CORS preflight, which the
// daemon never answers affirmatively, so a cross-origin page can never set
// it and reach those routes.
const CLIENT_HEADER = 'X-LAV-Client'

export async function fetchCharacters(): Promise<Character[]> {
  const res = await fetch('/api/characters')
  if (!res.ok) throw new Error(`GET /api/characters: ${res.status}`)
  const data = (await res.json()) as Character[] | null
  return data ?? []
}

// subscribeToCharacters opens the SSE stream and calls onUpdate for every
// character upsert the daemon broadcasts. Returns an unsubscribe function.
export function subscribeToCharacters(onUpdate: (character: Character) => void): () => void {
  const source = new EventSource('/api/events/stream')
  source.onmessage = (event) => {
    try {
      onUpdate(JSON.parse(event.data) as Character)
    } catch {
      // ignore malformed/non-JSON frames (e.g. the initial ": connected" comment)
    }
  }
  return () => source.close()
}

// characterAction posts to a character action endpoint and surfaces the
// daemon's plain-text error body instead of just the status code.
async function characterAction(path: string, body?: unknown): Promise<Response> {
  const headers: Record<string, string> = { [CLIENT_HEADER]: '1' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  const res = await fetch(path, {
    method: 'POST',
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `${path}: ${res.status}`)
  }
  return res
}

export async function createCharacter(spec: {
  race: Race
  territoryMode: TerritoryMode
  cwd: string
  branch: string
  class: string
  prompt: string
}): Promise<Character> {
  const res = await characterAction('/api/characters', {
    race: spec.race,
    territory_mode: spec.territoryMode,
    cwd: spec.cwd,
    branch: spec.branch,
    class: spec.class,
    prompt: spec.prompt,
  })
  return (await res.json()) as Character
}

// pickDirectory opens the daemon's native macOS folder picker and returns
// the chosen absolute path, or null if the user cancelled the dialog.
export async function pickDirectory(): Promise<string | null> {
  const res = await fetch('/api/pick-directory', { method: 'POST', headers: { [CLIENT_HEADER]: '1' } })
  if (res.status === 204) return null
  if (!res.ok) throw new Error((await res.text().catch(() => '')) || `POST /api/pick-directory: ${res.status}`)
  const data = (await res.json()) as { path: string }
  return data.path
}

export async function fetchBranches(cwd: string): Promise<{ isRepo: boolean; current: string; branches: string[] }> {
  const res = await fetch(`/api/branches?cwd=${encodeURIComponent(cwd)}`)
  if (!res.ok) throw new Error(`GET /api/branches: ${res.status}`)
  const data = (await res.json()) as { is_repo: boolean; current: string; branches: string[] }
  return { isRepo: data.is_repo, current: data.current, branches: data.branches }
}

export async function fetchCursorClasses(): Promise<CursorClassOption[]> {
  const res = await fetch('/api/cursor-classes')
  if (!res.ok) throw new Error(`GET /api/cursor-classes: ${res.status}`)
  const data = (await res.json()) as CursorClassOption[] | null
  return data ?? []
}

export function sendMessage(id: string, text: string): Promise<void> {
  return characterAction(`/api/characters/${id}/message`, { text }).then(() => undefined)
}

export function interruptCharacter(id: string): Promise<void> {
  return characterAction(`/api/characters/${id}/interrupt`).then(() => undefined)
}

export function stopCharacter(id: string): Promise<void> {
  return characterAction(`/api/characters/${id}/stop`).then(() => undefined)
}

export async function archiveCharacter(id: string): Promise<Character> {
  const res = await characterAction(`/api/characters/${id}/archive`)
  return (await res.json()) as Character
}

export async function unarchiveCharacter(id: string): Promise<Character> {
  const res = await characterAction(`/api/characters/${id}/unarchive`)
  return (await res.json()) as Character
}

// dismissCharacter removes a character for good. worktreeLeftAt is
// non-empty when an own-territory worktree had uncommitted changes and was
// left in place rather than discarded.
export async function dismissCharacter(id: string): Promise<{ worktreeLeftAt: string }> {
  const res = await characterAction(`/api/characters/${id}/dismiss`)
  const data = (await res.json()) as { worktree_left_at: string }
  return { worktreeLeftAt: data.worktree_left_at }
}

// markRead clears a character's unread mark — call this when the interface
// actually shows the user its transcript, the only thing that clears it.
export function markRead(id: string): Promise<void> {
  return characterAction(`/api/characters/${id}/read`).then(() => undefined)
}

export async function fetchEvents(id: string): Promise<PilotEvent[]> {
  const res = await fetch(`/api/characters/${id}/events`)
  if (!res.ok) throw new Error(`GET /api/characters/${id}/events: ${res.status}`)
  const data = (await res.json()) as PilotEvent[] | null
  return data ?? []
}

// subscribeToEvents opens one character's live transcript stream. Kept
// separate from subscribeToCharacters's global stream so a character's chat
// can be watched without touching the dashboard's own SSE connection.
export function subscribeToEvents(id: string, onEvent: (event: PilotEvent) => void): () => void {
  const source = new EventSource(`/api/characters/${id}/stream`)
  source.onmessage = (event) => {
    try {
      onEvent(JSON.parse(event.data) as PilotEvent)
    } catch {
      // ignore malformed/non-JSON frames (e.g. the initial ": connected" comment)
    }
  }
  return () => source.close()
}
