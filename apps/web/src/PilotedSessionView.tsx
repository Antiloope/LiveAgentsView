import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { PilotEvent, Session } from './types'
import {
  cancelPilotedSession,
  fetchPilotEvents,
  interruptPilotedSession,
  resolvePilotPermission,
  resumePilotedSession,
  sendPilotMessage,
  subscribeToPilotEvents,
} from './api'

const PROVIDER_LABEL: Record<string, string> = { 'claude-code': 'Claude Code', cursor: 'Cursor' }

interface Props {
  session: Session
  onClose: () => void
  onSessionUpdate: (session: Session) => void
}

export default function PilotedSessionView({ session, onClose, onSessionUpdate }: Props) {
  const [events, setEvents] = useState<PilotEvent[]>([])
  const [draft, setDraft] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setEvents([])
    fetchPilotEvents(session.id)
      .then(setEvents)
      .catch((err) => setError(String(err)))
    return subscribeToPilotEvents(session.id, (event) => {
      setEvents((prev) => [...prev, event])
    })
  }, [session.id])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'end' })
  }, [events.length])

  // A request_id shows its approve/deny controls until a matching
  // permission_resolved event for the same id arrives.
  const resolvedRequestIds = useMemo(() => {
    const set = new Set<string>()
    for (const e of events) {
      if (e.kind === 'permission_resolved' && e.request_id) set.add(e.request_id)
    }
    return set
  }, [events])

  const isCursor = session.provider === 'cursor'
  // Cursor: every message is its own one-shot process, so "can send" just
  // means "not currently mid-turn". Claude Code: the process stays attached
  // across turns, so sending only fails once it's gone (idle/failed).
  const canSend = isCursor ? session.state !== 'working' : session.state !== 'idle' && session.state !== 'failed'
  const canInterruptOrCancel = isCursor ? session.state === 'working' : canSend
  const canResume = !isCursor && (session.state === 'idle' || session.state === 'failed')

  const runAction = useCallback(
    (action: () => Promise<void | Session>) => {
      setBusy(true)
      setError(null)
      action()
        .then((updated) => {
          if (updated) onSessionUpdate(updated)
        })
        .catch((err) => setError(String(err)))
        .finally(() => setBusy(false))
    },
    [onSessionUpdate],
  )

  const send = useCallback(() => {
    const text = draft.trim()
    if (!text) return
    setDraft('')
    runAction(() => sendPilotMessage(session.id, text))
  }, [draft, runAction, session.id])

  return (
    <div className="pilot-view">
      <header className="pilot-header">
        <button type="button" className="pilot-back" onClick={onClose}>
          ← Back
        </button>
        <div className="pilot-title">
          <span className="provider">{PROVIDER_LABEL[session.provider] ?? session.provider}</span>
          <span className="repo">{session.repo || session.cwd}</span>
          {session.branch && <span className="branch">{session.branch}</span>}
          <span className={`state-pill state-${session.state}`}>{session.state}</span>
        </div>
        <div className="pilot-actions">
          {canResume && (
            <button type="button" disabled={busy} onClick={() => runAction(() => resumePilotedSession(session.id))}>
              Resume
            </button>
          )}
          {canInterruptOrCancel && (
            <>
              <button type="button" disabled={busy} onClick={() => runAction(() => interruptPilotedSession(session.id))}>
                Interrupt
              </button>
              <button
                type="button"
                className="danger"
                disabled={busy}
                onClick={() => runAction(() => cancelPilotedSession(session.id))}
              >
                Cancel
              </button>
            </>
          )}
        </div>
      </header>

      {error && <div className="banner banner-error">{error}</div>}
      {isCursor && (
        <div className="banner banner-note">
          Cursor piloted sessions auto-approve every tool call (no live permission gate) — see the session's transcript
          for what it did.
        </div>
      )}

      <div className="pilot-transcript">
        {events.map((event, i) => (
          <TranscriptEntry
            key={i}
            event={event}
            resolved={event.request_id ? resolvedRequestIds.has(event.request_id) : false}
            onApprove={(approve) =>
              runAction(() => resolvePilotPermission(session.id, event.request_id ?? '', approve))
            }
          />
        ))}
        <div ref={bottomRef} />
      </div>

      <form
        className="pilot-compose"
        onSubmit={(e) => {
          e.preventDefault()
          send()
        }}
      >
        <textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder={canSend ? 'Message this session…' : 'No live process — resume to keep chatting'}
          disabled={!canSend || busy}
          rows={2}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              send()
            }
          }}
        />
        <button type="submit" disabled={!canSend || busy || !draft.trim()}>
          Send
        </button>
      </form>
    </div>
  )
}

function TranscriptEntry({
  event,
  resolved,
  onApprove,
}: {
  event: PilotEvent
  resolved: boolean
  onApprove: (approve: boolean) => void
}) {
  switch (event.kind) {
    case 'user':
      return (
        <div className="bubble bubble-user">
          <span className="bubble-label">You</span>
          <p>{event.text}</p>
        </div>
      )
    case 'assistant':
      return (
        <div className="bubble bubble-assistant">
          <p>{event.text}</p>
        </div>
      )
    case 'tool_call':
      return (
        <div className="bubble bubble-tool">
          <span className="bubble-label">→ {event.tool_name || 'tool'}</span>
          {event.tool_input !== undefined && (
            <pre>{JSON.stringify(event.tool_input, null, 2).slice(0, 2000)}</pre>
          )}
        </div>
      )
    case 'permission_request':
      return (
        <div className="bubble bubble-permission">
          <span className="bubble-label">Permission requested: {event.tool_name || 'tool'}</span>
          {event.tool_input !== undefined && (
            <pre>{JSON.stringify(event.tool_input, null, 2).slice(0, 2000)}</pre>
          )}
          {resolved ? (
            <p className="hint">resolved</p>
          ) : (
            <div className="permission-actions">
              <button type="button" onClick={() => onApprove(true)}>
                Approve
              </button>
              <button type="button" className="danger" onClick={() => onApprove(false)}>
                Deny
              </button>
            </div>
          )}
        </div>
      )
    case 'permission_resolved':
      return (
        <div className="bubble bubble-system">
          <p>{event.approved ? 'Approved' : 'Denied'} the pending permission request.</p>
        </div>
      )
    case 'error':
      return (
        <div className="bubble bubble-error">
          <p>{event.text}</p>
        </div>
      )
    default:
      return (
        <div className="bubble bubble-system">
          <p>{event.text}</p>
        </div>
      )
  }
}
