import { useEffect, useMemo, useState, useCallback } from 'react'
import type { Session, State } from './types'
import { fetchSessions, subscribeToSessions } from './api'

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

  return (
    <div className="app">
      <header className="app-header">
        <h1>LiveAgentsView</h1>
        <span className="subtitle">
          {total} session{total === 1 ? '' : 's'} known
        </span>
      </header>

      {error && <div className="banner banner-error">Could not load sessions: {error}</div>}

      {total === 0 && !error && (
        <div className="empty">
          No sessions yet. Run <code>lav init</code>, then start a Claude Code, Codex or
          Cursor session natively — it will show up here.
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
                    <CopyPathButton path={s.cwd} />
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

// Opening a terminal at the session's path would need either a native
// helper on the host or the ability to spawn a process there — the daemon
// runs in a container and cannot spawn anything on the host. Copying the
// path is the available alternative.
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
