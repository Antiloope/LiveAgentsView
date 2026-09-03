---
name: LiveAgentsView
description: A 16-bit medieval camp where every coding agent is a party member — quietly out on quest, or lit by the campfire the moment it needs you.
colors:
  sky-top: "#2b1810"
  sky-mid: "#4a2818"
  ground: "#233b1e"
  ground-dark: "#152410"
  wood: "#7a4a26"
  wood-dark: "#452712"
  wood-light: "#b98a52"
  parchment: "#f2e2c4"
  parchment-dark: "#e0c99a"
  ink: "#2a1a0c"
  ink-soft: "#63492c"
  ink-invert: "#fbe9d4"
  gold: "#e2703a"
  gold-bright: "#f2955f"
  hp: "#c9432f"
  hp-track: "#431c14"
  mp: "#4a8f6a"
  mp-track: "#16281c"
  focus: "#f2955f"
  danger: "#e5473b"
typography:
  display:
    fontFamily: "Jersey 15, Work Sans, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 400
    lineHeight: 1
    letterSpacing: "0.03em"
  label:
    fontFamily: "Jersey 15, sans-serif"
    fontSize: "0.72rem"
    fontWeight: 400
    lineHeight: 1.2
    letterSpacing: "0.02em"
  body:
    fontFamily: "Work Sans, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
    fontSize: "0.9rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  mono:
    fontFamily: "ui-monospace, SF Mono, monospace"
    fontSize: "0.78rem"
    fontWeight: 400
    lineHeight: 1.4
rounded:
  none: "0px"
  sm: "6px"
spacing:
  xs: "0.3rem"
  sm: "0.55rem"
  md: "0.85rem"
  lg: "1.5rem"
  xl: "2.5rem"
components:
  button-primary:
    backgroundColor: "{colors.gold}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0.55rem 1.1rem"
  button-primary-hover:
    backgroundColor: "{colors.gold-bright}"
  button-secondary:
    backgroundColor: "{colors.parchment-dark}"
    textColor: "{colors.ink}"
    padding: "0.4rem 0.75rem"
  quest-token:
    backgroundColor: "rgba(255,255,255,0.04)"
    textColor: "{colors.ink-invert}"
    padding: "0.45rem"
  drawer-body:
    backgroundColor: "{colors.parchment}"
    textColor: "{colors.ink}"
---

# Design System: LiveAgentsView

## Overview

**Creative North Star: "The Attention Campfire"**

LiveAgentsView renders its agent roster as a D&D party at camp, not a status table. A session that is `working` stands out on a quest, off in a side sidebar as a compact token; every other state — waiting, blocked, failed, done, idle — pulls that agent back to camp, where it stands as a full pixel-art character lit by a shared campfire. Urgency reads as literal proximity to the fire: the front row (waiting/blocked/failed) sits bright and animated near the flame; the back row (done/idle) is dimmed and desaturated further back. Nothing is a badge count. Everything is staging.

The whole surface is 16-bit medieval pixel art — code-drawn SVG rects on an 8px grid, no image generation, no stock icon packs — paired with modern, legible type for anything actually read (chat bubbles, session details, form labels). A dedicated pixel display face is reserved strictly for HUD chrome: titles, state labels, buttons, flags. It never appears in body copy or transcript text, where legibility at length matters more than the bit-art bit.

The visual system commits to the game-HUD bit fully rather than treating it as decoration on an admin list: wood-and-parchment beveled panels with clipped corners, a torch-lit gradient sky, HP/mana bars under every character. Those bars are an explicit, disclosed placeholder — seeded-random per session id, not real telemetry, because no token-consumption field exists on the Session type yet. The system does not pretend otherwise anywhere in the UI or in this document.

**Key Characteristics:**
- A camp scene (fire, tents, banner, treeline) as the primary layout device, not a card grid
- Beveled parchment-and-wood chrome with clipped (not rounded) corners throughout
- Pixel display face for HUD labels only; a legible humanist sans for everything read at length
- Front-row/back-row brightness as the sole urgency signal — no numeric badges
- Drawn pixel-block icons and flat vector "runes" in place of any unicode/emoji glyph

## Colors

A dusk-to-firelight palette: warm ember-brown night sky and forest ground behind camp, warm wood and rust-trimmed parchment in front of it, with a small state-color set (green/amber/red/blue/gray) doing all status signaling. Where the original palette leaned on a cool violet night, every dark surface here is warmed toward brown and rust so nothing in the frame reads as moonlit or magical — the world is lit by the fire, not a spell.

### Primary
- **Ember Rust** (token name `gold`, `#e2703a`, bright variant `#f2955f`): the one recurring accent — buttons, focus rings, selected-state borders, banner flag, HP/MP bar highlights. Used sparingly against the wood/parchment field so it reads as "this is interactive or wants attention." Warmer and more saturated than the camp's structural wood, so it never blends into the frame.

### Secondary
- **Wood** (`#7a4a26`, dark `#452712`, light `#b98a52`): structural chrome — topbar, sidebar borders, drawer header, panel edges, scrollbar thumb. This is the frame the world sits inside, always dark-to-mid brown, never used for body surfaces.
- **Parchment** (`#f2e2c4`, dark `#e0c99a`): the one light surface in an otherwise dark-mode app — the session drawer body, the new-session modal, chat bubbles. Signals "this is a document/read surface," a visual register switch from the dark camp scene it opens over.

### Tertiary
- **Dusk Sky** (`sky-top` `#2b1810`, `sky-mid` `#4a2818`): the app's base background and the top of the camp-scene gradient. A warm ember-brown dusk, not a cool night — sets the torch-lit, campfire-forward mood everything else sits against.
- **Ground** (`#233b1e`, dark `#152410`): the turf half of the camp-scene gradient, giving the scene a horizon.

### Neutral
- **Ink** (`#2a1a0c`): body text on parchment surfaces (drawer, modal, bubbles).
- **Ink Soft** (`#63492c`): secondary/label text on parchment surfaces (detail-list terms, hints, bubble labels).
- **Ink Invert** (`#fbe9d4`): default body text color on the dark camp/app background.
- **Parchment Dark** (`#e0c99a`): muted/secondary text on dark surfaces (subtitles, mini-repo paths, empty-state copy).

State colors (not part of the primary/secondary/neutral roles, but load-bearing and reused across the sidebar, camp stands, status icons and the drawer's state pill): working `#3ec27a`, waiting `#f0a83c`, blocked `#e5473b`, done `#5b9bd8`, failed `#a41f2f`, idle `#8a8f85` — unchanged by the Ember Camp repaint, since they're a semantic system independent of the chrome palette. HP fill `#c9432f` on track `#431c14`; MP fill `#4a8f6a` (moved off the old blue to a muted sage green so nothing in the core palette reads as cool/blue anymore) on track `#16281c`.

### Named Rules
**The Firelight Proximity Rule.** Urgency is never a number or a badge — it is literal brightness and position. The front row near the campfire is full-color and animated (glow pulse, hovering status icon); the back row is `filter: brightness(0.8) saturate(0.85)` and sits further from the flame. If a new state needs to signal "needs you," it joins the front row; it does not get a red dot.

**The Parchment-Is-Reading Rule.** Dark wood/ember surfaces are for the camp scene and its chrome. The moment content needs to be read at length — chat, session details, forms — the surface switches to parchment (`#f2e2c4`) with dark ink text. This is the one legibility escape hatch in an otherwise dark, decorative palette.

## Typography

**Display/Label Font:** Jersey 15 (with Work Sans, sans-serif fallback)
**Body Font:** Work Sans (with -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif fallback)

**Character:** Jersey 15 carries the game-HUD identity — a chunky, jersey-numeral display face, single weight, unmistakably a scoreboard/HUD font — but it is confined to short strings (titles, state labels, button text, flags). Work Sans, a clean, warm grotesque sans, carries every sentence a user actually reads, so the world never trades comprehension for theme. This swaps out the earlier Pixelify Sans / Nunito pairing for a bolder, more authentically blocky display face and a slightly more neutral body face — the same chrome-only split, sharper execution.

### Hierarchy
- **Display** (400, 1.5rem, line-height 1, letter-spacing 0.03em, `.pixel-face`): the app title in the topbar (`LiveAgentsView`), rendered with a hard drop text-shadow (`2px 2px 0 rgba(0,0,0,0.5)`) for the pixel-art HUD feel. Jersey 15 ships one weight only, so this rule (and every `.pixel-face` use) pins `font-weight: 400` explicitly to avoid the browser synthesizing a fake bold.
- **Label** (400, 0.72rem, `.pixel-face`): every other piece of chrome text — sidebar header ("OUT ON QUESTS"), quest-token provider name, stand-flag provider name, drawer session name, state pill, primary buttons, modal heading. Always short (a few words), always pixel-faced.
- **Body** (400, 0.85–0.9rem, line-height 1.5): chat bubbles, drawer hints, session detail values, empty-state copy, modal inputs — anything that can run to a full sentence or more. Always Work Sans.
- **Mono/Small** (400, 0.78rem, ui-monospace/SF Mono): tool-call JSON payloads in the transcript, inline `<code>` in empty-state text.

### Named Rules
**The Chrome-Only Pixel Rule.** Jersey 15 is applied only via the `.pixel-face` class to fixed, short UI strings — titles, labels, button text, state pills. It never appears on body copy, transcript bubbles, form input values, or any text whose length isn't controlled by the design (that's Work Sans's job). This is the load-bearing accessibility decision that keeps the retro face from fighting legibility.

## Layout

The app is a fixed-height, non-scrolling shell (`height: 100dvh`, `overflow: hidden` on `.app`) with an internal two-pane row below the topbar: a fixed 200px "Out on quests" sidebar on the left, and a flexible camp `.scene` filling the rest. Below 760px, the row collapses to a column — the sidebar becomes a horizontal strip capped at 30vh above the scene, and the resizable drawer forces to full viewport width.

The camp scene itself is not a grid: it's a layered composition (sky gradient → stars → treeline → ground texture → tents/banner/campfire as absolutely-positioned decoration → two flex rows of party stands anchored to the bottom, wrapping and center-justified, `gap: 1.6rem`). The back row (done/idle) sits above the front row (waiting/blocked/failed) in the campfire's dimmer middle-distance; either row collapses to `display: none` when empty rather than leaving a gap.

The session drawer is a fixed-position right-hand panel, not a route or overlay that hides the camp — `min-width: 320px`, `max-width: 78vw`, user-resizable via a 10px drag handle and persisted to `localStorage`. It slides in over a pointer-events-none dim scrim, so clicking another party member behind it switches sessions directly without an intermediate close.

Spacing is not a strict token scale but a small set of reused rem values: `0.3rem`/`0.55rem` (tight, within a card), `0.85rem`–`1rem` (component padding, panel gaps), `1.5rem`–`2.5rem` (section/page padding).

## Elevation & Depth

The system is flat with hard-edged pixel-art depth cues, not soft shadow-based elevation. Depth reads through three device: (1) hard offset drop-shadows on sprites and floating UI (`drop-shadow(0 3px 2px rgba(0,0,0,0.5))` under party sprites, `0 4px 0 rgba(0,0,0,0.35)` under the topbar) that read as pixel-art ground shadows, not ambient blur; (2) inset bevel highlights on buttons and the modal (`inset 1px 1px 0 var(--gold-bright), inset -2px -2px 0 #a9501f`) simulating a beveled game-UI panel edge; (3) brightness/saturation reduction (not shadow) distinguishing the dimmed back row from the lit front row.

### Shadow Vocabulary
- **Sprite ground-shadow** (`filter: drop-shadow(0 3px 2px rgba(0,0,0,0.5))`): under every party sprite and drawer sprite, giving the flat SVG a sense of standing on ground.
- **Panel drop** (`box-shadow: 0 4px 0 rgba(0,0,0,0.35)`): hard, non-blurred offset under the topbar — a pixel-art panel edge, not a soft elevation shadow.
- **Bevel inset** (`inset 1px 1px 0 var(--gold-bright), inset -2px -2px 0 #a9501f`): on `.pixel-btn` and the modal, simulating a raised beveled game-UI surface.

### Named Rules
**The Hard-Edge Rule.** Where the system needs depth, offsets are crisp integers (`3px 2px`, `0 4px 0`) with zero or minimal blur, never a soft diffuse shadow. Soft ambient shadows would read as flat-modern UI, not pixel-art HUD.

## Shapes

Two distinct corner languages coexist by design, tied to material: wood/parchment HUD chrome uses a clipped octagonal bevel (`clip-path: polygon(5px 0, calc(100% - 5px) 0, 100% 5px, 100% calc(100% - 5px), calc(100% - 5px) 100%, 5px 100%, 0 calc(100% - 5px), 0 5px)`) on `.pixel-frame` and `.pixel-btn`, giving buttons and panels their small-corner-cut "game inventory slot" silhouette instead of a rounded corner. The stand-flag banner uses a smaller 3px version of the same cut with one asymmetric top edge. Everything else — cards, bubbles, inputs, the modal, the drawer — is unrounded (`border-radius: 0` implicitly, `--radius-sm: 6px` exists but is not applied anywhere observed in the shipped CSS) with a flat 2–4px solid border in wood or ink tones. Borders are always solid, never dashed, except the `.bubble.system` transcript entry, which intentionally uses a dashed border to mark itself as a system aside rather than dialogue.

Party sprites and status icons are drawn entirely from axis-aligned `<rect>` blocks on an 8px (sprite) or 2-unit (status icon) grid with `shapeRendering="crispEdges"` / `image-rendering: pixelated` — no curves, no anti-aliasing, anywhere in the character art.

## Components

### Buttons
- **Shape:** clipped octagonal bevel (`.pixel-frame` polygon cut, corners cut 5px)
- **Primary:** `.pixel-btn` — rust background (`#e2703a`), ink text, 3px solid dark-wood border, inset bevel highlight, Jersey 15 label, `0.55rem 1.1rem` padding. Used for the topbar "Recruit session" action, the modal's "Launch" submit, chat "Send".
- **Hover / Focus:** hover brightens to `--gold-bright` (`#f2955f`); active nudges `translateY(1px)` (a physical "pressed" cue, not an opacity/shadow change); disabled drops to `opacity: 0.55` with `cursor: not-allowed`. Global `:focus-visible` gets a 2px rust outline with 2px offset on every focusable element, not just buttons.
- **Secondary / Ghost:** plain-bordered buttons without the `.pixel-btn` class (cancel, interrupt, resume, copy-path, open-terminal) — 2px ink-soft or wood-dark border, parchment-dark or transparent background, Work Sans body text, no bevel or clip-path. The danger variant (Cancel in the pilot chat) swaps the border and text to `--danger` (`#e5473b`).

### Cards / Containers ("Stands" and "Tokens")
- **Party Stand** (camp, full character): no background/border of its own — a transparent button wrapping a floating provider flag, a state-colored radial glow, the pixel sprite with a bobbing status icon overlay, and twin HP/MP bars. Selection and attention states are carried entirely by the child `.stand-flag`'s border color (default wood-dark → gold on `needs-attention` → gold-bright with a glow ring on `selected`), not by a border on the stand itself.
- **Quest Token** (sidebar, mini card): a bordered rectangle (`rgba(255,255,255,0.04)` background, 2px wood-dark border), brightening and gold-bordering on hover/selected — the one component that behaves like a conventional list item.
- **Corner Style:** quest tokens and drawer/modal containers are square (no radius); the stand-flag banner uses the small clipped-bevel cut.
- **Shadow Strategy:** see Elevation & Depth — drop-shadow under sprites, no shadow on the token/stand containers themselves.

### Inputs / Fields
- **Style:** 2px solid `--ink-soft` border, `#fff8e8` (warm off-white) background, square corners, `0.5rem 0.6rem` padding, Work Sans inherited font — used identically across the modal form and the chat compose textarea.
- **Focus:** the shared global rust `:focus-visible` outline; no per-input glow or border-color shift beyond that.
- **Disabled:** compose textarea placeholder swaps to explain why sending is unavailable ("No live process — resume to keep chatting") rather than just graying out silently.

### Navigation (Quest Sidebar)
- Fixed 200px dark warm-brown gradient panel (`linear-gradient(180deg, #3a2415, #241407)`), Jersey 15 header label with a rust session count, vertically stacked quest tokens with `0.55rem` gaps. On narrow viewports it collapses to a horizontal strip above the scene, capped at 30vh, scrollable.

### Session Drawer (signature component)
The resizable, non-modal side panel that replaced an earlier full-page chat view — it never covers the camp behind it. A wood header bar (sprite + name + repo + state pill) sits above either a live pilot chat (driver-fidelity sessions: transcript bubbles for user/assistant/tool/permission/error/system, a compose bar, and interrupt/cancel/resume controls) or a read-only details panel (hooks/tailing-fidelity sessions: a definition list plus "open terminal"/"copy path" actions). The 10px drag handle on its left edge turns gold while dragging and persists width to `localStorage`. Permission-request bubbles are the one place a hard state color (`--hp` red) is used for a button background, marking Approve as the weighty action.

## Do's and Don'ts

### Do:
- **Do** confine Jersey 15 to `.pixel-face` chrome strings only (titles, labels, buttons, state pills), always at `font-weight: 400` since the face ships one weight; use Work Sans for anything read at length.
- **Do** use the clipped octagonal bevel (`clip-path` polygon) for wood/parchment HUD chrome — buttons, frames, the stand-flag banner.
- **Do** signal urgency through front-row/back-row position and brightness (`filter: brightness(0.8) saturate(0.85)`), never through a badge, counter, or dot.
- **Do** disclose HP/MP bars as placeholder wherever they appear or are discussed — they are seeded-random per session id, not derived from real token/context data.
- **Do** keep dark wood/ember surfaces for the camp scene and switch to parchment the moment content must be read at length (chat, details, forms).
- **Do** draw every icon (provider runes, status glyphs) as flat SVG shapes or 2-unit pixel-block cells in the same stroke language as the party sprites — never a unicode/emoji glyph or an external icon font.

### Don't:
- **Don't** apply Jersey 15 to body copy, transcript text, or any string whose length isn't controlled by the design — it breaks legibility at length.
- **Don't** let Jersey 15 fall back to a browser-synthesized bold — the face has one weight, so `font-weight` on any `.pixel-face` context must stay `400`.
- **Don't** round corners on cards, inputs, the drawer, or the modal; square corners (or the specific clipped-bevel cut) are the only two shapes in this system.
- **Don't** use soft, blurred ambient shadows; depth reads through hard-offset drop-shadows and inset bevels only.
- **Don't** let the resizable session drawer cover or hide the rest of the party — it is a side panel by design, not a route or full-screen takeover.
- **Don't** treat the HP/mana bars as, or extend them to look like, real telemetry until a token-consumption field actually exists on the Session type.
