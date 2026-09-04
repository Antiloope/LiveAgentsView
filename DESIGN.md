---
name: LiveAgentsView
description: Craft Pixel — working agents leave on quest; everyone else returns to the fire.
colors:
  sky-top: "#070b14"
  sky-mid: "#0c1428"
  ground: "#1a2618"
  ground-dark: "#0e1610"
  frame: "#152038"
  frame-mid: "#243556"
  parchment: "#f0e2c0"
  parchment-dark: "#dfcba0"
  ink: "#1a140c"
  ink-soft: "#4a3a28"
  ink-invert: "#f0e8d0"
  gold: "#c5a059"
  gold-bright: "#e8c878"
  gold-deep: "#8a6828"
  focus: "#e8c878"
  danger: "#e5473b"
  bevel-hi: "#e8c878"
  bevel-lo: "#1a2744"
  waiting: "#e8a838"
  failed: "#a41f2f"
  working: "#3ec27a"
  ready: "#5b9bd8"
  approve-fill: "#c9432f"
typography:
  display:
    fontFamily: "Cinzel Decorative, Cormorant Garamond, serif"
    fontSize: "1.45rem"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "0.02em"
  label:
    fontFamily: "Pixelify Sans, sans-serif"
    fontSize: "0.82rem"
    fontWeight: 400
    lineHeight: 1.2
    letterSpacing: "0.03em"
  body:
    fontFamily: "Cormorant Garamond, Georgia, serif"
    fontSize: "1.05rem"
    fontWeight: 500
    lineHeight: 1.45
    letterSpacing: "normal"
  mono:
    fontFamily: "ui-monospace, SF Mono, monospace"
    fontSize: "0.78rem"
    fontWeight: 400
    lineHeight: 1.4
rounded:
  none: "0px"
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
    rounded: "{rounded.none}"
    padding: "0.42rem 0.95rem"
  button-primary-hover:
    backgroundColor: "{colors.gold-bright}"
  button-secondary:
    backgroundColor: "{colors.frame-mid}"
    textColor: "{colors.ink-invert}"
    padding: "0.42rem 0.95rem"
  quest-token:
    backgroundColor: "#fff8e8"
    textColor: "{colors.ink}"
    padding: "0.45rem"
  drawer-body:
    backgroundColor: "{colors.parchment}"
    textColor: "{colors.ink}"
---

# Design System: LiveAgentsView

## Overview

**Creative North Star: "Craft Pixel"**

LiveAgentsView stages coding agents as a party at camp, never as a status table. A session that is `working` leaves for a quest and appears only in the left **Quest Ledger** as a compact token. Every other activity — `waiting`, `failed`, `ready` — returns to camp: a Pixi night clearing with gold-framed chrome, a hard fire, blue/maroon tents, pines under an indigo sky, and procedural kits that patrol on a shared grid. Urgency is staging: urgent agents walk nearer the flame at full opacity with distinct WAITING/FAILED glyphs and ground rings; calm agents sit further back, dimmed. Select opens a resizable right drawer that never covers the party.

**Product constraints agents must not re-invent:** every character is **piloted** (launched by LiveAgentsView). There is no Adopted / Hooks / Tailing class. Tool-permission **Approve/Deny is intentionally absent** — races auto-approve; waiting means the character asked the *user* something, answered in chat / interrupt / stop. Do not score missing permission UI as a gap.

Chrome is **navy enamel + burnished gold** with ornate SVG corner flourishes and CSS noise/grain textures — not SNES wood bevels and not SaaS glass. Party art is code-baked pixel kits (`camp/defs` → Pixi textures / canvas portraits). **Cinzel Decorative** owns the product title; **Pixelify Sans** owns short HUD strings; **Cormorant Garamond** owns anything read at length. There are no fake HP/MP bars under party members.

**Key Characteristics:**
- Attention is staging: Quest Ledger = working; camp = everyone else
- Craft Pixel HUD: navy/gold frames, parchment ledger, ornate corners
- Night camp Pixi scene + procedural kits (faces readable across the room)
- Urgency by row + distinct WAITING/FAILED cues — never color alone
- Cinzel / Pixelify / Cormorant type split; no party HP/MP chrome

## Colors

Navy night (`sky-top` / `sky-mid` / `frame`) carries the HUD metal. Gold (`gold` / `gold-bright` / `gold-deep`) is the only chrome accent. Parchment is for the ledger, drawer body, and forms. Status hues stay semantic: waiting amber, failed crimson, working green, ready blue. Firelight in the scene is warm orange against cool indigo — contrast is the glance cue.

## Typography

| Role | Face | Use |
|------|------|-----|
| Display | Cinzel Decorative | Product title in the top bar |
| HUD | Pixelify Sans | Buttons, ledger header, stand flags, scene labels |
| Body | Cormorant Garamond | Drawer transcript, details, empty copy |
| Mono | system UI mono | Paths / code snippets only |

Never put Cinzel or Pixelify on long uncontrolled transcript strings.

## Layout

Top bar (crest + title + Recruit / Archived) · left Quest Ledger (`~200px`, parchment) · center Pixi camp in a gold ornate frame · optional right session drawer. Below `760px`: column stack; ledger becomes a short strip; drawer goes full width.

## Elevation & Depth

Depth is hard gold ledges, inset navy/gold frame rings, and crisp integer offsets — not soft Material elevation. Ornate SVG corners sit on panel corners. CSS noise overlays add metal/parchment grain without sprite sheets.

## Shapes

`--radius-sm` is `0px`. Panels use square gold borders with double inset (navy + warm metal). Primary buttons may use corner rivets. Kits are axis-aligned pixels (`image-rendering: pixelated` / nearest textures). Corner flourishes are SVG paths, not emoji.

## Components

### Buttons
- **Primary (`.pixel-btn`):** gold metal fill, ink text, Pixelify, rivet corners, hard drop.
- **Secondary / Archived:** navy fill, parchment text, gold border.

### Quest Token
Parchment inventory row on the ledger: mini kit portrait, repo name, purple ACTIVE pill for `working`.

### Camp Canvas
Pixi full-bleed night camp: indigo sky, moon/stars, pine line, blue + maroon tents, dirt ring, hard multi-rect fire. Urgent nearer fire; calm further back. Selected: gold double ground ring. WAITING/FAILED: activity ring + pixel glyph.

### Session Drawer
Fixed right panel: navy/gold head, parchment body, Cormorant transcript. Primary actions
are chat (answer the character), interrupt, and stop — not tool-permission Approve/Deny
(that control is out of product). Never a full-screen route that hides the party.

### Inputs
Square, ink-soft border, `#fff8e8` fill, Cormorant; gold focus outline.

## Do's and Don'ts

### Do
- Keep Cinzel on the title, Pixelify on short HUD, Cormorant on long read.
- Use navy/gold ornate frames and parchment for the Quest Ledger.
- Stage urgency with camp row + WAITING/FAILED glyphs — never color alone.
- Bake party art through the procedural kit path shared by Pixi and Portrait.
- Keep the session drawer as a side panel that leaves the party visible.

### Don't
- Revive HP/MP under party stands or quest tokens.
- Fall back to SNES wood bevels or classic blue Final Fantasy menu fill as the shipping chrome.
- Use Jersey 15 / Work Sans as the primary craft-pixel faces.
- Use soft blurred elevation shadows or rounded card language (`border-radius` stays 0).
- Replace the Pixi + kit baker path with photo backgrounds or generative image sheets as the shipping art.
- Let the drawer become a status table that replaces camp staging.
- Bring back Adopted / Hooks / Tailing as if they were current product surfaces.
- Treat missing tool-permission Approve/Deny as a design or critique failure.

## Next craft (proposed — see IDEA-13)

Componentize chrome (buttons, type roles, chat bubbles, icons, layout shells) and camp
kits as layered parts (base body by race, class overlays, armor/items) so each piece can
be improved in isolation. Not decided as a ship plan until agreed.
