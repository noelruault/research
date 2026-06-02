# 02 — UI reverse-engineering (Track A fan-out)

A fan-out study of the most polished public Aseprite extensions, reverse-engineered
from source, to decide what makes the Pixelize dialog feel native. Three angles —
behreajj's Canvas pickers, K-Centroid + Magic Pencil dialog craft, and a broad
sweep — merged into one ADOPT / MAYBE / DISCARD catalogue. Per the brief, the
**discards are first-class results**: each is something we tested against and ruled
out, with the reason, so we don't reconsider it.

All snippets were read from source on `raw.githubusercontent.com`; repos and paths
are in §Sources. Where the agents disagreed or couldn't verify, it's flagged.

## Version floors (set the `minimumAsepriteVersion`)

From the `aseprite/api` `Changes.md` (corroborated across agents; the canvas gate is
the one disagreement — take the conservative value and feature-detect):

| Feature | API | Aseprite |
|---|---|---|
| `dlg:shades{}` | 9 | 1.2.17 |
| `dlg:modify{}` (reactive enable/visible) | 10 | 1.2.18 |
| `separator{ text= }` | 11 | 1.2.19 |
| `contributes.keys` | — | 1.2.35 |
| `dlg:canvas{}` + `GraphicsContext` + `app.theme` | 21–24 | 1.3-rc1 … 1.3.x |
| `dlg:tab{}` / `endtabs{}` | 26 | 1.3-rc7 |

**Canvas gate disagreement:** one agent reported canvas=API21, another canvas=API23
/ GraphicsContext=API24. Resolve by **feature-detecting** at runtime (gate on
`app.apiVersion` AND the presence of the canvas method) and degrade gracefully,
rather than hard-coding a number. A labeled-separator + reactive-visibility dialog
floors at **API 11 (1.2.19)**; adding the live canvas preview raises the floor to
**1.3**.

## ADOPT (high impact, low cost, makes it feel native)

1. **Labeled separators** `dlg:separator{ text="Resize:" }` — the single biggest
   "native" win for near-zero cost. Group: Resize / Palette / Mosaic / Preview.
2. **Reactive visibility/enable** `dlg:modify{ id=…, visible=/enabled= }` — show the
   custom-palette file picker only when Palette = "(custom file…)"; show build-map /
   pieces path fields only when their checkbox is on; grey out Apply until valid.
   Drive it from one function that hides-all-then-shows-active (see DISCARD-3 for why
   *not* the hand-enumerated style).
3. **`dlg:shades{ mode="pick", colors={…} }`** — the native palette swatch strip for
   showing the reduced palette and letting the user pick/inspect a color. Verified in
   JRiggles' Lospec Palette Importer. (A fully custom canvas swatch strip is the
   fallback only if `shades` is too limited — see behreajj pattern below.)
4. **Linked/locked resize** — a `check{id="lock"}` + width/height `number`s whose
   `onchange` cross-update via `modify{text=…}` (K-Centroid's exact pattern, §A1
   below). No native aspect-lock widget exists; this hand-rolled behavior is the
   norm. Adopt the behavior, with the fixes in DISCARD-1.
5. **`dlg:repaint()` wiring** — every input handler ends with `dlg:repaint()` to
   re-render. Simple and correct.
6. **Theme-matched canvas** — fill the preview background with
   `app.theme.color.window_face`, text in `app.theme.color.text` with a (+1,+1)
   shadow pass and a light/dark contrast flip; `ctx.antialias=false` for crisp
   pixels. Makes custom drawing indistinguishable from native UI.
7. **The behreajj preview trick** — build a small `Image` once, then
   `ctx:drawImage(img, srcRect, dstRect)` to scale it into the canvas; bulk-write
   pixels via `string.pack` + `image.bytes`, never per-pixel in `onpaint`. For
   Pixelize: render the reduced result at native size, let `drawImage` scale it.
8. **Icons + screenshots** in the repo/itch listing — large perceived-quality payoff,
   zero code risk; every polished example leads with them.

## MAYBE (situational)

- **Canvas live before/after preview** — the headline feature, but it gates on 1.3
  and needs the binary to run on a downscaled proxy fast enough to feel live (or a
  "Preview" button). High value, real cost. Phase 2 of the build plan.
- **Non-modal `show{ wait=false }` + position persistence** — lets the preview stay
  open while editing. But there are **verified API bugs**: bounds ↔ `autoscrollbars`
  reset each other (aseprite#4758), bounds-after-show only applies on mouse-move
  (#4757), reset-to-(0,0) on close (#3018). Adopt only with the Sprite-Analyzer
  save/restore-bounds idiom (§A3) and persist bounds yourself; never combine with
  `autoscrollbars`.
- **`contributes.keys`** — can bind a shortcut to *open* Pixelize, nothing in-dialog;
  one optional entry, ships a separate `.aseprite-keys` XML.
- **CI publish via `butler`** — itch.io is the real channel, but the butler action is
  game-oriented; a plain `gh release` upload may be simpler. Worth one-time setup,
  not load-bearing.

## DISCARD (tested against, ruled out — with the reason)

- **D1 — K-Centroid's 4-field px+% resize, copied wholesale.** Real smells: `decimals=4`
  shows ugly `66.6667`; asymmetric `math.min` clamps silently cap typed values with
  no feedback (type 200% → snaps to 100); the `onchange`-writes-sibling pattern
  relies on undocumented "modify doesn't re-fire onchange" behavior (loop-prone); a
  redundant OK-time re-clamp band-aids out-of-range live values. **Adopt the
  behavior, not the code:** single source of truth (px fields; % read-only or a
  single 1–100% slider), `decimals≤2`, and if upscaling is allowed the clamp logic
  must be redesigned. (The shipped file also has a misspelled `"Itterations"` label
  and dead debug comments — it's not the gold standard its reputation implies.)
- **D2 — Magic Pencil's close+reopen-to-resize dance.** The author himself flags it
  ("a 'phantom' dialog is left… breaks the onchange event"). It's a workaround for
  "you cannot resize a live dialog," not a pattern. For pure visibility changes,
  `dlg:modify{visible=}` alone is smoother and flicker-free. Only reopen-with-bounds
  when you genuinely must change window *size*.
- **D3 — the giant hand-enumerated `SelectMode` (≈20 chained `modify{visible=}`).**
  Doesn't scale: every new mode touches one monolith and a missed widget lingers
  stale. Use a data-driven `modeWidgets[mode] = {ids…}` + hide-all-then-show-active
  loop instead.
- **D4 — behreajj's color-science stack for "a few helpers."** Pulling
  `aseutilities.lua` (3,933 lines) + Lab/Rgb/Curve modules to get 4 bit-packing
  helpers is the wrong trade — copy the ~10-line `aseColorToHex`/`hexToAseColor`
  standalone. Also discard `graphBezier` (~500 lines for one widget), the
  in-`onpaint` recompute-everything pattern (cache the result image instead), and
  per-repaint `Image()` allocation for a full-size preview.
- **D5 — unbounded loops in event handlers** (behreajj's gamut `repeat…until` inside a
  mouse handler) and **shipped dead debug widgets** (`visible=false` Debug/Profile
  buttons). Never.
- **D6 — full custom-painted window (Attachment-System style).** Re-implements
  scrollbars/hit-testing/focus on one canvas — enormous surface, core-team-only
  maintenance. Pixelize needs *one* preview canvas inside otherwise-native widgets.
- **D7 — interactive paint-on-canvas (`onmousedown/move/keydown`)** beyond a simple
  swatch click — low payoff for a preview; a preview needs only `onpaint` +
  `repaint`. Wiring canvas focus/keyboard adds bugs without improving the flow.
- **D8 — `Dialog:file{}` preset import/export, tabs, custom-drawn "chain-link"
  controls** — scope creep / version-locked / low payoff for a handful-of-numbers
  dialog. Labeled separators beat tabs here.

## Two load-bearing snippets (verbatim, for the build)

### §A1 — linked resize (K-Centroid `extension.lua`)
Each field's `onchange` writes its sibling via `dialog:modify`, original size
captured once as a closure constant; `ratio` check force-syncs on tick.
```lua
:number{ id="width", label="Width:", text=tostring(width), decimals=0, focus=true,
  onchange=function()
    dialog:modify{ id="widthp", text=tostring(math.min(100, dialog.data.width/width*100)) }
    if dialog.data.ratio then
      dialog:modify{ id="heightp", text=tostring(math.min(100, dialog.data.widthp)) }
      dialog:modify{ id="height",  text=tostring(math.min(height, math.round(dialog.data.heightp/100*height))) }
    end
  end }
```

### §A3 — clean bounds save/restore (Sprite Analyzer)
`Refresh = Close + Create + Show`, bounds captured on Close, re-applied after
Create; an `isRefreshing` flag suppresses the `onclose` callback during the
internal close.
```lua
function Dlg:Close()   self.b = self.dialog and self.dialog.bounds or self.b; self.dialog:close() end
function Dlg:Refresh() self.isRefreshing=true; self:Close(); self:Create(); self:Show() end
-- in :Create(), after widgets: if self.b then local nb=self.dialog.bounds; nb.x=self.b.x; nb.y=self.b.y; self.dialog.bounds=nb end
-- ctor onclose: if not self.isRefreshing then self.onclose(self.b) end; self.isRefreshing=false
```
Always multiply explicit pixel bounds by `app.preferences.general["ui_scale"]` for HiDPI.

## Sources

- behreajj/AsepriteAddons — `dialogs/color/lchPicker.lua`, `support/canvasutilities.lua`, `support/aseutilities.lua`: https://github.com/behreajj/AsepriteAddons
- Astropulse/K-Centroid-Aseprite `extension.lua` (MIT): https://github.com/Astropulse/K-Centroid-Aseprite
- thkwznk/aseprite-scripts — Magic Pencil, Sprite Analyzer, Theme Preferences, On-Screen Controls: https://github.com/thkwznk/aseprite-scripts
- JRiggles — Lospec-Palette-Importer, Aseprite-Extension-Template, Aseprite-Extension-Updater: https://github.com/JRiggles
- aseprite official — Dialog/GraphicsContext/app.theme/Changes: https://github.com/aseprite/api ; Attachment-System: https://github.com/aseprite/Attachment-System
- Verified dialog bugs: aseprite/aseprite #4758, #4757, #3018, #3747
