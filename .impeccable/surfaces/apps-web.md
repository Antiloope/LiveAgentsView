---
version: 1
slug: "apps-web"
primary_target: "apps/web"
related_targets: []
---

## Scope and visitor mode

apps/web — the whole embedded dashboard. Mode: Operate. The visitor is the product's
own author (dogfooding), watching this surface on an always-visible second monitor
that today is occupied by Claude Code/Codex/Cursor terminal tabs and IDE windows.

## Audience, job, action/task, proof/content, constraints

Developer running several coding agents in parallel across repos/worktrees. Job: know
at a glance who needs attention, queue tasks, jump into a session. No real session data
exists yet as design reference — placeholder content only, not fabricated as real. Must
not read as generic SaaS admin (explicit constraint). Must not read as gamified/childish
(explicit constraint, still holds) — the world commits to a real game HUD, not a cute
skin on an admin list.

## Direction contract

THESIS: The player has a camp; every agent is sent out on a quest and only returns to
camp when it needs the player or has finished — the attention queue IS the scene, never
a status list.

OWN-WORLD: 16-bit medieval pixel art (code-drawn D&D party sprites, beveled wood-frame
panels, gold trim) paired with a modern legible sans (Nunito) for everything read at
length; a pixel display face (Pixelify Sans) is reserved for HUD chrome only — names,
state labels, buttons. RPG-style HP/mana bars (random placeholder until real token data
exists). Drawn pixel-block icons replace any unicode glyph.

STORY: Whoever is `working` stands in a left "Out on quests" sidebar. Whoever is
waiting/blocked/failed/done/idle is at camp, lit by a central fire — urgent front row,
calm back row dimmed. Clicking anyone opens a resizable (drag-handle) side chat drawer
without hiding the rest of the party.

FIRST VIEWPORT: Left quest sidebar (mini tokens) beside a camp scene — fire, tents,
banner, front/back rows scaling with party size; top bar with title and Recruit action.

FORM: Direction pinned directly by the user through two rounds of code-led mockups (no
image generation available) — chosen over the prior "garden roster" world by explicit
rejection, and over a plain battle-line composition by the user's own attention-queue
framing.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish
review, the verdict, DESIGN.md, and every shipping raster carrying its provenance.

## Memorable moment

The camp is quiet when nothing needs you and crowds toward the firelight the moment
something does — attention is legible from across the room, before reading a single
name.

## Unresolved decisions

Real token-derived HP/mana data (backend work, not this surface). Camp density at
20+ concurrent agents (wrap/scroll today, no pagination yet). Exact archetype pool
size (5 today) if more visual variety is wanted later.
