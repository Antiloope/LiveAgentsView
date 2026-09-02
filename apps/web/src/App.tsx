import { useEffect, useMemo, useState, useCallback, type FormEvent } from 'react'
import type { PilotProvider, Session, State } from './types'
import { fetchSessions, subscribeToSessions, openTerminal, launchPilotedSession } from './api'
import PilotedSessionView from './PilotedSessionView'

// Grouping and order follow attention priority: BLOCKED and FAILED surface
// loudly, DONE is grouped and quiet.
const GROUPS: { state: State; label: string; hint: string }[] = [
  { state: 'blocked', label: 'Needs you now', hint: 'waiting on a permission decision' },
  { state: 'failed', label: 'Failed', hint: 'errored or crashed' },
  { state: 'waiting', label: 'Asked you something', hint: 'end of turn, expects a reply' },
  { state: 'working', label: 'Working', hint: 'actively running' },
  { state: 'done', label: 'Finished', hint: 'closed the task, low priority' },
  { state: 'idle', label: 'Idle', hint: 'session ended or no signal yet' },
]

const PROVIDER_LABEL: Record<string, string> = {
  'claude-code': 'Claude Code',
  codex: 'Codex',
  cursor: 'Cursor',
}

export default function App() {
  const [sessions, setSessions] = useState<Record<string, Session>>({})
  const [error, setError] = useState<string | null>(null)
  const [openSessionId, setOpenSessionId] = useState<string | null>(null)
  const [showNewForm, setShowNewForm] = useState(false)

  useEffect(() => {
    fetchSessions()
      .then((list) => {
        const byId: Record<string, Session> = {}
        for (const s of list) byId[s.id] = s
        setSessions(byId)
      })
      .catch((err) => setError(String(err)))

    return subscribeToSessions((session) => {
      setSessions((prev) => ({ ...prev, [session.id]: session }))
    })
  }, [])

  const grouped = useMemo(() => {
    const all = Object.values(sessions)
    return GROUPS.map((g) => ({
      ...g,
      sessions: all
        .filter((s) => s.state === g.state)
        .sort((a, b) => (a.updated_at < b.updated_at ? 1 : -1)),
    }))
  }, [sessions])

  const total = Object.keys(sessions).length
  const openSession = openSessionId ? sessions[openSessionId] : null

  if (openSession) {
    return (
      <div className="app">
        <PilotedSessionView
          session={openSession}
          onClose={() => setOpenSessionId(null)}
          onSessionUpdate={(s) => setSessions((prev) => ({ ...prev, [s.id]: s }))}
        />
      </div>
    )
  }

  return (
    <div className="app">
      <header className="app-header">
        <h1>LiveAgentsView</h1>
        <span className="subtitle">
          {total} session{total === 1 ? '' : 's'} known
        </span>
        <button type="button" className="new-piloted" onClick={() => setShowNewForm(true)}>
          New piloted session
        </button>
      </header>

      {showNewForm && (
        <NewPilotedSessionForm
          onCancel={() => setShowNewForm(false)}
          onLaunched={(session) => {
            setSessions((prev) => ({ ...prev, [session.id]: session }))
            setShowNewForm(false)
            setOpenSessionId(session.id)
          }}
        />
      )}

      {error && <div className="banner banner-error">Could not load sessions: {error}</div>}

      {total === 0 && !error && (
        <div className="empty">
          No sessions yet. Run <code>lav init</code>, then start a Claude Code, Codex or
          Cursor session natively — it will show up here. Or launch a piloted session above.
        </div>
      )}

      {grouped
        .filter((g) => g.sessions.length > 0)
        .map((g) => (
          <section key={g.state} className={`group group-${g.state}`}>
            <h2>
              {g.label} <span className="count">{g.sessions.length}</span>
            </h2>
            <p className="hint">{g.hint}</p>
            <ul className="session-list">
              {g.sessions.map((s) => (
                <li key={s.id} className="session-card">
                  <div className="session-main">
                    <span className="provider">{PROVIDER_LABEL[s.provider] ?? s.provider}</span>
                    <span className="repo">{s.repo || s.cwd || s.id}</span>
                    {s.branch && <span className="branch">{s.branch}</span>}
                    {s.worktree && <span className="worktree">{s.worktree}</span>}
                    <span className="fidelity">{s.fidelity}</span>
                  </div>
                  {s.last_message && <p className="last-message">{s.last_message}</p>}
                  <div className="session-meta">
                    <span title={s.cwd}>{s.cwd}</span>
                    {s.fidelity === 'driver' ? (
                      <button type="button" className="view-chat" onClick={() => setOpenSessionId(s.id)}>
                        View chat
                      </button>
                    ) : (
                      <>
                        <OpenTerminalButton path={s.cwd} />
                        <CopyPathButton path={s.cwd} />
                      </>
                    )}
                    <time dateTime={s.updated_at}>{new Date(s.updated_at).toLocaleString()}</time>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        ))}
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
    <form className="new-session-form" onSubmit={submit}>
      <h2>New piloted session</h2>
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
        <button type="submit" disabled={busy || !cwd.trim() || !prompt.trim()}>
          {busy ? 'Launching…' : 'Launch'}
        </button>
      </div>
    </form>
  )
}

// Asks the daemon to spawn a terminal at path. Only works when the daemon
// runs natively on the host (scripts/lav-service-install.sh), not via
// scripts/dev-up.sh's containerized daemon — the request fails there since
// the container has no terminal to open. CopyPathButton stays as a
// fallback for that case.
function OpenTerminalButton({ path }: { path: string }) {
  const [failed, setFailed] = useState(false)

  const open = useCallback(() => {
    openTerminal(path)
      .then(() => setFailed(false))
      .catch(() => {
        setFailed(true)
        setTimeout(() => setFailed(false), 2000)
      })
  }, [path])

  if (!path) return null
  return (
    <button type="button" className="open-terminal" onClick={open} title="Open in terminal">
      {failed ? 'Could not open' : 'Open terminal'}
    </button>
  )
}

function CopyPathButton({ path }: { path: string }) {
  const [copied, setCopied] = useState(false)

  const copy = useCallback(() => {
    navigator.clipboard
      .writeText(path)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      })
      .catch(() => {})
  }, [path])

  if (!path) return null
  return (
    <button type="button" className="copy-path" onClick={copy} title="Copy path">
      {copied ? 'Copied' : 'Copy path'}
    </button>
  )
}
