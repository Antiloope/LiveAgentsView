import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { PilotEvent, Session } from './types'
import { PROVIDER_LABEL, STATE_COLOR, STATE_LABEL } from './sprites'
import Portrait from './Portrait'
import {
  archiveSession,
  cancelPilotedSession,
  fetchPilotEvents,
  interruptPilotedSession,
  resolvePilotPermission,
  resumePilotedSession,
  sendPilotMessage,
  subscribeToPilotEvents,
} from './api'

const DEFAULT_WIDTH = 460
const MIN_WIDTH = 320
const STORAGE_KEY = 'lav.drawerWidth'

interface Props {
  session: Session | null
  onClose: () => void
  onSessionUpdate: (session: Session) => void
}

export default function SessionDrawer({ session, onClose, onSessionUpdate }: Props) {
  const drawerRef = useRef<HTMLDivElement>(null)
  const [dragging, setDragging] = useState(false)
  const open = session !== null

  useEffect(() => {
    const saved = Number(localStorage.getItem(STORAGE_KEY))
    if (drawerRef.current && saved >= MIN_WIDTH) {
      drawerRef.current.style.width = `${saved}px`
    }
  }, [])

  const startDrag = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setDragging(true)
  }, [])

  useEffect(() => {
    if (!dragging) return
    const onMove = (e: MouseEvent) => {
      const w = Math.max(MIN_WIDTH, Math.min(window.innerWidth * 0.78, window.innerWidth - e.clientX))
      if (drawerRef.current) drawerRef.current.style.width = `${w}px`
    }
    const onUp = () => {
      setDragging(false)
      if (drawerRef.current) localStorage.setItem(STORAGE_KEY, drawerRef.current.style.width.replace('px', ''))
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
  }, [dragging])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  // Move focus into the drawer for keyboard users when it opens, so Tab
  // starts from its own controls instead of wherever the party click left it.
  useEffect(() => {
    if (open) drawerRef.current?.querySelector<HTMLElement>('.drawer-close')?.focus()
  }, [open, session?.id])

  return (
    <>
      {/* Decorative dim only — pointer-events:none so clicking another party
          member or quest token (still visible beside the drawer by design)
          switches the drawer's session directly instead of needing a close
          click first. Escape and the ✕ button remain the ways to close. */}
      <div className={`scrim${open ? ' open' : ''}`} />
      <div
        ref={drawerRef}
        className={`drawer${open ? ' open' : ''}`}
        style={{ width: DEFAULT_WIDTH }}
        role="dialog"
        aria-label={session ? `${PROVIDER_LABEL[session.provider]} session` : undefined}
        aria-hidden={!open}
      >
        <div className={`drawer-handle${dragging ? ' dragging' : ''}`} onMouseDown={startDrag} title="Drag to resize" />
        <div className="drawer-body">
          {session && <DrawerContent session={session} onClose={onClose} onSessionUpdate={onSessionUpdate} />}
        </div>
      </div>
    </>
  )
}

function DrawerContent({
  session,
  onClose,
  onSessionUpdate,
}: {
  session: Session
  onClose: () => void
  onSessionUpdate: (session: Session) => void
}) {
  return (
    <>
      <div className="drawer-header">
        <div className="drawer-sprite">
          <Portrait sessionId={session.id} model={session.model} />
        </div>
        <div className="drawer-header-text">
          <div className="name pixel-face">
            {PROVIDER_LABEL[session.provider]} — {session.repo || session.cwd || session.id}
          </div>
          <div className="repo">{session.branch || session.worktree || session.cwd}</div>
          <div className="state-pill pixel-face" style={{ background: STATE_COLOR[session.state], color: session.state === 'waiting' || session.state === 'idle' ? '#2a1a0c' : '#fff8e8' }}>
            {STATE_LABEL[session.state]}
          </div>
        </div>
        <button type="button" className="drawer-close" aria-label="Close" onClick={onClose}>
          ✕
        </button>
      </div>
      <PilotChat session={session} onSessionUpdate={onSessionUpdate} onClose={onClose} />
    </>
  )
}

function PilotChat({
  session,
  onSessionUpdate,
  onClose,
}: {
  session: Session
  onSessionUpdate: (session: Session) => void
  onClose: () => void
}) {
  const [events, setEvents] = useState<PilotEvent[]>([])
  const [draft, setDraft] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const composeRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    setEvents([])
    fetchPilotEvents(session.id)
      .then((list) => {
        setEvents(list)
        // A session with no history yet is exactly what a fresh recruit
        // looks like (Claude Code seats it at camp with no initial
        // message) — focus the compose box so the first quest can be
        // typed right away, same box used for every message after it.
        if (list.length === 0) composeRef.current?.focus()
      })
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
  const canArchive = session.state !== 'working' && !session.archived

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

  const archive = useCallback(() => {
    setBusy(true)
    setError(null)
    archiveSession(session.id)
      .then((updated) => {
        onSessionUpdate(updated)
        onClose()
      })
      .catch((err) => setError(String(err)))
      .finally(() => setBusy(false))
  }, [session.id, onSessionUpdate, onClose])

  return (
    <>
      <div className="drawer-pilot-actions">
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
        {canArchive && (
          <button type="button" disabled={busy} onClick={archive}>
            Archive
          </button>
        )}
      </div>

      {error && <div className="banner banner-error">{error}</div>}
      {isCursor && (
        <div className="banner banner-note">
          Cursor piloted sessions auto-approve every tool call (no live permission gate) — see the transcript for what
          it did.
        </div>
      )}

      <div className="transcript">
        {events.map((event, i) => (
          <TranscriptEntry
            key={i}
            event={event}
            resolved={event.request_id ? resolvedRequestIds.has(event.request_id) : false}
            onApprove={(approve) => runAction(() => resolvePilotPermission(session.id, event.request_id ?? '', approve))}
          />
        ))}
        <div ref={bottomRef} />
      </div>

      <form
        className="compose"
        onSubmit={(e) => {
          e.preventDefault()
          send()
        }}
      >
        <textarea
          ref={composeRef}
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
        <button type="submit" className="pixel-btn" disabled={!canSend || busy || !draft.trim()}>
          Send
        </button>
      </form>
    </>
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
        <div className="bubble user">
          <span className="bubble-label">You</span>
          <p>{event.text}</p>
        </div>
      )
    case 'assistant':
      return (
        <div className="bubble assistant">
          <p>{event.text}</p>
        </div>
      )
    case 'tool_call':
      return (
        <div className="bubble tool">
          <span className="bubble-label">→ {event.tool_name || 'tool'}</span>
          {event.tool_input !== undefined && <pre>{JSON.stringify(event.tool_input, null, 2).slice(0, 2000)}</pre>}
        </div>
      )
    case 'permission_request':
      return (
        <div className="bubble permission">
          <span className="bubble-label">Permission requested: {event.tool_name || 'tool'}</span>
          {event.tool_input !== undefined && <pre>{JSON.stringify(event.tool_input, null, 2).slice(0, 2000)}</pre>}
          {resolved ? (
            <p className="hint">resolved</p>
          ) : (
            <div className="actions">
              <button type="button" className="approve" onClick={() => onApprove(true)}>
                Approve
              </button>
              <button type="button" onClick={() => onApprove(false)}>
                Deny
              </button>
            </div>
          )}
        </div>
      )
    case 'permission_resolved':
      return (
        <div className="bubble system">
          <p>{event.approved ? 'Approved' : 'Denied'} the pending permission request.</p>
        </div>
      )
    case 'error':
      return (
        <div className="bubble error">
          <p>{event.text}</p>
        </div>
      )
    default:
      return (
        <div className="bubble system">
          <p>{event.text}</p>
        </div>
      )
  }
}

