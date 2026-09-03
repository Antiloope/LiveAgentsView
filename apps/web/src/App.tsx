import { useEffect, useMemo, useState, useCallback } from 'react'
import type { Character } from './types'
import { fetchCharacters, subscribeToCharacters, unarchiveCharacter } from './api'
import { NEEDS_ATTENTION, RACE_LABEL, ACTIVITY_LABEL } from './sprites'
import PartyStand from './PartyStand'
import QuestToken from './QuestToken'
import SessionDrawer from './SessionDrawer'
import RecruitPanel from './RecruitPanel'

export default function App() {
  const [characters, setCharacters] = useState<Record<string, Character>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showRecruit, setShowRecruit] = useState(false)
  const [showArchived, setShowArchived] = useState(false)

  useEffect(() => {
    fetchCharacters()
      .then((list) => {
        const byId: Record<string, Character> = {}
        for (const c of list) byId[c.id] = c
        setCharacters(byId)
      })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false))

    return subscribeToCharacters((character) => {
      setCharacters((prev) => ({ ...prev, [character.id]: character }))
    })
  }, [])

  const { questCharacters, urgentCamp, calmCamp, archivedCharacters } = useMemo(() => {
    const all = Object.values(characters).sort((a, b) => (a.updated_at < b.updated_at ? 1 : -1))
    const visible = all.filter((c) => !c.archived)
    const quest = visible.filter((c) => c.activity === 'working')
    const camp = visible.filter((c) => c.activity !== 'working')
    return {
      questCharacters: quest,
      urgentCamp: camp.filter((c) => NEEDS_ATTENTION.includes(c.activity)),
      calmCamp: camp.filter((c) => !NEEDS_ATTENTION.includes(c.activity)),
      archivedCharacters: all.filter((c) => c.archived),
    }
  }, [characters])

  const total = questCharacters.length + urgentCamp.length + calmCamp.length
  const selectedCharacter = selectedId ? (characters[selectedId] ?? null) : null

  const selectCharacter = useCallback((id: string) => {
    setSelectedId((prev) => (prev === id ? null : id))
  }, [])

  const dismissCharacterLocally = useCallback(
    (id: string) => {
      setCharacters((prev) => {
        const next = { ...prev }
        delete next[id]
        return next
      })
      setSelectedId((prev) => (prev === id ? null : prev))
    },
    [],
  )

  return (
    <div className="app">
      <header className="topbar">
        <div>
          <h1 className="pixel-face">LiveAgentsView</h1>
          <span className="subtitle">
            {total} character{total === 1 ? '' : 's'} known
          </span>
        </div>
        <div className="topbar-actions">
          <button type="button" className="pixel-btn archived-btn" onClick={() => setShowArchived(true)}>
            Archived ({archivedCharacters.length})
          </button>
          <button type="button" className="pixel-btn recruit-btn" onClick={() => setShowRecruit(true)}>
            + Recruit
          </button>
        </div>
      </header>

      {error && <div className="banner banner-error">Could not load characters: {error}</div>}

      {!loading && total === 0 && !error && (
        <div className="empty">No characters yet. Recruit one above to bring it to camp.</div>
      )}

      <div className="layout-row">
        <aside className="sidebar">
          <div className="sidebar-header">
            <span className="pixel-face">OUT ON QUESTS</span>
            <span className="count">{questCharacters.length}</span>
          </div>
          <div className="sidebar-list">
            {loading ? (
              <div className="sidebar-empty">Loading…</div>
            ) : questCharacters.length === 0 ? (
              <div className="sidebar-empty">No one is out right now.</div>
            ) : (
              questCharacters.map((c) => (
                <QuestToken key={c.id} character={c} selected={c.id === selectedId} onSelect={selectCharacter} />
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
              {calmCamp.map((c) => (
                <PartyStand key={c.id} character={c} calm selected={c.id === selectedId} onSelect={selectCharacter} />
              ))}
            </div>
            <div className="camp-row front">
              {urgentCamp.map((c) => (
                <PartyStand key={c.id} character={c} selected={c.id === selectedId} onSelect={selectCharacter} />
              ))}
            </div>
          </div>
          {!loading && urgentCamp.length === 0 && calmCamp.length === 0 && (
            <div className="empty-camp">Camp is quiet — everyone's out on a quest.</div>
          )}
        </div>
      </div>

      <SessionDrawer
        character={selectedCharacter}
        onClose={() => setSelectedId(null)}
        onCharacterUpdate={(c) => setCharacters((prev) => ({ ...prev, [c.id]: c }))}
        onDismissed={dismissCharacterLocally}
      />

      {showRecruit && (
        <RecruitPanel
          onCancel={() => setShowRecruit(false)}
          onRecruited={(character) => {
            setCharacters((prev) => ({ ...prev, [character.id]: character }))
            setShowRecruit(false)
            setSelectedId(character.id)
          }}
        />
      )}

      {showArchived && (
        <ArchivedCharactersModal
          characters={archivedCharacters}
          onClose={() => setShowArchived(false)}
          onUnarchive={(character) => setCharacters((prev) => ({ ...prev, [character.id]: character }))}
        />
      )}
    </div>
  )
}

function ArchivedCharactersModal({
  characters,
  onClose,
  onUnarchive,
}: {
  characters: Character[]
  onClose: () => void
  onUnarchive: (character: Character) => void
}) {
  const [busyId, setBusyId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const unarchive = useCallback(
    (id: string) => {
      setBusyId(id)
      setError(null)
      unarchiveCharacter(id)
        .then(onUnarchive)
        .catch((err) => setError(String(err)))
        .finally(() => setBusyId(null))
    },
    [onUnarchive],
  )

  return (
    <div className="modal-scrim" onClick={onClose}>
      <div className="modal archived-modal" onClick={(e) => e.stopPropagation()}>
        <h2 className="pixel-face">Archived characters</h2>
        {error && <div className="banner banner-error">{error}</div>}
        {characters.length === 0 ? (
          <p className="archived-empty">No archived characters.</p>
        ) : (
          <ul className="archived-list">
            {characters.map((c) => (
              <li key={c.id} className="archived-row">
                <div className="archived-row-info">
                  <span className="archived-row-title">
                    {RACE_LABEL[c.race]} — {c.repo || c.territory.path || c.id}
                  </span>
                  <span className="archived-row-meta">
                    {ACTIVITY_LABEL[c.activity]}
                    {c.last_message ? ` · ${c.last_message.slice(0, 120)}` : ''}
                  </span>
                </div>
                <button type="button" disabled={busyId === c.id} onClick={() => unarchive(c.id)}>
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
