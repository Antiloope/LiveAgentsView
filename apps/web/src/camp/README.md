# Camp scene framework

Procedural **Craft Pixel camp** for the LiveAgentsView dashboard. Sprites and tiles are
defined in TypeScript and baked to Pixi textures at runtime — no hand-authored PNG
sheets, no generative image assets in the shipping path.

## Layout

```
camp/
  defs/     Palette, kits, parts/, items/, assembleKit
  bake/     Pixel buffer → ImageData → Pixi Texture (nearest-neighbor)
  scene/    CampCanvas + backdrop + fire + patrol
  lab/      Optional craft harness (unwired)
  index.ts  Public exports
```

## Locked product atoms (design pass)

- World: Craft Pixel — navy/gold ornate HUD + parchment Quest Ledger
- Type: Cinzel Decorative (title) + Pixelify Sans (HUD) + Cormorant Garamond (long read)
- Palette: navy night + gold trim (`CAMP_PALETTE`)
- No fake HP/MP under party stands
- Piloted-only characters; no Adopted/Hooks/Tailing kits path
- No tool-permission Approve/Deny chrome (auto-approve races)

## Kit assembly

`assembleKit` paints `KitLayerId` order (`shadow` → `body` → `clothes` → `head` → `weapon` → `fx`).
Race/class kits pick default items from `defs/items` (staff, hood, pauldron, …); overrides optional.
`bakeKitBuffer` is the shared Portrait + Pixi entry.

## Dependency

`pixi.js` only; code-split via Vite `manualChunks`.
