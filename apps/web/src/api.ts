import type { PilotEvent, PilotProvider, Session } from './types'

export async function fetchSessions(): Promise<Session[]> {
  const res = await fetch('/api/sessions')
  if (!res.ok) throw new Error(`GET /api/sessions: ${res.status}`)
  const data = (await res.json()) as Session[] | null
  return data ?? []
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

// pilotAction posts to a piloted-session action endpoint and surfaces the
// daemon's plain-text error body (e.g. "a turn is already in progress for
// this session") instead of just the status code.
async function pilotAction(path: string, body?: unknown): Promise<Response> {
  const res = await fetch(path, {
    method: 'POST',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `${path}: ${res.status}`)
  }
  return res
}

export async function launchPilotedSession(spec: {
  provider: PilotProvider
  cwd: string
  branch: string
  prompt: string
}): Promise<Session> {
  const res = await pilotAction('/api/piloted/sessions', spec)
  return (await res.json()) as Session
}

export function sendPilotMessage(id: string, text: string): Promise<void> {
  return pilotAction(`/api/piloted/sessions/${id}/message`, { text }).then(() => undefined)
}

export function resolvePilotPermission(id: string, requestId: string, approve: boolean): Promise<void> {
  return pilotAction(`/api/piloted/sessions/${id}/permission`, { request_id: requestId, approve }).then(
    () => undefined,
  )
}

export function interruptPilotedSession(id: string): Promise<void> {
  return pilotAction(`/api/piloted/sessions/${id}/interrupt`).then(() => undefined)
}

export function cancelPilotedSession(id: string): Promise<void> {
  return pilotAction(`/api/piloted/sessions/${id}/cancel`).then(() => undefined)
}

export async function resumePilotedSession(id: string): Promise<Session> {
  const res = await pilotAction(`/api/piloted/sessions/${id}/resume`)
  return (await res.json()) as Session
}

export async function fetchPilotEvents(id: string): Promise<PilotEvent[]> {
  const res = await fetch(`/api/piloted/sessions/${id}/events`)
  if (!res.ok) throw new Error(`GET /api/piloted/sessions/${id}/events: ${res.status}`)
  const data = (await res.json()) as PilotEvent[] | null
  return data ?? []
}

// subscribeToPilotEvents opens one session's live transcript stream. Kept
// separate from subscribeToSessions's global stream so a piloted session's
// chat can be watched without touching the dashboard's own SSE connection.
export function subscribeToPilotEvents(id: string, onEvent: (event: PilotEvent) => void): () => void {
  const source = new EventSource(`/api/piloted/sessions/${id}/stream`)
  source.onmessage = (event) => {
    try {
      onEvent(JSON.parse(event.data) as PilotEvent)
    } catch {
      // ignore malformed/non-JSON frames (e.g. the initial ": connected" comment)
    }
  }
  return () => source.close()
}
