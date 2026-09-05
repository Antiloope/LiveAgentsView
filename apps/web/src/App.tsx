import { useEffect, useMemo, useState, useCallback } from 'react'
import type { Character } from './types'
import { fetchCharacters, subscribeToCharacters, unarchiveCharacter } from './api'
import { NEEDS_ATTENTION, RACE_LABEL, ACTIVITY_LABEL } from './sprites'
import QuestToken from './QuestToken'
import SessionDrawer from './SessionDrawer'
import RecruitPanel from './RecruitPanel'
import { CampCanvas } from './camp'
import { reviewModeEnabled, reviewPartyForQuery } from './camp/scene/reviewParty'
import { TopBar, QuestLedger, SceneFrame, Button, HudLabel } from './ui'

export default function App() {
  const [characters, setCharacters] = useState<Record<string, Character>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showRecruit, setShowRecruit] = useState(false)
  const [showArchived, setShowArchived] = useState(false)

  useEffect(() => {
    if (reviewModeEnabled()) {
      const byId: Record<string, Character> = {}
      for (const c of reviewPartyForQuery()) byId[c.id] = c
      setCharacters(byId)
      setLoading(false)
      return
    }

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

  return (
    <div className="app">
      <TopBar
        characterCount={total}
        archivedCount={archivedCharacters.length}
        onRecruit={() => setShowRecruit(true)}
        onShowArchived={() => setShowArchived(true)}
      />

      {error && <div className="banner banner-error">Could not load characters: {error}</div>}

      {!loading && total === 0 && !error && (
        <div className="empty">No characters yet. Recruit one above to bring it to camp.</div>
      )}

      <div className="layout-row">
        <QuestLedger count={questCharacters.length}>
          {loading ? (
            <div className="sidebar-empty">Loading…</div>
          ) : questCharacters.length === 0 ? (
            <div className="sidebar-empty">No one is out right now.</div>
          ) : (
            questCharacters.map((c) => (
              <QuestToken key={c.id} character={c} selected={c.id === selectedId} onSelect={selectCharacter} />
            ))
          )}
        </QuestLedger>

        <SceneFrame>
          <CampCanvas
            className="camp-pixi"
            figures={[
              ...calmCamp.map((c) => ({ character: c, calm: true as const })),
              ...urgentCamp.map((c) => ({ character: c, calm: false as const })),
            ]}
            selectedId={selectedId}
            onSelect={selectCharacter}
          />
          {!loading && urgentCamp.length === 0 && calmCamp.length === 0 && (
            <div className="empty-camp">Camp is quiet — everyone's out on a quest.</div>
          )}
        </SceneFrame>
      </div>

      <SessionDrawer
        character={selectedCharacter}
        onClose={() => setSelectedId(null)}
        onCharacterUpdate={(c) => setCharacters((prev) => ({ ...prev, [c.id]: c }))}
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
        <HudLabel as="h2">Archived characters</HudLabel>
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
                <Button type="button" disabled={busyId === c.id} onClick={() => unarchive(c.id)}>
                  Unarchive
                </Button>
              </li>
            ))}
          </ul>
        )}
        <div className="new-session-actions">
          <Button variant="secondary" type="button" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>
  )
}
