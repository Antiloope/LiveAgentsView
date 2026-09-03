import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ClaudeClassId, CursorModelOption, PilotProvider, Session } from './types'
import { fetchBranches, fetchCursorModels, launchPilotedSession, pickDirectory } from './api'
import { ProviderRune, MapRune, ShieldRune, HoodRune, SatchelRune, SignpostRune, SparkRune, SearchRune } from './Glyphs'

interface ClaudeClass {
  id: ClaudeClassId
  name: string
  flavor: string
  tags: string[]
  depth: number
  speed: number
  arrives: string
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
    arrives: 'Dragonkin Warrior',
    Rune: ShieldRune,
  },
  {
    id: 'sonnet',
    name: 'SONNET',
    flavor: 'The all-rounder. Reliable on any road.',
    tags: ['Everyday features', 'Most sessions'],
    depth: 60,
    speed: 60,
    arrives: 'Elf Rogue',
    Rune: HoodRune,
  },
  {
    id: 'haiku',
    name: 'HAIKU',
    flavor: 'The quick scout. Fastest, lightest, first out.',
    tags: ['Quick fixes', 'Fast loops', 'Small chores'],
    depth: 30,
    speed: 90,
    arrives: 'Halfling (small)',
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
  onLaunched: (session: Session) => void
}

export default function RecruitPanel({ onCancel, onLaunched }: Props) {
  const [provider, setProvider] = useState<PilotProvider>('claude-code')
  const [claudeClass, setClaudeClass] = useState<ClaudeClassId>('opus')
  const [cursorModels, setCursorModels] = useState<CursorModelOption[]>([])
  const [cursorModelsError, setCursorModelsError] = useState<string | null>(null)
  const [cursorModel, setCursorModel] = useState('auto')
  const [filter, setFilter] = useState('')
  const [cwd, setCwd] = useState('')
  const [repoInfo, setRepoInfo] = useState<{ current: string; branches: string[] } | null>(null)
  const [branch, setBranch] = useState('')
  const [prompt, setPrompt] = useState('')
  const [pickerBusy, setPickerBusy] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Fetched once per panel open, on first switch to Cursor — the daemon
  // caches the underlying `agent --list-models` call itself, so this just
  // avoids re-fetching every time the user flips providers back and forth.
  useEffect(() => {
    if (provider !== 'cursor' || cursorModels.length > 0 || cursorModelsError) return
    fetchCursorModels()
      .then(setCursorModels)
      .catch((err) => setCursorModelsError(String(err)))
  }, [provider, cursorModels.length, cursorModelsError])

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
          setBranch(info.current)
        })
      })
      .catch((err) => setError(String(err)))
      .finally(() => setPickerBusy(false))
  }, [])

  const autoOption = cursorModels.find((m) => m.id === 'auto')
  const grouped = useMemo(() => {
    const q = filter.trim().toLowerCase()
    const byFamily = new Map<string, CursorModelOption[]>()
    for (const m of cursorModels) {
      if (m.id === 'auto') continue
      if (q && !m.id.toLowerCase().includes(q) && !m.label.toLowerCase().includes(q)) continue
      if (!byFamily.has(familyFor(m.id))) byFamily.set(familyFor(m.id), [])
      byFamily.get(familyFor(m.id))!.push(m)
    }
    return byFamily
  }, [cursorModels, filter])

  const model = provider === 'cursor' ? cursorModel : claudeClass
  // Cursor has no such thing as an idle process — every turn is its own
  // one-shot invocation with the prompt baked into its argv, so a Cursor
  // recruit needs something to do from the very first launch. Claude Code's
  // process just sits attached waiting on stdin, so its first message can
  // wait for the drawer.
  const needsPrompt = provider === 'cursor'
  const canSubmit = cwd.trim() !== '' && (!needsPrompt || prompt.trim() !== '') && !busy

  const submit = useCallback(() => {
    setBusy(true)
    setError(null)
    launchPilotedSession({ provider, cwd: cwd.trim(), branch: branch.trim(), model, prompt: prompt.trim() })
      .then(onLaunched)
      .catch((err) => setError(String(err)))
      .finally(() => setBusy(false))
  }, [provider, cwd, branch, model, prompt, onLaunched])

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
            <span className="field-label">Provider</span>
            <div className="provider-row">
              <button
                type="button"
                className={`provider-pick${provider === 'claude-code' ? ' on' : ''}`}
                onClick={() => setProvider('claude-code')}
              >
                <ProviderRune provider="claude-code" /> Claude Code
              </button>
              <button
                type="button"
                className={`provider-pick${provider === 'cursor' ? ' on' : ''}`}
                onClick={() => setProvider('cursor')}
              >
                <ProviderRune provider="cursor" /> Cursor
              </button>
            </div>
          </div>

          {provider === 'claude-code' ? (
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
                      <div className="cr-arrives">→ arrives a {c.arrives}</div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div>
              <span className="field-label">Model</span>
              {cursorModelsError && <div className="banner banner-error">{cursorModelsError}</div>}
              <button type="button" className={`auto-card${cursorModel === 'auto' ? ' on' : ''}`} onClick={() => setCursorModel('auto')}>
                <div className="cr-icon">
                  <SparkRune className="rune md" />
                </div>
                <div className="cr-body">
                  <div className="cr-top">
                    <span className="cr-name">AUTO</span>
                    <span className="auto-badge">RECOMMENDED</span>
                  </div>
                  <div className="cr-flavor">
                    {autoOption?.label ?? "Cursor picks the best model for the task automatically — its own default."}
                  </div>
                </div>
              </button>

              <div className="or-label">
                or choose a specific model — {cursorModels.length > 0 ? cursorModels.length - 1 : '…'} available right now
              </div>
              <div className="filter-wrap">
                <SearchRune />
                <input type="text" placeholder="Filter by name or id…" value={filter} onChange={(e) => setFilter(e.target.value)} />
              </div>
              <div className="cursor-list">
                {[...grouped.entries()].map(([fam, models]) => (
                  <div className="model-group" key={fam}>
                    <div className="model-group-head">
                      <span>{fam}</span>
                      <b>{models.length}</b>
                    </div>
                    {models.map((m) => (
                      <div
                        key={m.id}
                        role="button"
                        tabIndex={0}
                        className={`model-row${cursorModel === m.id ? ' on' : ''}`}
                        onClick={() => setCursorModel(m.id)}
                        onKeyDown={(e) => e.key === 'Enter' && setCursorModel(m.id)}
                      >
                        <span className="mname">{m.label}</span>
                        <span className="mid mono">{m.id}</span>
                      </div>
                    ))}
                  </div>
                ))}
                {cursorModels.length === 0 && !cursorModelsError && <div className="list-note">Loading Cursor's model list…</div>}
                {cursorModels.length > 0 && grouped.size === 0 && <div className="list-note">No models match "{filter}".</div>}
              </div>
            </div>
          )}

          <div>
            <span className="field-label">Territory</span>
            {cwd ? (
              <div className="chip">
                <MapRune />
                <div>
                  <div className="path mono">{cwd}</div>
                  <div className="meta">
                    {repoInfo ? (repoInfo.branches.length > 0 ? `git repo · ${repoInfo.branches.length} branches` : 'not a git repository') : ''}
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
          </div>

          {cwd && repoInfo && repoInfo.branches.length > 0 && (
            <div>
              <span className="field-label">Trail</span>
              <select className="trail" value={branch} onChange={(e) => setBranch(e.target.value)}>
                {repoInfo.branches.map((b) => (
                  <option key={b} value={b}>
                    {b}
                    {b === repoInfo.current ? ' (current)' : ''}
                  </option>
                ))}
              </select>
            </div>
          )}

          {needsPrompt && (
            <div>
              <span className="field-label">First message</span>
              <textarea
                className="recruit-prompt"
                rows={3}
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Cursor starts fresh each turn, so it needs something to do right away…"
              />
            </div>
          )}
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
