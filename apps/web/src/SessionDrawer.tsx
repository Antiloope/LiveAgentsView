import { useCallback, useEffect, useRef, useState } from 'react'
import type { Character, PilotEvent } from './types'
import { RACE_LABEL, ACTIVITY_COLOR, ACTIVITY_LABEL } from './sprites'
import {
  CloseRune,
  MenuRune,
  PennantRune,
  TentRune,
  PauseRune,
  HaltRune,
  QuillRune,
  SigilRune,
  RaceRune,
} from './Glyphs'
import { Collapsible, Markdown, PortraitThumb, SessionChrome } from './ui'
import {
  archiveCharacter,
  fetchEvents,
  interruptCharacter,
  markRead,
  sendMessage,
  stopCharacter,
  subscribeToEvents,
} from './api'
import './SessionDrawer.css'

const DEFAULT_WIDTH = 460
const MIN_WIDTH = 320
const STORAGE_KEY = 'lav.drawerWidth'

interface Props {
  character: Character | null
  onClose: () => void
  onCharacterUpdate: (character: Character) => void
}

export default function SessionDrawer({ character, onClose, onCharacterUpdate }: Props) {
  const drawerRef = useRef<HTMLDivElement>(null)
  const [dragging, setDragging] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
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

  // Escape closes the board first and the drawer only once nothing is open
  // on top of it, so one press never does both.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape' || !open) return
      if (menuOpen) setMenuOpen(false)
      else onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, menuOpen, onClose])

  useEffect(() => {
    if (!open) setMenuOpen(false)
  }, [open, character?.id])

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
          menuOpen={menuOpen}
          setMenuOpen={setMenuOpen}
        />
      )}
    </SessionChrome>
  )
}

function DrawerContent({
  character,
  onClose,
  onCharacterUpdate,
  menuOpen,
  setMenuOpen,
}: {
  character: Character
  onClose: () => void
  onCharacterUpdate: (character: Character) => void
  menuOpen: boolean
  setMenuOpen: (open: boolean) => void
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

  // A click anywhere else puts the board back up; the board and its plate
  // keep their own clicks from counting as that.
  useEffect(() => {
    if (!menuOpen) return
    const close = () => setMenuOpen(false)
    document.addEventListener('click', close)
    return () => document.removeEventListener('click', close)
  }, [menuOpen, setMenuOpen])

  const working = character.activity === 'working'

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
    setMenuOpen(false)
    if (
      !window.confirm(
        `Archive ${RACE_LABEL[character.race]}? ${working ? 'Its current quest will be stopped. ' : ''}Its transcript and territory stay, and talking to it later wakes it again.`,
      )
    ) {
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
  }, [character.id, character.race, working, onCharacterUpdate, onClose, setMenuOpen])

  const branch = character.territory.branch || character.territory.path

  return (
    <>
      <header className="sheet">
        <div className="plates">
          <button
            type="button"
            className="plate drawer-chamfer-sm"
            aria-expanded={menuOpen}
            aria-label="Character actions"
            onClick={(e) => {
              e.stopPropagation()
              setMenuOpen(!menuOpen)
            }}
          >
            <MenuRune />
          </button>
          <button type="button" className="plate drawer-chamfer-sm drawer-close" aria-label="Close" onClick={onClose}>
            <CloseRune />
          </button>
        </div>

        <div className="board-wrap">
          <div
            className={`board drawer-chamfer${menuOpen ? ' open' : ''}`}
            onClick={(e) => e.stopPropagation()}
            aria-hidden={!menuOpen}
          >
            <span className="board-nail" aria-hidden="true" />
            <div className="board-head">Character actions</div>
            <button
              type="button"
              className="board-item"
              disabled={busy || character.archived}
              tabIndex={menuOpen ? 0 : -1}
              onClick={archive}
            >
              <TentRune />
              <span>
                Archive
                <span className="sub">
                  {character.archived
                    ? 'Already at camp.'
                    : 'Sends it to sleep. Transcript and territory stay.'}
                </span>
              </span>
            </button>
          </div>
        </div>

        <div className="sheet-top">
          <div className="niche">
            <PortraitThumb characterId={character.id} race={character.race} />
          </div>
          <div className="ident">
            <div className="holding" title={character.repo || character.territory.path}>
              {character.repo || character.territory.path || character.id}
            </div>
            <div className="pennant">
              <PennantRune />
              <span className="branch" title={branch}>
                {branch}
              </span>
              <span className="sheet-dot" aria-hidden="true" />
              <span className="terr">{character.territory.mode === 'own' ? 'own territory' : 'shared territory'}</span>
            </div>
            <div className="build">
              <span className="race">
                <RaceRune race={character.race} />
                {RACE_LABEL[character.race]}
              </span>
              {character.class && (
                <>
                  <span className="sheet-dot" aria-hidden="true" />
                  <span className="stone drawer-chamfer-sm">{character.class}</span>
                </>
              )}
            </div>
          </div>
        </div>
      </header>

      <div
        className="condition"
        data-cond={character.activity}
        style={{ ['--cond' as string]: ACTIVITY_COLOR[character.activity] }}
      >
        <span className="gem" aria-hidden="true" />
        <span className="cond-label">{ACTIVITY_LABEL[character.activity]}</span>
        <span className="cond-note">{conditionNote(character)}</span>
      </div>

      {working && (
        <div className="quest-strip" role="status" aria-live="polite">
          <span className="quest-spark" aria-hidden="true" />
          <span>{questStatus(events)}</span>
        </div>
      )}

      <div className="log">
        {events.map((event, i) => (
          <TranscriptEntry key={i} event={event} />
        ))}
        {error && (
          <div className="entry note note--error">
            <SigilRune className="rune" />
            {error}
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <form
        className="desk"
        onSubmit={(e) => {
          e.preventDefault()
          send()
        }}
      >
        <div className="sheetline">
          <textarea
            ref={composeRef}
            className="vellum"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder={working ? 'Message this character — delivered once its current quest ends…' : 'Message this character…'}
            disabled={busy}
            rows={3}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                send()
              }
            }}
          />
          <button type="submit" className="press" disabled={busy || !draft.trim()} aria-label="Send">
            <QuillRune />
          </button>
        </div>
        <div className="desk-foot">
          <span>Enter sends · Shift+Enter new line</span>
          {working && (
            <span className="warband">
              <button
                type="button"
                className="plaque drawer-chamfer-sm"
                disabled={busy}
                onClick={() => runAction(() => interruptCharacter(character.id))}
              >
                <PauseRune /> Interrupt
              </button>
              <button
                type="button"
                className="plaque plaque--halt drawer-chamfer-sm"
                disabled={busy}
                onClick={() => runAction(() => stopCharacter(character.id))}
              >
                <HaltRune /> Stop
              </button>
            </span>
          )}
        </div>
      </form>
    </>
  )
}

// The line under the condition: what the activity means for the user right
// now. Presence only tells them anything when the character is at camp, so
// it is spelled out there and left out of the other three.
function conditionNote(character: Character): string {
  switch (character.activity) {
    case 'working':
      return 'out on a quest'
    case 'waiting':
      return 'waiting on your answer'
    case 'failed':
      return 'its quest failed'
    default:
      return character.presence === 'asleep' ? 'asleep at camp' : 'awake at camp'
  }
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
        <div className="entry strip">
          <div className="said">{event.text}</div>
          <span className="seal" aria-hidden="true">
            <SigilRune className="rune" />
          </span>
        </div>
      )
    case 'assistant':
      return (
        <div className="entry">
          <SigilRune className="rune" />
          <div className="said">
            <Markdown text={event.text ?? ''} />
          </div>
        </div>
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
        <Collapsible label={toolLabel(event.tool_name)} summary={summary ?? undefined}>
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
        <div className="entry note note--error">
          <SigilRune className="rune" />
          {event.text}
        </div>
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
        <div className="entry note note--system">
          <SigilRune className="rune" />
          {event.text}
        </div>
      )
    }
  }
}
