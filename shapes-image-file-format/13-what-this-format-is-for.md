# 13 — What this format is for, and what it must fix to deserve it

**Every report before this one asks "is it smaller than WebP?"** That question assumed the products are comparable. They are not: WebP delivers pixels, this delivers pixels *and* a segmentation. This report states the capabilities honestly, prices them where the record allows, names the weaknesses that block them, and ranks the work by how much value it unlocks rather than how many bytes it saves.

Nothing here is a new measurement. Every number is drawn from reports 01–12, and anything unmeasured is labelled **UNMEASURED** rather than argued.

## The baseline was wrong

The comparison that matches this product is not WebP. It is **WebP plus a region-map sidecar**, because that is what a consumer would have to assemble to get the same capabilities.

| at 28.5 dB, 3840×2160 | bytes | has structure? |
|---|---|---|
| WebP alone | 137,033 B | no |
| **WebP + a region map** (137,033 + the 121,047 B wall bill) | **258,080 B** | yes |
| **shape coder** | **153,190 B** | yes — the structure *is* the image |

**Against the baseline that matches the product, the shape coder is 41% smaller.** Raster-plus-sidecar pays for boundaries *and* every pixel; the shape coder pays for boundaries and derives the pixels from them. The geometry stops being overhead and becomes load-bearing.

> **Caveat, and it is not small.** The 121,047 B sidecar figure is *our own* wall coder, and report 06 #12 showed that coster is not decodable. A real sidecar would also use a purpose-built region-map codec, not ours. This comparison has **never been run properly** and is priority P1 below. Treat the 41% as a strong hypothesis, not a result.

## Strengths, with the number that supports each

**1. The region adjacency graph already exists.** `colorBytes2` builds `share[]` — boundary-length-weighted adjacency between every pair of touching regions — because the *compressor* needs it. Capability inherits it for free. Colour-proximity selection, similarity merging, flood operations are graph queries, not image processing.

**2. Edits are O(regions), not O(pixels).** At the 11,121-region mark that is 11,121 values against 8,294,400 pixels — **~750× less work** — and the edit is *exact*: no edge bleed, no resampling halo, no anti-aliasing artifact, because the boundary is stored rather than re-inferred.

**3. No generational loss.** Re-encoding a WebP degrades the whole image. Here an untouched region's colour is a number that did not change. Round-trip editing is lossless outside the edit.

**4. Bounded per-pixel error.** Piecewise-constant regions make max |Δ| knowable and enforceable per region. DCT ringing is not locally boundable. "No pixel is wrong by more than N" is a guarantee no transform codec offers at any bitrate.

**5. The level-of-detail hierarchy is already built.** The merge produces nested partitions — 227 → 1,383 → 11,121 → 96,359 → … Truncate anywhere and the result is a valid coarser image. Report 04 killed this as a *bytes* argument (6.37 dB behind JPEG-2000); as a *capability* it is untouched and already implemented in `hdMarks`.

**6. Deterministic, shared segmentation — and this one survives the killed list.** Report 04 #2 killed "let the decoder regrow the regions" as an *information* argument: anything derivable from decoded pixels is a context model bounded by `H(X | causal past)`. Airtight for bytes, and **silent on capability**. If every client re-segments, every client gets a different segmentation — different library, version, thresholds, seeds. Shipping the partition means region #4,211 is the same object on every device forever. That is authored intent, and nothing in twelve reports touches it.

**7. It degrades gracefully across scale.** Encoded at 960×540 and upscaled to 3840×2160, the shape coder scores **24.59 dB at 20,618 B against WebP's 24.54 dB at 20,066 B** (report 06 #11) — a tie at the bottom of the rate axis.

**8. Immediate-mode drawing, no decode step.** Regions are polygons and colours; they go to a GPU directly. There is no entropy-decode-then-IDCT-then-upload pipeline.

## Applications, by domain

Marked **[needs P0]** where the claim depends on the regions being *semantically* meaningful — which has never been measured.

### Photo editing

- **Background removal and subject selection** become graph traversal instead of inference. The mask is not computed, it is *read*. **[needs P0]**
- **Selective recolour by colour proximity** — "every region within ΔE of this blue" is a query over `share[]` plus a colour distance. Exact edges, no halo, no feathering artifacts.
- **Non-destructive local adjustment.** Adjust one region; every other region is bit-identical afterwards. No layer stack required — the format *is* the layer stack.
- **Colour-blindness remapping.** Shift specific regions to a distinguishable palette, exactly, without touching luminance structure. Currently impossible without segmentation.
- **Print and screen-printing separations.** Posterization into flat regions is a *defect* for photography and a *feature* here: regions map onto spot colours directly.

### Animation

- **Cutout / puppet animation from a photograph.** ~1,700 named regions, each independently transformable. This is the "shapes" pitch at its strongest, and it is what `pixelize` and `sprites` consume.
- **Per-region tweening.** Interpolate colour or boundary between two keyframes; identity is stable because ids are transmitted, not recomputed.
- **Per-frame cost is O(regions).** No decode per frame.

### Gaming

- **Semantic LOD.** The nested hierarchy gives level-of-detail that follows *objects*, not a mip pyramid that follows a grid. Distant object → coarser partition, same file.
- **Palette swaps without extra textures.** Team colours, damage states, seasonal variants — change region colours, ship one asset.
- **Deterministic across clients.** Bounded error plus a transmitted partition means every client renders identically, which lockstep simulations require and lossy textures cannot promise.
- **Truncatable streaming.** Ship coarse regions first, refine as bandwidth allows, never a broken frame.

### Photography and archival

- **Bounded-error archival** for scientific, medical and forensic use, where "visually lossless" is not an acceptable guarantee but "no pixel off by more than N" is.
- **Progressive transmission** over constrained links, with a valid image at every truncation point.

### Analysis and accessibility

- **Per-region contrast auditing** for WCAG — region adjacency plus colour gives contrast ratios between *actual adjacent areas*, not sampled pixels.
- **Structural description.** A region graph with areas, colours and adjacencies is a far better input to automatic description than a pixel grid. **[needs P0]**

## Weaknesses, ranked by how much they block the above

**W1 — Nobody has ever checked whether the regions are semantically meaningful. UNMEASURED.** This is the single biggest hole in the entire programme. Every capability marked [needs P0] rests on regions landing on *objects*. Report 04 found flat interiors band badly on smooth sky, which hints that regions may be following *illumination gradients* rather than object boundaries. A segmentation that splits the sky into twelve bands and merges a person with the wall behind them is useless for "select the person", regardless of how cheaply it codes. Twelve reports measured whether regions are *cheap*. None measured whether they are *right*.

**W2 — The capability band is narrow, and it is not where fidelity is good.** The shape story holds from ~227 to ~11,000 regions = **21.99 to 28.5 dB**. At 4K lossless the partition degenerates to **1.305 px/region** — 6.36M "regions", no shapes at all, just a raster coder with extra steps. The honest pitch is *structure in exchange for mid fidelity*, never *structure for free*.

**W3 — Still behind on bytes, though far less than it was.** At 28.7 dB, +19.3% over WebP became **+8.3%** after report 09's interleave, and RCT should take it to roughly **+4.7%** (report 11). Higher up the axis it is +32.7% at 30.0 dB and +48.6% at 31.5 dB. **AVIF is 30–50% ahead everywhere** and is not a target. For the capability pitch the requirement is not "smaller than WebP" — it is "not *meaningfully larger* than WebP", and that is now within reach at the operating point that matters.

**W4 — Not resolution-independent.** Boundaries are crack edges on a pixel lattice. Report 06 #4 already killed "renders at 8K for the same bytes" as a baseline error. It upscales *gracefully* (strength 7), which is not the same thing. True resolution independence needs curve-fitted boundaries, and nobody has priced that.

**W5 — The wall coder is not decodable.** Report 06 #12: one context tap reads a bit that has not been coded. Legality costs **+3.4% to +12.7%** of the wall bill, and the record still carries the illegal figures. Every capability claim that quotes a size inherits this.

**W6 — There is no container and no bitstream.** Every byte figure in twelve reports is an idealised adaptive-arithmetic cross-entropy with no header, no framing, no error resilience. Real files are bigger. A capability pitch requires an actual format, and none exists.

**W7 — One photograph.** Every number in the record comes from the same macOS Sierra wallpaper. No corpus, no BD-rate, no content diversity. Nothing here can be claimed to generalise.

**W8 — Encoder cost is unmeasured and appears large.** The 4K scale-space merge is expensive enough that agents recovered partitions from rendered PNGs to avoid re-running it. If encoding a 4K image takes minutes, entire application classes are excluded. **UNMEASURED.**

## Priorities — ranked by value unlocked, not bytes saved

**P0 — Measure segmentation quality.** Blocks W1 and most of the applications. Do the regions follow objects or illumination? Cheapest decisive version: take the 1,383 and 11,121-region partitions, overlay them on the source, and measure boundary recall against a human or SAM-derived object segmentation; plus a qualitative look at whether "the sky", "the ridge", "the snow" are single regions or shattered. **If regions do not follow objects, the capability pitch collapses and we should know that before building anything on it.** This is the highest-value measurement available and it has never been attempted.

**P1 — Run the WebP + sidecar benchmark properly, across the ladder.** This is the comparison that matches the product, and the 41% headline above is currently a hypothesis resting on our own illegal wall coder. Steelman the sidecar with a purpose-built region-map codec. If it holds, it is the strongest result in the entire study; if it fails, the positioning above is wrong and we need to know immediately.

**P2 — Adopt the cross-channel transform (B10).** Takes the mid-axis from +8.3% to roughly +4.7% over WebP. Cheap, measured, and it is what turns "structure costs 20%" into "structure costs almost nothing". *In progress.*

**P3 — Fix the wall coder's legality (#12).** The record contains numbers no decoder can produce. Everything downstream inherits the error, including P1.

**P4 — Build a real container.** Without a bitstream there is no format, only a study. Required before any application work.

**P5 — A second image, then a corpus (W7).** Blocks every generalisation claim. Cheapest useful version: Kodak-24 at one small size through the existing frontier.

**P6 — Measure encoder cost (W8).** Determines which applications are even reachable. Trivial to measure and nobody has.

**P7 — Curve-fitted boundaries** for genuine resolution independence. Large, speculative, and only worth it if P0 says the regions are meaningful.

## The shift this implies

The compression verdict does not change: **no rate, no resolution, not lossless where a shape representation is reliably smaller than a well-configured WebP** (README, unchanged). What changes is what that verdict *means*. Being smaller than WebP was never necessary for the product described here. Being *within a few percent* of WebP while carrying a segmentation that WebP cannot carry at any price is a different and much better claim — and after report 09 and report 11 it is nearly true.

The next unit of work should therefore be **P0**, not another byte lever. If the regions are not meaningful, the cheapest possible format is worthless. If they are, we are already close enough on bytes to stop optimising and start building.
