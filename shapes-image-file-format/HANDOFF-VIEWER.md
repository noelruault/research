# Handoff — build the viewer/editor

**For a fresh session.** Everything below is measured and committed. Read `HANDOFF.md` first for the format's state, then this for what to build next and why.

The research phase is over. The format is a real file that carries geometry, colour, alpha and a subject selection. **Nothing has ever opened one and used it.** That is the whole job now.

---

## 1. The one-paragraph version

An image format that stores a picture as ~1,200 flat-coloured regions instead of 300,000 pixels. It is a **poor image codec** (loses to WebP on all 24 Kodak images) and a **good structured-image format** (beats WebP + a region-map sidecar on all 24, by ~30%). The regions are addressable, have stable identity across re-encodes, and edits are O(regions). A portable detector now writes a subject selection into the file for ~100 bytes. The next thing to build is the tool that makes any of that visible.

---

## 2. What exists, verified

| piece | state | where |
|---|---|---|
| encoder — Potts/Mumford-Shah merge + Ising relaxation | ✅ | `code/lab` (`lab hd`) |
| container **SHPC v3** — walls, colour, alpha, selection | ✅ round-trips bit-exact | `code/lab/container.go` |
| **alpha**, per-region flat (mode 1) | ✅ silhouettes hold at 0.00% dissolution | `DESIGN-ALPHA.md` |
| **selection chunk** (mode 1 ids, mode 2 + confidence/provenance) | ✅ 96–168 B | report 44 |
| **portable detector** u2net, Apache-2.0 | ✅ 0.9505 vs Vision, 163–246 ms | report 44 |
| mask → region ids | ✅ `lab snap` | `code/lab/snap.go` |
| region adjacency graph, areas, nested scale-space | ✅ already computed by the encoder | `colorBytes2`'s `share[]`, `hdMarks` |
| decoder → **WASM** | ✅ **compiles today** | `GOOS=js GOARCH=wasm go build ./...` |
| pure-Go brotli for the browser | ✅ verified builds for `js/wasm` | `github.com/andybalholm/brotli` v1.2.2 |
| **canvas viewer** | ❌ nothing | — |
| **attribute painting UI** | ❌ nothing | — |
| **regions → puppet parts** | ❌ draft only | `sprites/.plans/image-to-rects.md` |

**The decoder already compiles to WASM and the last external-process call (brotli) has a pure-Go replacement.** That is the single most important fact for starting: the browser path is unblocked.

---

## 3. The idea that makes this more than a viewer

**The selection chunk generalises to per-region attributes.** An instance id is one byte per region. So is a bone id.

| attribute | status | what it enables |
|---|---|---|
| instance id | ✅ shipping | subject selection |
| **bone id** | ❌ | **the rig** |
| z-order | ❌ | layering |
| material tag | ❌ | "metal", "skin", "foliage" — palette swaps, physics |

**A rig is just another attribute channel**, and the format already has the slot. That is the bridge from background removal to animation.

### Why this could matter

`sprites` (`noelruault/sprites`) is a Zig puppet library whose defining constraint is **the art *is* primitives** — rotated rects and circles only, enforced at compile time, no textures, no atlas, no image decode, ever. It animates many rigs at 60 fps because there is no per-frame raster.

Our regions are exactly that shape of data. **A photo decomposed into ~1,200 flat regions is already a puppet's worth of parts.**

### Prior art, honestly

- **Meta, "Animating Childlike Drawings with 2.5D Character Rigs"** (Feb 2025) — single drawing → 2.5D rig → retarget 3D skeletal motion, real-time.
- **UniRig**, **Tripo** — auto-rigging for 3D from a single image.

All are **humanoid-shaped problems**, output **meshes or warps**, and are **tools, not formats**.

**Where we differ, all measured:**
1. **Output is shapes, not a mesh.** GPU-cheap primitives, no decode.
2. **Regions have stable identity.** A rig can reference region #4,211 and it survives re-encode — report 28 measured that a *re-derived* mask keeps only 24–40% of its boundaries.
3. **It is one file.** Art + partition + selection travel together, ~12 KB.

### The honest hard part

**u2net gives you subject-vs-background. It does not give you limbs.** Automatic part decomposition — knowing which regions form the arm — is research-grade and nothing here solves it. Do not promise it.

**But that is exactly where "ease things rather than invent" wins.** Rigging today means drawing meshes and painting vertex weights. With regions it becomes **clicking ~1,200 addressable pieces instead of 300,000 pixels**, with boundaries already exact — a 250× reduction in what the human manipulates, the same ratio measured for selection.

And the scale-space can **propose**: coarse marks (227 regions) give blobs that often *are* limbs, connectivity groups them, the human confirms. **Propose-and-confirm, not magic.**

---

## 4. Three concepts, ranked

**A — "Photo to puppet".** Drop an image → detector cuts the subject → regions become parts → paint bone ids → animates with `sprites`' existing gaits. The demo that sells it: *a photo of your dog, walking.*

**B — Region editor.** Open `.shpc`, click regions, recolour, adjust, save. No generational loss, exact edges. Useful, unremarkable, and **A needs it anyway**.

**C — Game asset pipeline.** Photo/art → shapes → Zig tables for `sprites`. Closes the loop with `zigquest`; narrow audience.

**A contains B. C is an exporter you add later.**

---

## 5. Three questions the owner has not yet answered

These change the architecture, so ask before building:

1. **Web-first, or does the Zig/`sprites` path matter equally?** Web-only is simpler; the `sprites` bridge is where the performance story lives.
2. **Is "photo of your dog, walking" the target demo**, or is static editing the product and animation the stretch?
3. **How much manual is acceptable?** Fully automatic rigging is not honest yet; propose-and-confirm is. Does that satisfy the vision, or must it be one-click?

---

## 6. Suggested order

1. **WASM decoder + canvas viewer.** Open a `.shpc`, draw regions, click one, highlight it. Unblocked today. This is the thing that has never existed.
2. **Region editing.** Recolour, alpha, toggle selection membership. O(regions), already how the data works.
3. **Attribute channel, generalised.** Make the selection chunk carry arbitrary per-region bytes, not just instance ids. Small format change, big capability.
4. **Ingest.** Detector in the loop for fresh JPEGs. **Note: a `.shpc` that already carries a selection needs no model at all** — the viewer is model-free, only ingest isn't.
5. **Bone ids + `sprites` bridge.** The animation payoff.

**Do not start with the model.** Steps 1–3 need no inference and no network.

---

## 7. Rules this project runs on — read before measuring anything

`PRINCIPLES.md` (what it's for), `METHODOLOGY.md` (how to measure and what confounds each metric), `PREREGISTRATION.md` (thresholds committed before runs), `06-corrections-and-falsifications.md` (**14 claims this study produced and then killed**).

Three that will bite hardest:

- **Never score a comparison on our own data structure.** Use a neutral referee.
- **Steelman the other side.** Falsifications #11 and #14 are both "we compared against a strawman", and #14 cost a published headline mid-session.
- **A flat number beside moving ones is a bug, not a finding.**

---

## 8. What is deliberately absent

Some detector-internals exploration was done during this session for **learning purposes only**. It produced no shippable artefact, is not in this repository, and must not be reconstructed here. What survives is the **benchmark comparison** — which detector matches which, and by how much — and that is in report 44 and `LICENSES.md`.

**Apple's model files are `.gitignore`d** (`shapes-image-file-format/weights/`, `*.espresso.*`). This repo pushes to GitHub; committing them would publish them. Vision remains usable as an optional fast path on macOS through its public API, because calling a system API is not redistribution.

**The shipping detector is u2net, Apache-2.0** — and `LICENSES.md` records exactly what was verified at source and what was not.

### Local-only notes

`shapes-image-file-format/private-notes/` is **gitignored** and holds what that exploration produced, kept on disk because the knowledge is useful and out of git because this repo is public. Its `README.md` explains each piece and why none of it shipped.

**One item there is promotable**: `private-notes/go-runtime/shpnet/` is a pure-Go inference runtime for u2net — **our own code on an Apache-2.0 model**, scoring IoU 0.9428 against `onnxruntime` with foreground fractions matching to 0.1%. It removes Python and `onnxruntime` from the stack entirely, which would make detection, decoding and editing **one Go binary in the browser**. It is unreviewed and slow (78 s/image, naïve convolution), so it was parked rather than committed. Reviewing and promoting it is a strong early move for the viewer work.
