---
title: Chat transcript rendering — markdown, collapsed reasoning and tool use, live quest indicator
slug: chat-transcript-rendering
status: validated
created: 2026-09-05
updated: 2026-09-05
next: none
chain: specify+implement+validate
---

# Spec: Chat transcript rendering

## Intent

The session drawer's transcript reads like a chat instead of a log: the
character's markdown renders as markdown and stays inside its bubble (today a
table is dumped as one raw string that overflows the drawer), reasoning and
tool use are present but collapsed behind a chevron, and while a character is
working the transcript's tail says so live instead of going silent until the
turn ends.

## Out of scope

- Streaming partial assistant text. Claude Code's `--include-partial-messages`
  is not turned on: an assistant message still appears when its block closes.
- Raw HTML inside the model's markdown. It is shown as text, never parsed
  into DOM.
- Syntax highlighting for fenced code. No highlighter dependency; code blocks
  are mono, scrollable, unpainted.
- Tool *results*. `tool_call/completed` stays skipped for Cursor and
  `tool_result` stays a user-line duplicate for Claude Code — only the call is
  shown, as today.
- A persisted quest object. The indicator is derived live from the character's
  activity and the tail of the transcript; nothing new is stored.
- The compose box, the drawer's layout, and its responsive behaviour.
- Markdown in what the *user* typed. A user bubble shows exactly the text that
  was sent, with its line breaks preserved.

## Already decided

- The transcript is flat and a quest is what the user asks in the chat, not a
  persisted object with its own state — `docs/03-decisions.md` (2026-09-03,
  character model). The working indicator therefore stores nothing.
- Approve/Deny is out of product: a tool call is something to *read*, never
  something to authorize — `docs/03-decisions.md`, `DESIGN.md`.
- Drawer body type is Cormorant Garamond, mono is for paths and code only, and
  Cinzel/Pixelify never go on long uncontrolled transcript strings —
  `DESIGN.md` "Typography". Rendered markdown keeps Cormorant for prose and
  system mono for code; the collapsed row's label is short HUD text.
- `--radius-sm` is `0px`, borders are square gold with hard offsets, no soft
  elevation — `DESIGN.md` "Shapes" / "Elevation & Depth". The disclosure row is
  a square parchment strip, not a rounded card.
- Showing the model's reasoning is new product surface; it was asked for
  directly by the maintainer in the 2026-09-05 session that opened this spec.
  It adds an event kind to an existing transcript, not a new dashboard surface.

## Verified live on this machine (not assumed)

Read out of the real transcripts under `~/.liveagentsview/pilot/*.jsonl` and
two direct `claude` probes, 2026-09-05:

- **cursor-agent streams reasoning as deltas.** `{"type":"thinking",
  "subtype":"delta","text":"…"}` — 84 of them in one real session — closed by a
  `{"type":"thinking","subtype":"completed"}` line that carries **no text**.
  The deltas are the only place the reasoning exists, so they must be
  accumulated and flushed as one event.
- **Claude Code redacts reasoning.** Its assistant messages do contain
  `{"type":"thinking", …}` content blocks, but every one of the four found
  across three real sessions has `"thinking": ""` with only a `signature`.
  Two fresh probes (`--verbose`, and again with `--include-partial-messages`)
  produced no thinking text either. So a Claude Code character will usually
  show no reasoning block at all — that is the CLI's behaviour, not a bug in
  this feature, and nothing may render an empty one.
- **Unknown provider lines reach the chat as raw JSON.** `rate_limit_event`
  (seen in three sessions) has no case in `handleClaudeLine`, so it falls to
  the `default:` passthrough and becomes a full-width system bubble of JSON.

## Open questions

None.

## Acceptance

- [ ] An assistant message containing a GFM table, a fenced code block, lists,
      headings, links, bold and inline code renders as formatted markdown
      inside the bubble.
- [ ] Nothing overflows: at the drawer's 320px minimum the transcript has no
      horizontal scrollbar, and a wide table or long code line scrolls inside
      its own container. Long unbroken strings (URLs, paths) wrap.
- [ ] `<script>` or any other raw HTML in the model's output is displayed as
      literal text, not parsed into the DOM.
- [ ] A user bubble shows the exact text sent, line breaks preserved, with no
      markdown interpretation.
- [ ] Every `tool_call` renders collapsed: a one-line row with a chevron, the
      tool name and its one-line summary; expanding reveals the full input.
      The row is a real `<button>` — keyboard focusable, `aria-expanded`, and
      the chevron rotates with the state.
- [ ] Reasoning arrives as its own persisted event (`kind: "thinking"`),
      replays identically on reload, and renders collapsed with the same
      disclosure. For Cursor, one event per thinking run — not one per delta.
- [ ] A redacted or empty thinking block produces no event and no empty bubble.
- [ ] A system event whose text is raw provider JSON (e.g. `rate_limit_event`)
      renders as a collapsed one-line row, not a wall of JSON. Short system
      text (`session started`, `turn interrupted`) still renders as the plain
      dashed centre line it is today.
- [ ] While the character's activity is `working`, a live indicator sits at the
      tail of the transcript naming what it is doing now — reasoning, a named
      tool, or just working — and disappears the moment the turn ends.
- [ ] The transcript still auto-scrolls to the bottom as events arrive,
      including when only the indicator changes.
- [ ] `docker compose` build of the web app succeeds and
      `scripts/check-doc-citations.sh` passes.

## How

**Frontend** (`apps/web`)

- Dependencies: `react-markdown` + `remark-gfm`, installed through Docker so
  the committed `package-lock.json` stays in sync without Node on the host.
  Raw HTML is off by default in react-markdown, which is what keeps provider
  output from reaching the DOM as markup.
- `src/ui/Markdown.tsx` / `.css` — the one place markdown becomes DOM.
  Parchment styling for headings, lists, blockquotes, tables and code;
  `overflow-x: auto` wrappers on `table` and `pre`; `overflow-wrap: anywhere`
  on prose so a long path cannot widen the bubble.
- `src/ui/Collapsible.tsx` / `.css` — the chevron disclosure row used by tool
  calls, reasoning and raw system lines. Collapsed by default, state local to
  the entry.
- `src/ui/ChatBubble.css` — `min-width: 0` and wrapping rules so a bubble can
  never push the flex column wider than the drawer.
- `src/SessionDrawer.tsx` — `TranscriptEntry` routes assistant text through
  `Markdown`, tool calls and reasoning through `Collapsible`, and JSON-looking
  system text through a collapsed row; a new indicator component renders at the
  tail while `character.activity === 'working'`, reading the last event to name
  the current step.
- `src/types.ts` — `'thinking'` joins `PilotEventKind`.

**Daemon** (`apps/lav`)

- `internal/pilot/pilot.go` — `EventThinking = "thinking"`, and a per-character
  buffer for Cursor's reasoning deltas alongside the other `mu`-guarded fields.
  `emit` already persists any kind as JSON in the existing `events` table, so
  there is no schema change and no migration.
- `internal/pilot/claude.go` — the content-block struct gains its `thinking`
  field; a `thinking` block emits an event only when the text is non-empty.
- `internal/pilot/cursor.go` — `thinking/delta` appends to the buffer;
  `thinking/completed` flushes it as one event; an `assistant`, `tool_call` or
  `result` line flushes first, so a run that never completes is not lost.

## Validation

2026-09-05, against the real daemon on this machine
(`scripts/lav-service-install.sh`), with two throwaway characters recruited
into own territory on this repo — one Cursor, one Claude Code — plus the
maintainer's own pre-existing Cursor transcript (the one whose unformatted
table started this spec). Both test characters were dismissed afterwards and
their worktrees removed.

Every acceptance item holds:

- **Markdown renders.** The Cursor character's answer came back with an H1, a
  three-column GFM table, a bullet list and a fenced block; all rendered.
  The maintainer's existing transcript — the raw-text table from the report —
  now renders as a real table with heading, code chips and zebra rows.
- **Nothing overflows.** Measured in the page, not eyeballed: with the drawer
  dragged to its 320px minimum, `transcript.scrollWidth - clientWidth` and the
  same on `body` were both `0`, while an 8-column table and a 200-character
  code line reported 2555px and 1371px of scroll *inside their own boxes*. A
  200-character path in prose wrapped instead of pushing the column.
- **HTML is text.** Asked for a literal `<b>` and `<script>`, the bubble's DOM
  came back as `&lt;b&gt;negrita&lt;/b&gt; y &lt;script&gt;alert(1)&lt;/script&gt;`
  with zero `<b>` and zero `<script>` elements in the transcript. A markdown
  link rendered as an `<a>` with `target="_blank"` and
  `rel="noreferrer noopener"`.
- **User text is verbatim.** The sent message renders in its own bubble with
  its line breaks and no markdown interpretation.
- **Tool calls collapse.** Cursor gave `→ shell` with its command on the row
  and `→ glob` with its pattern; Claude Code gave `→ Bash | wc -l *.md |
  sort -rn` from its flat `tool_input`. Clicking the row flipped
  `aria-expanded` to `true` and revealed the full JSON.
- **Reasoning arrives and collapses.** The Cursor character produced three
  `thinking` events — one per run, not one per delta — each rendering as a
  dashed *Reasoning* row with its first line as the summary, replayed the same
  way from history after a reload.
- **No empty reasoning.** The Claude Code character emitted no `thinking`
  event at all, as expected from the redacted blocks: no empty row appeared.
- **Raw provider lines collapse.** `rate_limit_event` rendered as a
  `provider line` row instead of the JSON wall it used to be. `session
  started` still renders as the plain dashed centre line.
- **Live indicator.** While working, the tail showed `On the quest…` and
  `Running glob…` / `Running Bash…`, and it disappeared the moment the turn
  ended.
- **Build and rules.** `tsc --noEmit` + `vite build` and `go build ./...`,
  `go vet`, `gofmt -l` all clean (run in Docker, nothing installed on the
  host); `scripts/check-doc-citations.sh` passes.

### Fixed in passing, outside this spec's scope

Live validation blanked the whole dashboard: `withKitColors`
(`apps/web/src/camp/defs/palette.ts`) iterated every field of a kit's color
record, so the `halfling-cleric` kit's `small: true` proportion flag landed in
the palette's color table and the next `parseHex` threw
`hex.replace is not a function`, taking the camp canvas — and with it the page
— down. One in three Claude Code characters hashes to that kit, which is why
it had not shown up before. Now only string values naming a real palette role
are applied. Unrelated to the transcript, but it made the feature
unverifiable, so it was fixed here rather than left in place.

### Noted, not fixed

Opening a drawer on a character that is *already* working can miss events
emitted in the gap between the history fetch's response and the SSE
subscription going live — six of eleven events rendered in one such open, all
eleven after a reload. That race predates this spec (`PilotChat`'s buffer only
covers events arriving *before* the fetch resolves) and belongs to whoever
takes the transcript stream next.

## Handoff

```
Spec: docs/sdd/specs/chat-transcript-rendering.md
Status: in-progress
Next: implement
```
