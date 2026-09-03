import { useEffect, useMemo, useState, useCallback } from 'react'
import type { Session } from './types'
import { fetchSessions, subscribeToSessions, unarchiveSession } from './api'
import { NEEDS_ATTENTION, PROVIDER_LABEL, STATE_LABEL } from './sprites'
import PartyStand from './PartyStand'
import QuestToken from './QuestToken'
import SessionDrawer from './SessionDrawer'
import RecruitPanel from './RecruitPanel'

export default function App() {
  const [sessions, setSessions] = useState<Record<string, Session>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showNewForm, setShowNewForm] = useState(false)
  const [showArchived, setShowArchived] = useState(false)

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

  const { questSessions, urgentCamp, calmCamp, archivedSessions } = useMemo(() => {
    const all = Object.values(sessions).sort((a, b) => (a.updated_at < b.updated_at ? 1 : -1))
    const visible = all.filter((s) => !s.archived)
    const quest = visible.filter((s) => s.state === 'working')
    const camp = visible.filter((s) => s.state !== 'working')
    return {
      questSessions: quest,
      urgentCamp: camp.filter((s) => NEEDS_ATTENTION.includes(s.state)),
      calmCamp: camp.filter((s) => !NEEDS_ATTENTION.includes(s.state)),
      archivedSessions: all.filter((s) => s.archived),
    }
  }, [sessions])

  const total = questSessions.length + urgentCamp.length + calmCamp.length
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
        <div className="topbar-actions">
          <button type="button" className="pixel-btn archived-btn" onClick={() => setShowArchived(true)}>
            Archived ({archivedSessions.length})
          </button>
          <button type="button" className="pixel-btn recruit-btn" onClick={() => setShowNewForm(true)}>
            + Recruit session
          </button>
        </div>
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
        <RecruitPanel
          onCancel={() => setShowNewForm(false)}
          onLaunched={(session) => {
            setSessions((prev) => ({ ...prev, [session.id]: session }))
            setShowNewForm(false)
            setSelectedId(session.id)
          }}
        />
      )}

      {showArchived && (
        <ArchivedSessionsModal
          sessions={archivedSessions}
          onClose={() => setShowArchived(false)}
          onUnarchive={(session) => setSessions((prev) => ({ ...prev, [session.id]: session }))}
        />
      )}
    </div>
  )
}

function ArchivedSessionsModal({
  sessions,
  onClose,
  onUnarchive,
}: {
  sessions: Session[]
  onClose: () => void
  onUnarchive: (session: Session) => void
}) {
  const [busyId, setBusyId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const unarchive = useCallback(
    (id: string) => {
      setBusyId(id)
      setError(null)
      unarchiveSession(id)
        .then(onUnarchive)
        .catch((err) => setError(String(err)))
        .finally(() => setBusyId(null))
    },
    [onUnarchive],
  )

  return (
    <div className="modal-scrim" onClick={onClose}>
      <div className="modal archived-modal" onClick={(e) => e.stopPropagation()}>
        <h2 className="pixel-face">Archived sessions</h2>
        {error && <div className="banner banner-error">{error}</div>}
        {sessions.length === 0 ? (
          <p className="archived-empty">No archived sessions.</p>
        ) : (
          <ul className="archived-list">
            {sessions.map((s) => (
              <li key={s.id} className="archived-row">
                <div className="archived-row-info">
                  <span className="archived-row-title">
                    {PROVIDER_LABEL[s.provider]} — {s.repo || s.cwd || s.id}
                  </span>
                  <span className="archived-row-meta">
                    {STATE_LABEL[s.state]}
                    {s.last_message ? ` · ${s.last_message.slice(0, 120)}` : ''}
                  </span>
                </div>
                <button type="button" disabled={busyId === s.id} onClick={() => unarchive(s.id)}>
                  Unarchive
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="new-session-actions">
          <button type="button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
