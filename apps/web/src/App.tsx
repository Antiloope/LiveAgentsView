import { useEffect, useMemo, useState, useCallback, type FormEvent } from 'react'
import type { PilotProvider, Session } from './types'
import { fetchSessions, subscribeToSessions, launchPilotedSession } from './api'
import { NEEDS_ATTENTION } from './sprites'
import PartyStand from './PartyStand'
import QuestToken from './QuestToken'
import SessionDrawer from './SessionDrawer'

export default function App() {
  const [sessions, setSessions] = useState<Record<string, Session>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showNewForm, setShowNewForm] = useState(false)

  useEffect(() => {
    fetchSessions()
      .then((list) => {
        const byId: Record<string, Session> = {}
        for (const s of list) byId[s.id] = s
        setSessions(byId)
      })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false))

    return subscribeToSessions((session) => {
      setSessions((prev) => ({ ...prev, [session.id]: session }))
    })
  }, [])

  const { questSessions, urgentCamp, calmCamp } = useMemo(() => {
    const all = Object.values(sessions).sort((a, b) => (a.updated_at < b.updated_at ? 1 : -1))
    const quest = all.filter((s) => s.state === 'working')
    const camp = all.filter((s) => s.state !== 'working')
    return {
      questSessions: quest,
      urgentCamp: camp.filter((s) => NEEDS_ATTENTION.includes(s.state)),
      calmCamp: camp.filter((s) => !NEEDS_ATTENTION.includes(s.state)),
    }
  }, [sessions])

  const total = Object.keys(sessions).length
  const selectedSession = selectedId ? (sessions[selectedId] ?? null) : null

  const selectSession = useCallback((id: string) => {
    setSelectedId((prev) => (prev === id ? null : id))
  }, [])

  return (
    <div className="app">
      <header className="topbar">
        <div>
          <h1 className="pixel-face">LiveAgentsView</h1>
          <span className="subtitle">
            {total} session{total === 1 ? '' : 's'} known
          </span>
        </div>
        <button type="button" className="pixel-btn recruit-btn" onClick={() => setShowNewForm(true)}>
          + Recruit session
        </button>
      </header>

      {error && <div className="banner banner-error">Could not load sessions: {error}</div>}

      {!loading && total === 0 && !error && (
        <div className="empty">
          No sessions yet. Run <code>lav init</code>, then start a Claude Code, Codex or Cursor session natively — it
          will show up here. Or recruit a piloted session above.
        </div>
      )}

      <div className="layout-row">
        <aside className="sidebar">
          <div className="sidebar-header">
            <span className="pixel-face">OUT ON QUESTS</span>
            <span className="count">{questSessions.length}</span>
          </div>
          <div className="sidebar-list">
            {loading ? (
              <div className="sidebar-empty">Loading…</div>
            ) : questSessions.length === 0 ? (
              <div className="sidebar-empty">No one is out right now.</div>
            ) : (
              questSessions.map((s) => (
                <QuestToken key={s.id} session={s} selected={s.id === selectedId} onSelect={selectSession} />
              ))
            )}
          </div>
        </aside>

        <div className="scene">
          <div className="stars" />
          <div className="treeline" />
          <div className="ground-tex" />
          <div className="tent" />
          <div className="tent right" />
          <div className="banner-pole" />
          <div className="firelight" />
          <div className="campfire">
            <div className="log" />
            <div className="flame" />
          </div>
          <div className="camp-floor">
            <div className="camp-row back">
              {calmCamp.map((s) => (
                <PartyStand key={s.id} session={s} calm selected={s.id === selectedId} onSelect={selectSession} />
              ))}
            </div>
            <div className="camp-row front">
              {urgentCamp.map((s) => (
                <PartyStand key={s.id} session={s} selected={s.id === selectedId} onSelect={selectSession} />
              ))}
            </div>
          </div>
          {!loading && urgentCamp.length === 0 && calmCamp.length === 0 && (
            <div className="empty-camp">Camp is quiet — everyone's out on a quest.</div>
          )}
        </div>
      </div>

      <SessionDrawer
        session={selectedSession}
        onClose={() => setSelectedId(null)}
        onSessionUpdate={(s) => setSessions((prev) => ({ ...prev, [s.id]: s }))}
      />

      {showNewForm && (
        <NewPilotedSessionForm
          onCancel={() => setShowNewForm(false)}
          onLaunched={(session) => {
            setSessions((prev) => ({ ...prev, [session.id]: session }))
            setShowNewForm(false)
            setSelectedId(session.id)
          }}
        />
      )}
    </div>
  )
}

const PILOT_PROVIDERS: { value: PilotProvider; label: string }[] = [
  { value: 'claude-code', label: 'Claude Code' },
  { value: 'cursor', label: 'Cursor (auto-approves every tool call — see docs before pointing it at anything real)' },
]

function NewPilotedSessionForm({
  onLaunched,
  onCancel,
}: {
  onLaunched: (session: Session) => void
  onCancel: () => void
}) {
  const [provider, setProvider] = useState<PilotProvider>('claude-code')
  const [cwd, setCwd] = useState('')
  const [branch, setBranch] = useState('')
  const [prompt, setPrompt] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = useCallback(
    (e: FormEvent) => {
      e.preventDefault()
      setBusy(true)
      setError(null)
      launchPilotedSession({ provider, cwd: cwd.trim(), branch: branch.trim(), prompt: prompt.trim() })
        .then(onLaunched)
        .catch((err) => setError(String(err)))
        .finally(() => setBusy(false))
    },
    [provider, cwd, branch, prompt, onLaunched],
  )

  return (
    <div className="modal-scrim" onClick={onCancel}>
      <form className="modal new-session-form" onSubmit={submit} onClick={(e) => e.stopPropagation()}>
        <h2 className="pixel-face">Recruit a piloted session</h2>
        {error && <div className="banner banner-error">{error}</div>}
        <label>
          Provider
          <select value={provider} onChange={(e) => setProvider(e.target.value as PilotProvider)}>
            {PILOT_PROVIDERS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Directory
          <input
            type="text"
            value={cwd}
            onChange={(e) => setCwd(e.target.value)}
            placeholder="/path/to/an/existing/repo/or/worktree"
            required
          />
        </label>
        <label>
          Branch (optional — must already exist)
          <input type="text" value={branch} onChange={(e) => setBranch(e.target.value)} placeholder="" />
        </label>
        <label>
          Initial prompt
          <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={3} required />
        </label>
        <div className="new-session-actions">
          <button type="button" onClick={onCancel} disabled={busy}>
            Cancel
          </button>
          <button type="submit" className="pixel-btn" disabled={busy || !cwd.trim() || !prompt.trim()}>
            {busy ? 'Launching…' : 'Launch'}
          </button>
        </div>
      </form>
    </div>
  )
}
