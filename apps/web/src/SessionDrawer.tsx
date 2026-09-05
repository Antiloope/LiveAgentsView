import { useCallback, useEffect, useRef, useState } from 'react'
import type { Character, PilotEvent } from './types'
import { RACE_LABEL, ACTIVITY_COLOR, ACTIVITY_LABEL } from './sprites'
import { Button, ChatBubble, Collapsible, HudLabel, Markdown, PortraitThumb, QuestIndicator, SessionChrome } from './ui'
import {
  archiveCharacter,
  dismissCharacter,
  fetchEvents,
  interruptCharacter,
  markRead,
  sendMessage,
  stopCharacter,
  subscribeToEvents,
} from './api'

const DEFAULT_WIDTH = 460
const MIN_WIDTH = 320
const STORAGE_KEY = 'lav.drawerWidth'

interface Props {
  character: Character | null
  onClose: () => void
  onCharacterUpdate: (character: Character) => void
  onDismissed: (id: string) => void
}

export default function SessionDrawer({ character, onClose, onCharacterUpdate, onDismissed }: Props) {
  const drawerRef = useRef<HTMLDivElement>(null)
  const [dragging, setDragging] = useState(false)
  const open = character !== null

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
  }, [open, character?.id])

  return (
    <SessionChrome
      open={open}
      drawerRef={drawerRef}
      width={DEFAULT_WIDTH}
      dragging={dragging}
      onHandleMouseDown={startDrag}
      label={character ? `${RACE_LABEL[character.race]} character` : undefined}
    >
      {character && (
        <DrawerContent
          character={character}
          onClose={onClose}
          onCharacterUpdate={onCharacterUpdate}
          onDismissed={onDismissed}
        />
      )}
    </SessionChrome>
  )
}

function DrawerContent({
  character,
  onClose,
  onCharacterUpdate,
  onDismissed,
}: {
  character: Character
  onClose: () => void
  onCharacterUpdate: (character: Character) => void
  onDismissed: (id: string) => void
}) {
  return (
    <>
      <div className="drawer-header">
        <div className="drawer-sprite">
          <PortraitThumb characterId={character.id} race={character.race} />
        </div>
        <div className="drawer-header-text">
          <HudLabel as="div" className="name">
            {RACE_LABEL[character.race]} — {character.repo || character.territory.path || character.id}
          </HudLabel>
          <div className="repo">{character.territory.branch || character.territory.path}</div>
          {character.class && <span className="class-badge mono">{character.class}</span>}
          <HudLabel
            as="div"
            className="state-pill"
            style={{ background: ACTIVITY_COLOR[character.activity], color: character.activity === 'waiting' ? '#2a1a0c' : '#fff8e8' }}
          >
            {ACTIVITY_LABEL[character.activity]} · {character.presence}
          </HudLabel>
        </div>
        <button type="button" className="drawer-close" aria-label="Close" onClick={onClose}>
          ✕
        </button>
      </div>
      <PilotChat character={character} onCharacterUpdate={onCharacterUpdate} onClose={onClose} onDismissed={onDismissed} />
    </>
  )
}

function PilotChat({
  character,
  onCharacterUpdate,
  onClose,
  onDismissed,
}: {
  character: Character
  onCharacterUpdate: (character: Character) => void
  onClose: () => void
  onDismissed: (id: string) => void
}) {
  const [events, setEvents] = useState<PilotEvent[]>([])
  const [draft, setDraft] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const composeRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    setEvents([])
    // The history fetch and the live subscription race: a live event can
    // arrive before the fetch resolves. Buffer those instead of appending
    // them immediately, so the fetch's own setEvents(list) below — which
    // must overwrite the initial [] — can't clobber them. Once resolved,
    // fold in whatever was buffered, skipping any the fetch's own read
    // already picked up (same character, same server-assigned `at`).
    let resolved = false
    const buffered: PilotEvent[] = []
    fetchEvents(character.id)
      .then((list) => {
        resolved = true
        const known = new Set(list.map((e) => JSON.stringify(e)))
        const extra = buffered.filter((e) => !known.has(JSON.stringify(e)))
        const merged = [...list, ...extra]
        setEvents(merged)
        // A character with no history yet is exactly what a fresh recruit
        // looks like — focus the compose box so its first quest can be
        // typed right away, same box used for every message after it.
        if (merged.length === 0) composeRef.current?.focus()
      })
      .catch((err) => setError(String(err)))
    return subscribeToEvents(character.id, (event) => {
      if (!resolved) {
        buffered.push(event)
        return
      }
      setEvents((prev) => [...prev, event])
    })
  }, [character.id])

  // The interface's own explicit "read" signal — the only thing that
  // clears the unread mark, per the character model. Fires once when the
  // drawer opens on an already-unread character, and again if a quest ends
  // without a question while its drawer is already open — the user is
  // watching this exact transcript live, so that news is seen the moment
  // it arrives, not left marked unread until the drawer is reopened.
  // Fire-and-forget: a failure here just leaves the mark up for next time.
  useEffect(() => {
    if (character.unread) markRead(character.id).catch(() => undefined)
  }, [character.id, character.unread])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'end' })
  }, [events.length, character.activity])

  const canStop = character.activity === 'working'
  const canArchive = !character.archived

  const runAction = useCallback(
    (action: () => Promise<void | Character>) => {
      setBusy(true)
      setError(null)
      action()
        .then((updated) => {
          if (updated) onCharacterUpdate(updated)
        })
        .catch((err) => setError(String(err)))
        .finally(() => setBusy(false))
    },
    [onCharacterUpdate],
  )

  const send = useCallback(() => {
    const text = draft.trim()
    if (!text) return
    setDraft('')
    runAction(() => sendMessage(character.id, text))
  }, [draft, runAction, character.id])

  const archive = useCallback(() => {
    if (!window.confirm(`Archive ${RACE_LABEL[character.race]}? ${canStop ? 'Its current quest will be stopped. ' : ''}Its transcript and territory stay, and talking to it later wakes it again.`)) {
      return
    }
    setBusy(true)
    setError(null)
    archiveCharacter(character.id)
      .then((updated) => {
        onCharacterUpdate(updated)
        onClose()
      })
      .catch((err) => setError(String(err)))
      .finally(() => setBusy(false))
  }, [character.id, character.race, canStop, onCharacterUpdate, onClose])

  const dismiss = useCallback(() => {
    if (!window.confirm(`Dismiss ${RACE_LABEL[character.race]} for good? This stops it and deletes its transcript. This cannot be undone.`)) {
      return
    }
    setBusy(true)
    setError(null)
    dismissCharacter(character.id)
      .then(({ worktreeLeftAt }) => {
        onDismissed(character.id)
        onClose()
        if (worktreeLeftAt) {
          window.alert(`Its worktree had uncommitted changes, so it was left in place at:\n${worktreeLeftAt}`)
        }
      })
      .catch((err) => setError(String(err)))
      .finally(() => setBusy(false))
  }, [character.id, character.race, onDismissed, onClose])

  return (
    <>
      <div className="drawer-pilot-actions">
        {canStop && (
          <>
            <Button type="button" disabled={busy} onClick={() => runAction(() => interruptCharacter(character.id))}>
              Interrupt
            </Button>
            <Button type="button" variant="danger" disabled={busy} onClick={() => runAction(() => stopCharacter(character.id))}>
              Stop
            </Button>
          </>
        )}
        {canArchive && (
          <Button type="button" disabled={busy} onClick={archive}>
            Archive
          </Button>
        )}
        <Button type="button" variant="danger" disabled={busy} onClick={dismiss}>
          Dismiss
        </Button>
      </div>

      {error && <div className="banner banner-error">{error}</div>}

      <div className="transcript">
        {events.map((event, i) => (
          <TranscriptEntry key={i} event={event} />
        ))}
        {character.activity === 'working' && <QuestIndicator text={questStatus(events)} />}
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
          placeholder={character.activity === 'working' ? 'Message this character — delivered once its current quest ends…' : 'Message this character…'}
          disabled={busy}
          rows={2}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              send()
            }
          }}
        />
        <Button type="submit" disabled={busy || !draft.trim()}>
          Send
        </Button>
      </form>
    </>
  )
}

// The one field worth showing on a collapsed tool row, in the order a tool
// is most likely to carry it. cursor-agent spells them in camelCase and
// Claude Code in snake_case, so both spellings are listed.
const TOOL_SUMMARY_FIELDS = [
  'command',
  'pattern',
  'globPattern',
  'query',
  'searchTerm',
  'file_path',
  'filePath',
  'path',
  'prompt',
  'url',
  'targetDirectory',
]

function pickSummaryField(fields: Record<string, unknown>): string | null {
  for (const field of TOOL_SUMMARY_FIELDS) {
    const v = fields[field]
    if (typeof v === 'string' && v !== '') return v.length > 300 ? v.slice(0, 300) + '…' : v
  }
  return null
}

// cursor-agent's tool_call payload wraps args in a discriminated union
// alongside call bookkeeping — {"shellToolCall":{"args":{"command":...,
// "parsingResult":{...}},"description":"..."},"toolCallId":"...",
// "startedAtMs":"...","hookAdditionalContexts":[]} — nearly all of it noise
// next to the one field worth showing. Claude Code's tool_input is that
// same field layout already flat, so both shapes are tried in turn and
// anything neither matches keeps the raw JSON as its only detail.
function summarizeToolInput(input: unknown): string | null {
  if (typeof input !== 'object' || input === null) return null
  for (const value of Object.values(input as Record<string, unknown>)) {
    if (typeof value !== 'object' || value === null) continue
    const args = (value as Record<string, unknown>).args
    if (typeof args !== 'object' || args === null) continue
    const summary = pickSummaryField(args as Record<string, unknown>)
    if (summary !== null) return summary
  }
  return pickSummaryField(input as Record<string, unknown>)
}

// Claude Code names its tools for people (Bash, Read, Edit); cursor-agent
// names them for its own union (shellToolCall), so trim that suffix to get
// the same short word in the row.
function toolLabel(name: string | undefined): string {
  if (!name) return 'tool'
  return name.endsWith('ToolCall') ? name.slice(0, -'ToolCall'.length) : name
}

// First line of a block, cut to something a single row can hold.
function firstLine(text: string, max = 120): string {
  const line = text.trim().split('\n')[0]
  return line.length > max ? line.slice(0, max) + '…' : line
}

// A system event carrying a raw provider line (rate_limit_event and any
// other type the daemon has no case for) is JSON, not a sentence: reformat
// it so it can sit behind a collapsed row instead of filling the transcript.
function rawProviderLine(text: string): string | null {
  const trimmed = text.trim()
  if (!trimmed.startsWith('{')) return null
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return null
  }
}

// What the character is doing right now, read off the tail of its own
// transcript — the last thing it did is the best available description of
// the step it is in.
function questStatus(events: PilotEvent[]): string {
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i]
    if (event.kind === 'tool_call') return `Running ${toolLabel(event.tool_name)}…`
    if (event.kind === 'thinking') return 'Thinking…'
    if (event.kind === 'assistant' || event.kind === 'user') break
  }
  return 'On the quest…'
}

function TranscriptEntry({ event }: { event: PilotEvent }) {
  switch (event.kind) {
    case 'user':
      return (
        <ChatBubble variant="user" label="You">
          <p>{event.text}</p>
        </ChatBubble>
      )
    case 'assistant':
      return (
        <ChatBubble variant="assistant">
          <Markdown text={event.text ?? ''} />
        </ChatBubble>
      )
    case 'thinking':
      return (
        <Collapsible tone="thinking" label="Reasoning" summary={firstLine(event.text ?? '')}>
          <Markdown text={event.text ?? ''} />
        </Collapsible>
      )
    case 'tool_call': {
      const summary = summarizeToolInput(event.tool_input)
      return (
        <Collapsible label={`→ ${toolLabel(event.tool_name)}`} summary={summary ?? undefined}>
          {event.tool_input !== undefined ? (
            <pre>{JSON.stringify(event.tool_input, null, 2).slice(0, 20000)}</pre>
          ) : (
            <p>No input recorded.</p>
          )}
        </Collapsible>
      )
    }
    case 'error':
      return (
        <ChatBubble variant="error">
          <p>{event.text}</p>
        </ChatBubble>
      )
    default: {
      const raw = rawProviderLine(event.text ?? '')
      if (raw !== null) {
        return (
          <Collapsible tone="raw" label="provider line" summary={firstLine(raw.replace(/\s+/g, ' '))}>
            <pre>{raw}</pre>
          </Collapsible>
        )
      }
      return (
        <ChatBubble variant="system">
          <p>{event.text}</p>
        </ChatBubble>
      )
    }
  }
}
