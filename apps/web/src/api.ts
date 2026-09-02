import type { Session } from './types'

export async function fetchSessions(): Promise<Session[]> {
  const res = await fetch('/api/sessions')
  if (!res.ok) throw new Error(`GET /api/sessions: ${res.status}`)
  const data = (await res.json()) as Session[] | null
  return data ?? []
}

// openTerminal asks the daemon to spawn a terminal at cwd. Only works when
// the daemon runs natively on the host, not inside a container.
export async function openTerminal(cwd: string): Promise<void> {
  const res = await fetch('/api/open-terminal', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ cwd }),
  })
  if (!res.ok) throw new Error(`POST /api/open-terminal: ${res.status}`)
}

// subscribeToSessions opens the SSE stream and calls onUpdate for every
// session upsert the daemon broadcasts. Returns an unsubscribe function.
export function subscribeToSessions(onUpdate: (session: Session) => void): () => void {
  const source = new EventSource('/api/events/stream')
  source.onmessage = (event) => {
    try {
      onUpdate(JSON.parse(event.data) as Session)
    } catch {
      // ignore malformed/non-JSON frames (e.g. the initial ": connected" comment)
    }
  }
  return () => source.close()
}
