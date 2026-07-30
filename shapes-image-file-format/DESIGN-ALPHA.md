# Design study — alpha, both ways

**Status: approach A is built and shipping; approach B is reserved and unbuilt.** Alpha now travels the whole pipeline — load → merge → container → decode — as SHPC v2 mode 1. See "A1b" below. This still documents both approaches so B can be picked up later with the reasoning intact. It supersedes the single-line "alpha is per-region, not per-pixel" call recorded in `PREREGISTRATION.md` Phase 1 item 1 — see Amendment 1 there.

Governed by [`PRINCIPLES.md`](PRINCIPLES.md), principle 7 in particular: **other formats inspire us, they do not set our bar.** PNG has an alpha channel because PNG is a grid of pixels and has nowhere else to put coverage. That is a consequence of its representation, not a specification we inherit. The question here is what *a format made of shapes* should do, asked from scratch.

## The thing that makes this different from every raster format

In a raster format, a hard silhouette is an alpha problem: the grid has no idea where the object ends, so coverage has to be transmitted per pixel.

**Here the silhouette is already a region boundary**, and boundaries are what this format stores — 79–96% of the bill at every operating point (`AUTORESEARCH.md`). A fully-transparent area is not a channel value. It is a region you do not draw.

That reframes the whole question:

- **Binary alpha may cost approximately nothing.** One flag per region, or the convention that region id 0 is void. Against a colour bill that is 3.7–21.0% of the file, per-region flags are noise.
- **The anti-aliased rim may not need to be stored at all.** In a raster, AA is a *resolution artifact* — the true silhouette is a curve somewhere between two pixel centres, and the encoder stores a coverage estimate because the curve was lost. We did not lose it. A decoder holding the boundary can compute coverage analytically at draw time, at whatever resolution it is drawing.

That second point is the interesting one, and it is **an argument, not a measurement** — see A4. If it holds, most of the "soft alpha" measured in real sprites is information we already have in a better form, and re-transmitting it as a channel would be paying twice.

## The two approaches

### A — Per-region flat alpha

One alpha value per region, alongside its colour.

**For.** Fits the existing model exactly — a region is already a flat colour, this makes it a flat RGBA. Cost is ~nregions values against a colour bill that is a minority of the file. The silhouette is exact and hard: no fringe, no matte halo, no colour bleeding from a background that was composited at encode time. Editing stays O(regions) — "make this region 50% transparent" is one number.

**Against.** Cannot represent variation *within* a region. Genuine translucency — glass, smoke, glow, a soft shadow — would have to be approximated by splitting into more regions, which costs boundary bits, the expensive part of the file.

**Open:** whether the merge would need to split regions on alpha the way it splits on colour, and what that costs. Unmeasured.

### B — Per-pixel alpha plane

Alpha as its own full-resolution channel, coded separately.

**For.** Represents anything. Soft shadows, gradients, volumetrics, whatever an artist draws. No approximation.

**Against.** It is a raster plane in a shape format — the thing this project exists to avoid. It reintroduces a per-pixel decode step for one channel, and the pixel-count-scaling cost the region representation was chosen to escape.

**A correction to an earlier overstatement.** `PREREGISTRATION.md` said per-pixel alpha "breaks the piecewise-constant model outright". That was too strong. Colour would stay piecewise-constant; only alpha would get a different representation. It is a hybrid, not a violation — a worse one to *justify*, but not a contradiction.

**Cheaper than it sounds, in the common case.** For a mostly-binary sprite the alpha plane is nearly all 0s and 255s, and its only structure is the silhouette — which the wall coder already codes for the colour partition. A per-pixel alpha plane would be re-coding a boundary the file already contains. That is the strongest argument against B and it is also, precisely, an argument for measuring it rather than assuming it.

### C — Both, selected per file

They are not exclusive. A mode field in the header:

| mode | meaning |
|---|---|
| 0 | no alpha — costs one byte, keeps v1's numbers intact |
| 1 | per-region flat alpha |
| 2 | per-pixel alpha plane |

**This is the lazy correct move and it answers "could we optionally add alpha?" — yes.** Reserve the mode field now; implement mode 1 first, because it covers the initial game-asset niche; leave 2 unimplemented until A3 says whether real art needs it. A reserved field costs a byte. A version bump later costs a compatibility story.

### D — Ideas worth recording, not pursued

- **Per-region alpha ramp.** A linear alpha gradient per region instead of a flat value. Directly reuses the parked P-08 affine-colour machinery (`code/lab/affine.go`), which was killed on numbers that have since moved by 28%. Would cover soft shadows without a full plane.
- **A second partition for alpha.** Run the merge on the alpha channel independently, giving alpha its own regions. Doubles the boundary bill in the worst case, but alpha boundaries and colour boundaries usually coincide, so the second partition might code almost free against the first as context.
- **Analytic coverage at draw time.** No alpha stored for the rim at all; the renderer derives it from boundary geometry. See A4.

## A1 is done, and it retracts the fix this document proposed

**Run it and look: [`A1-silhouette/`](A1-silhouette/), numbers in [`DESIGN-ALPHA-A1-data.txt`](DESIGN-ALPHA-A1-data.txt), verb `lab silhouette`.**

The merge dissolves the silhouette on all three sprites, at every mark — 16% to 62% of silhouette crossings. So the hazard is real. **But the cause is not the merge, and the fix proposed below is aimed at the wrong stage.**

| sprite | crossings | dissolved | **invisible** |
|---|---|---|---|
| ak74 (940 → 195 regions) | 234 | 36.75% → 62.39% | **76.07%** |
| bow (1,368 → 166 regions) | 694 | 37.32% → 52.02% | **99.28%** |
| pickaxe (160 regions) | 92 | 16.30% | **16.30%** |

"Invisible" counts crossings where the value `load()` hands the merge on the *opaque* side is already near-black, and therefore identical to the void.

**Dissolved ≤ invisible on every row, every sprite, every mark. On pickaxe they are exactly equal — 15 and 15.** The merge never dissolved a crossing it could actually see.

**The mechanism.** `load()` (`common.go:32`) discards alpha, and Go's PNG decoder returns **premultiplied** values. A transparent pixel arrives as `(0,0,0)` — black. A soft rim pixel arrives darkened toward black in proportion to its transparency. **A dark sprite on a transparent background becomes a dark shape on a black background**, and no merge rule can separate two colours that are equal.

It is visible in the panels: in `ak74-328regions.png` the orange wooden stock keeps its silhouette while every edge of the black gun body, grip and barrel is marked dissolved — same image, same merge, outcome decided purely by whether the edge colour differs from black. In `bow-476regions.png` the bowstring is gone along its entire length.

**The tell was bow's plateau.** Dissolution sits at exactly 259 crossings across three successive marks (476, 804, 1,368 regions). At 1,368 regions on a 3,420 px image the merge is barely merging anything, and 37% of the silhouette is still gone. A floor that deep is not something a merge is doing.

**What replaces the proposed fix.** Stop destroying alpha at load; carry it into the merge. Then a transparency edge *is* a difference, and the constraint below is either trivially satisfied or unnecessary — **A1 cannot distinguish those two, because no version of this merge has ever been handed an alpha channel.** That is the next question, not a settled answer.

## "Never merge across an alpha edge" — what that meant

> **Superseded by A1, kept for the reasoning.** The hazard described here is real and measured. The fix is not: it treats a symptom of an upstream loss. Read it as the hypothesis A1 tested.

Two separate things, and the phrasing ran them together.

**The hazard.** The merge decides what to join using colour SSE only. It cannot see alpha. So two neighbouring regions with similar colour but different alpha — a red object on a red background, where one is transparent — score as an ideal merge and get joined. The silhouette dissolves, and the file is now wrong in a way no fidelity metric would catch, because colour PSNR never looks at alpha.

**The fix**, for approach A: treat an alpha difference as an infinite merge cost, the same way the merge already treats non-adjacency. Regions on either side of a transparency edge never become one region.

**It is not about optionality.** Alpha being optional is mode 0 above, a separate decision. The constraint only exists when alpha is present.

**It is untested.** Nobody has run the merge on an image with alpha and looked at what happens. That is A1, and it is the cheapest item here.

## A1b — built, and the constraint turned out to be unnecessary

**Before and after, same sprites, same protocol** (`A1-silhouette/*-AFTER.png`):

| sprite | before (A1) | after (A1b) |
|---|---|---|
| ak74, 4 marks | 36.75% – 62.39% | **0.00% at every mark** |
| bow, 5 marks | 37.32% – 52.02% | **0.00%**, except 10.66% at the coarsest rung |
| pickaxe | 16.30% | **0.00%** |

**What changed.** `load()` now reads **NRGBA** instead of RGBA — straight alpha instead of premultiplied — so the authored colour survives and transparency lands in its own channel. The merge carries alpha as a **fourth SSE channel**, in both `runRD` and `relax`. `exactPartition` keys on alpha, so two pixels identical in RGB but differing in transparency can never start in one region.

**Opaque work is untouched, by construction rather than by luck.** A constant fourth channel contributes exactly zero to every `dSSE` term, and `Img.A` is nil for a source with no alpha. Verified against a pre-change binary built from a git worktree: **13 of 13 renders byte-identical** across a full scale-space, and locked by `TestAlphaOpaqueIsInert`.

**The residual 10.66% is a different thing from the old bug.** At bow's coarsest mark the merge *can* see the alpha difference and merges anyway, because the RD key judges that boundary not worth its bits at that rate. Before, dissolution hit a hard floor that no amount of fineness could clear — bow sat at 37.32% across three successive marks including 1,368 regions, where the merge is barely merging. **A floor means lost information; a single coarse rung means a priced decision.** The never-merge-across-alpha constraint stays available for extreme coarsening, where it would now be a deliberate override rather than a repair.

## SHPC v2 — what shipped

Mode 1 (per-region flat alpha), approach A. Round-trips bit-exactly on all three sprites, alpha included:

| sprite | regions | walls | colour | **alpha** | file | round trip |
|---|---|---|---|---|---|---|
| ak74 | 813 | 183 B | 1,263 B | **361 B** | 1,831 B | EXACT |
| bow | 753 | 227 B | 519 B | **540 B** | 1,310 B | EXACT |
| pickaxe | 83 | 61 B | 246 B | **17 B** | 345 B | EXACT |

Pickaxe's 17 B for 83 regions is approach A's argument made concrete — binary alpha costs essentially nothing. Bow, the softest of the three, pays the most.

**v1 files still decode** bit-exactly under the new decoder. On an opaque image v2 costs **816 B against v1's 815 B — exactly the one byte** of the mode field, measured, not estimated.

**Mode 2 is reserved and rejected by the decoder**, because A3 has not shown that real game art needs a per-pixel plane.

**The round-trip check now compares alpha on the same footing as colour.** Without that, an alpha plane decoding to garbage would still have printed EXACT — nothing else in the decoder reads it.

## Research items

None of these have registered thresholds yet. **They get pre-registered before they run**, per principle 2. Listed in the order they should happen.

**A1 — Does the merge actually dissolve silhouettes? — DONE, and it moved the fix upstream.** Yes on all three sprites (16–62% of crossings), but `dissolved ≤ invisible` everywhere: every dissolved crossing was one the merge received as black-on-black, because `load()` drops alpha and premultiplication turns transparency into black. The merge is correct; the pipeline loses the information before it. See the section above and `DESIGN-ALPHA-A1-data.txt`.

**A1b — Carry alpha into the merge, then re-run A1. — DONE. The constraint is not needed.** Alpha now runs load → merge → container → decode. Silhouette dissolution goes from 16–62% to **0.00% at every usable mark**; the only residual is 10.66% at bow's coarsest rung, where the merge can see the alpha difference and trades it away on rate-distortion grounds — a choice, not a loss. Opaque baselines are byte-identical. Data: `DESIGN-ALPHA-A1b-data.txt`, images in `A1-silhouette/*-AFTER.png`.

**A2 — What does per-region flat alpha cost?** Add alpha to the colour chunk, encode a sprite corpus, report the delta against the same file without alpha. Expected to be small; "expected" is not a number.

**A3 — How much real game art needs soft alpha?** The pilot below is n=3 and inconclusive by construction. Needs a real sprite corpus and a rim test wider than 8-connectivity, so a multi-pixel AA band is not miscounted as interior translucency.

**A4 — Can analytic coverage replace the stored rim?** Render a boundary with computed coverage, compare against the source's anti-aliased rim, report the error. If it is small, approach B loses most of its remaining justification and the format gets a property no raster codec has: resolution-independent edge quality. Note the dependency — this is adjacent to P-07 (curve-fitted boundaries) in `PARKED.md`, whose trigger has already fired and which is parked on cost.

**A5 — If A3 says soft alpha matters: per-pixel plane vs second partition vs per-region ramp.** Priced side by side on identical partitions, per the standing invariant. Only worth running if A3 fires.

## Pilot data for A3 — read the caveats

Full output and command: [`DESIGN-ALPHA-data.txt`](DESIGN-ALPHA-data.txt). Verb: `lab alphahist <sprite.png ...>`.

| sprite | pixels | soft% | rim | interior | interior% |
|---|---|---|---|---|---|
| ak74.png | 2,352 | 12.07% | 203 | 81 | 3.44% |
| bow.png | 3,420 | 18.25% | 564 | 60 | 1.75% |
| pickaxe.png | 400 | 0.00% | 0 | 0 | 0.00% |

**What this does not show.** Three sprites is not a corpus. All three are tiny (20×20 to 84×28), so the silhouette rim is a large share of the image purely from perimeter-to-area — the same art at 4× linear size would report far less soft%. And the "interior" column is an **upper bound**: the 8-neighbour test misses the inner row of a 2px anti-aliased band, counting it as translucency when it is still rim.

**What it does show.** Most soft alpha in these sprites is rim (71% and 90% of soft pixels), and one of three is perfectly binary. That is consistent with the design argument above and it is not evidence for it. A3 is still open.

The nine `verify/*.png` files in the same repo were also run and excluded: all are alpha 255 everywhere. They are screenshots of rendered output, not sprites. Recorded here so nobody re-runs them expecting data.
