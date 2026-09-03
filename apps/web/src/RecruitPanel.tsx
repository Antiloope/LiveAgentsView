import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ClaudeClassId, Character, CursorClassOption, Race, TerritoryMode } from './types'
import { createCharacter, fetchBranches, fetchCursorClasses, pickDirectory } from './api'
import { RaceRune, MapRune, ShieldRune, HoodRune, SatchelRune, SignpostRune, SparkRune, SearchRune } from './Glyphs'

interface ClaudeClass {
  id: ClaudeClassId
  name: string
  flavor: string
  tags: string[]
  depth: number
  speed: number
  Rune: typeof ShieldRune
}

const CLAUDE_CLASSES: ClaudeClass[] = [
  {
    id: 'opus',
    name: 'OPUS',
    flavor: 'The heavy hitter. Most thorough, takes its time.',
    tags: ['Big refactors', 'Gnarly bugs', 'Unfamiliar code'],
    depth: 90,
    speed: 30,
    Rune: ShieldRune,
  },
  {
    id: 'sonnet',
    name: 'SONNET',
    flavor: 'The all-rounder. Reliable on any road.',
    tags: ['Everyday features', 'Most quests'],
    depth: 60,
    speed: 60,
    Rune: HoodRune,
  },
  {
    id: 'haiku',
    name: 'HAIKU',
    flavor: 'The quick scout. Fastest, lightest, first out.',
    tags: ['Quick fixes', 'Fast loops', 'Small chores'],
    depth: 30,
    speed: 90,
    Rune: SatchelRune,
  },
]

// Groups a cursor-agent model id by its own prefix family — read straight
// off the id the CLI itself returned, never a claim this daemon invented
// about the model.
function familyFor(id: string): string {
  if (id.startsWith('claude')) return 'Claude'
  if (id.startsWith('gpt') || id.startsWith('codex')) return 'GPT & Codex'
  if (id.startsWith('cursor-grok') || id.startsWith('grok')) return 'Grok'
  if (id.startsWith('gemini')) return 'Gemini'
  if (id.startsWith('composer')) return 'Composer'
  if (id.startsWith('kimi')) return 'Kimi'
  if (id.startsWith('glm')) return 'GLM'
  return 'Other'
}

interface Props {
  onCancel: () => void
  onRecruited: (character: Character) => void
}

export default function RecruitPanel({ onCancel, onRecruited }: Props) {
  const [race, setRace] = useState<Race>('claude-code')
  const [claudeClass, setClaudeClass] = useState<ClaudeClassId>('opus')
  const [cursorClasses, setCursorClasses] = useState<CursorClassOption[]>([])
  const [cursorClassesError, setCursorClassesError] = useState<string | null>(null)
  const [cursorClass, setCursorClass] = useState('auto')
  const [filter, setFilter] = useState('')
  const [territoryMode, setTerritoryMode] = useState<TerritoryMode>('own')
  const [cwd, setCwd] = useState('')
  const [repoInfo, setRepoInfo] = useState<{ isRepo: boolean; current: string; branches: string[] } | null>(null)
  const [branch, setBranch] = useState('')
  const [prompt, setPrompt] = useState('')
  const [pickerBusy, setPickerBusy] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Fetched once per panel open, on first switch to Cursor — the daemon
  // caches the underlying class-catalog call itself, so this just avoids
  // re-fetching every time the user flips races back and forth.
  useEffect(() => {
    if (race !== 'cursor' || cursorClasses.length > 0 || cursorClassesError) return
    fetchCursorClasses()
      .then(setCursorClasses)
      .catch((err) => setCursorClassesError(String(err)))
  }, [race, cursorClasses.length, cursorClassesError])

  const openMap = useCallback(() => {
    setPickerBusy(true)
    setError(null)
    pickDirectory()
      .then((path) => {
        if (!path) return
        setCwd(path)
        setBranch('')
        setRepoInfo(null)
        return fetchBranches(path).then((info) => {
          setRepoInfo(info)
          // Shared territory runs on whatever is already checked out, so
          // showing it here is informational. Own territory branches off
          // this repo into a new worktree — preselecting the branch that's
          // already checked out in the main one would only ever collide,
          // so it starts empty and falls back to an auto-generated name.
          if (territoryMode === 'shared') setBranch(info.current)
        })
      })
      .catch((err) => setError(String(err)))
      .finally(() => setPickerBusy(false))
  }, [territoryMode])

  const autoOption = cursorClasses.find((m) => m.id === 'auto')
  const grouped = useMemo(() => {
    const q = filter.trim().toLowerCase()
    const byFamily = new Map<string, CursorClassOption[]>()
    for (const m of cursorClasses) {
      if (m.id === 'auto') continue
      if (q && !m.id.toLowerCase().includes(q) && !m.label.toLowerCase().includes(q)) continue
      if (!byFamily.has(familyFor(m.id))) byFamily.set(familyFor(m.id), [])
      byFamily.get(familyFor(m.id))!.push(m)
    }
    return byFamily
  }, [cursorClasses, filter])

  const cls = race === 'cursor' ? cursorClass : claudeClass
  // Own territory needs a git repository to branch from; a plain directory
  // can only ever host a shared territory.
  const notARepo = territoryMode === 'own' && repoInfo !== null && !repoInfo.isRepo
  const canSubmit = cwd.trim() !== '' && !notARepo && !busy

  const submit = useCallback(() => {
    setBusy(true)
    setError(null)
    createCharacter({
      race,
      territoryMode,
      cwd: cwd.trim(),
      branch: territoryMode === 'own' ? branch.trim() : '',
      class: cls,
      prompt: prompt.trim(),
    })
      .then(onRecruited)
      .catch((err) => setError(String(err)))
      .finally(() => setBusy(false))
  }, [race, territoryMode, cwd, branch, cls, prompt, onRecruited])

  return (
    <div className="scrim recruit-scrim open" onClick={onCancel}>
      <aside className="recruit-panel" onClick={(e) => e.stopPropagation()}>
        <div className="recruit-head">
          <span className="pixel-face">New arrival</span>
          <button type="button" className="drawer-close" aria-label="Cancel" onClick={onCancel}>
            ✕
          </button>
        </div>
        <div className="recruit-body">
          {error && <div className="banner banner-error">{error}</div>}

          <div>
            <span className="field-label">Race</span>
            <div className="provider-row">
              <button type="button" className={`provider-pick${race === 'claude-code' ? ' on' : ''}`} onClick={() => setRace('claude-code')}>
                <RaceRune race="claude-code" /> Claude Code
              </button>
              <button type="button" className={`provider-pick${race === 'cursor' ? ' on' : ''}`} onClick={() => setRace('cursor')}>
                <RaceRune race="cursor" /> Cursor
              </button>
            </div>
          </div>

          {race === 'claude-code' ? (
            <div>
              <span className="field-label">Class</span>
              <div className="class-rows">
                {CLAUDE_CLASSES.map((c) => (
                  <button
                    type="button"
                    key={c.id}
                    className={`class-row${claudeClass === c.id ? ' on' : ''}`}
                    onClick={() => setClaudeClass(c.id)}
                  >
                    <div className="cr-icon">
                      <c.Rune className="rune md" />
                    </div>
                    <div className="cr-body">
                      <div className="cr-top">
                        <span className="cr-name">{c.name}</span>
                        <span className="cr-id mono">claude-{c.id}</span>
                      </div>
                      <div className="cr-flavor">{c.flavor}</div>
                      <div className="cr-tags">
                        {c.tags.map((t) => (
                          <span className="tag" key={t}>
                            {t}
                          </span>
                        ))}
                      </div>
                      <div className="cr-stats">
                        <div className="rp-stat">
                          <span>Depth</span>
                          <div className="rp-track">
                            <div className="rp-fill" style={{ width: `${c.depth}%` }} />
                          </div>
                        </div>
                        <div className="rp-stat">
                          <span>Speed</span>
                          <div className="rp-track speed">
                            <div className="rp-fill speed" style={{ width: `${c.speed}%` }} />
                          </div>
                        </div>
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div>
              <span className="field-label">Class</span>
              {cursorClassesError && <div className="banner banner-error">{cursorClassesError}</div>}
              <button type="button" className={`auto-card${cursorClass === 'auto' ? ' on' : ''}`} onClick={() => setCursorClass('auto')}>
                <div className="cr-icon">
                  <SparkRune className="rune md" />
                </div>
                <div className="cr-body">
                  <div className="cr-top">
                    <span className="cr-name">AUTO</span>
                    <span className="auto-badge">RECOMMENDED</span>
                  </div>
                  <div className="cr-flavor">
                    {autoOption?.label ?? "Cursor picks the best class for the task automatically — its own default."}
                  </div>
                </div>
              </button>

              <div className="or-label">
                or choose a specific class — {cursorClasses.length > 0 ? cursorClasses.length - 1 : '…'} available right now
              </div>
              <div className="filter-wrap">
                <SearchRune />
                <input type="text" placeholder="Filter by name or id…" value={filter} onChange={(e) => setFilter(e.target.value)} />
              </div>
              <div className="cursor-list">
                {[...grouped.entries()].map(([fam, classes]) => (
                  <div className="model-group" key={fam}>
                    <div className="model-group-head">
                      <span>{fam}</span>
                      <b>{classes.length}</b>
                    </div>
                    {classes.map((c) => (
                      <div
                        key={c.id}
                        role="button"
                        tabIndex={0}
                        className={`model-row${cursorClass === c.id ? ' on' : ''}`}
                        onClick={() => setCursorClass(c.id)}
                        onKeyDown={(e) => e.key === 'Enter' && setCursorClass(c.id)}
                      >
                        <span className="mname">{c.label}</span>
                        <span className="mid mono">{c.id}</span>
                      </div>
                    ))}
                  </div>
                ))}
                {cursorClasses.length === 0 && !cursorClassesError && <div className="list-note">Loading Cursor's class list…</div>}
                {cursorClasses.length > 0 && grouped.size === 0 && <div className="list-note">No classes match "{filter}".</div>}
              </div>
            </div>
          )}

          <div>
            <span className="field-label">Territory</span>
            <div className="provider-row">
              <button
                type="button"
                className={`provider-pick${territoryMode === 'own' ? ' on' : ''}`}
                onClick={() => setTerritoryMode('own')}
              >
                Own worktree
              </button>
              <button
                type="button"
                className={`provider-pick${territoryMode === 'shared' ? ' on' : ''}`}
                onClick={() => setTerritoryMode('shared')}
              >
                Shared directory
              </button>
            </div>
            <p className="territory-hint">
              {territoryMode === 'own'
                ? 'Works in its own git worktree, branched off the repo you pick below — your checkout is never touched.'
                : 'Works directly in the directory you pick below, exactly as it is — no git command is ever run against it.'}
            </p>
          </div>

          <div>
            {cwd ? (
              <div className="chip">
                <MapRune />
                <div>
                  <div className="path mono">{cwd}</div>
                  <div className="meta">
                    {repoInfo ? (repoInfo.isRepo ? `git repo · ${repoInfo.branches.length} branches` : 'not a git repository') : ''}
                  </div>
                </div>
                <button type="button" className="re" onClick={openMap}>
                  change
                </button>
              </div>
            ) : (
              <button type="button" className="field-btn" onClick={openMap} disabled={pickerBusy}>
                <MapRune /> {pickerBusy ? 'Opening Finder…' : 'Open the map…'}
              </button>
            )}
            {notARepo && (
              <p className="territory-hint warn">
                Not a git repository — own territory is unavailable here. Pick shared territory, or a different directory.
              </p>
            )}
          </div>

          {territoryMode === 'own' && cwd && repoInfo?.isRepo && (
            <div>
              <span className="field-label">Trail (branch)</span>
              <input
                className="trail"
                list="branch-suggestions"
                type="text"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder={`lav/${race === 'cursor' ? 'cursor' : 'claude'}-…`}
              />
              <datalist id="branch-suggestions">
                {repoInfo.branches.map((b) => (
                  <option key={b} value={b} />
                ))}
              </datalist>
            </div>
          )}

          <div>
            <span className="field-label">First message (optional)</span>
            <textarea
              className="recruit-prompt"
              rows={3}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Give it a quest now, or leave blank and talk to it once it's at camp…"
            />
          </div>
        </div>
        <div className="recruit-foot">
          <button type="button" className="horn-btn" disabled={!canSubmit} onClick={submit}>
            <SignpostRune /> {busy ? 'Recruiting…' : 'Recruit'}
          </button>
        </div>
      </aside>
    </div>
  )
}
