# 13 — What this format is for, and what it must fix to deserve it

**Every report before this one asks "is it smaller than WebP?"** That question assumed the products are comparable. They are not: WebP delivers pixels, this delivers pixels *and* a segmentation. This report states the capabilities honestly, prices them where the record allows, names the weaknesses that block them, and ranks the work by how much value it unlocks rather than how many bytes it saves.

Nothing here is a new measurement. Every number is drawn from reports 01–12, and anything unmeasured is labelled **UNMEASURED** rather than argued.

> **Refreshed after reports 14, 15 and 16.** Three claims below were superseded within hours of writing: W1 ("nobody has checked whether the regions are meaningful") was answered positively by report 14; W3's byte gap went from the +8.3% quoted here to **+0.91%** at matched fidelity (report 16); and P0 and P2 are both done. The updates are inline and marked. A new weakness — W9, the capability point and the byte point being different operating points — comes from report 14 and did not exist when this was written.

## The baseline was wrong

The comparison that matches this product is not WebP. It is **WebP plus a region-map sidecar**, because that is what a consumer would have to assemble to get the same capabilities.

| at 28.5 dB, 3840×2160 | bytes | has structure? |
|---|---|---|
| WebP alone | 137,033 B | no |
| **WebP + a region map** (137,033 + the 121,047 B wall bill) | **258,080 B** | yes |
| **shape coder** | **153,190 B** | yes — the structure *is* the image |

**Against the baseline that matches the product, the shape coder is 41% smaller.** Raster-plus-sidecar pays for boundaries *and* every pixel; the shape coder pays for boundaries and derives the pixels from them. The geometry stops being overhead and becomes load-bearing.

> **Now measured — report 19.** Re-run at matched fidelity with a legal wall coder and the sidecar steelmanned across five encodings: **44.1% smaller at 28.5 dB and 40.1% at the capability point**. Our boundary coder *is* the strongest sidecar, beating the best off-the-shelf option (raw label plane + `xz -9e`) by **2.46×**, so pricing the sidecar with it is conservative rather than self-serving. Against what a consumer could actually download, the margin is 60–66%. The 41% here was a good guess.

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

Marked **[needs P0]** where the claim depends on the regions being *semantically* meaningful. **Report 14 has since answered P0 positively** — heavy-tailed partitions, sky is 2 regions, a hundred regions cover 90–99% of the image — but only at **227–1,383 regions**. At 11,121 the median region is 34 px of speckle, so these marks now mean "holds at the capability rate, degrades above it".

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

**W1 — ~~Nobody has ever checked whether the regions are semantically meaningful~~ — ANSWERED, report 14.** The pessimism here was wrong. The partitions are heavy-tailed (largest region 3,450× the median at 1,383 regions, where uniform banding would give ~1×), the sky is **2 regions**, and a hundred regions describe **99.4%** of a 4K photograph at 227 regions and 90% at 1,383. Regions follow *tonal* boundaries, which on this image means they follow objects wherever tone and object coincide, and trace cloud form in the sky. Report 04's observation was incomplete rather than wrong: the sky *is* split by brightness, but into 2 regions — a usable failure, since selecting it is a union of two ids. **Residual gap:** no boundary-recall number against annotated ground truth or SAM; three windows, one photograph.

**W2 — The capability band is narrow, and it is not where fidelity is good.** The shape story holds from ~227 to ~11,000 regions = **21.99 to 28.5 dB**. At 4K lossless the partition degenerates to **1.305 px/region** — 6.36M "regions", no shapes at all, just a raster coder with extra steps. The honest pitch is *structure in exchange for mid fidelity*, never *structure for free*.

**W3 — ~~Still behind on bytes~~ — closed at the mid-axis *on one image*, reports 16/21; reopened by report 22.** On three Kodak photographs the same pipeline is **+9.4% to +71.5%** over WebP. Parity holds where a hundred regions explain most of the picture (90% on Sierra) and fails where they explain ~40% (Kodak). The claim below is true and content-dependent. At **matched fidelity** the shape coder is **132,280 B against WebP's 131,082 B — +0.91%**, with every component decodable. The +19.3% this document was written around described a coder with an illegal wall coster and no cross-channel transform. **What remains:** the wall half is still an idealised cross-entropy while WebP's number is a real file, so roughly that last 1% is container overhead not yet paid — parity is plausible and **unproven** until W6 is fixed. Higher up the axis the gap is larger and has not been re-measured. **AVIF is still 30–50% ahead everywhere** and is not a target.

**W4 — Not resolution-independent.** Boundaries are crack edges on a pixel lattice. Report 06 #4 already killed "renders at 8K for the same bytes" as a baseline error. It upscales *gracefully* (strength 7), which is not the same thing. True resolution independence needs curve-fitted boundaries, and nobody has priced that.

**W5 — ~~The wall coder is not decodable~~ — FIXED, report 20.** `potts.go` reads `V(x+1,y)` where it read the uncoded `Hz(x+1,y)`, and report 08 is regenerated from the legal coder. Measured cost: **+4.33% to +13.71%** where CAE is chosen, 0.00% below ~6,400 regions. The parity and sidecar headlines never depended on it — they use the interleaved coder, independently confirmed decodable by `lab wallcheck`.

**W6 — ~~There is no container and no bitstream~~ — BUILT, report 21.** SHPC v1 emits a real file that round-trips bit-exactly, at **+21 B / +19 B** over the estimates. What is still missing is everything *beyond* the minimum: no error resilience, no truncation (so report 13's progressive-stream strength is claimed but unimplemented), no metadata, no alpha, no colour-space tag.

**W7 — One photograph.** Every number in the record comes from the same macOS Sierra wallpaper. No corpus, no BD-rate, no content diversity. Nothing here can be claimed to generalise.

**W9 — The capability operating point and the byte operating point are different. (New, report 14.)** The segmentation is best at **227–1,383 regions (21.99–24.99 dB)**; every byte result, including the 0.91%, was measured at **11,121 regions (28.51 dB)** where the median region is 34 px of texture speckle. Nine reports optimised a rate the applications do not want, and the rate they *do* want has never been benchmarked against WebP. This is now the largest open question in the programme.

**W8 — ~~Encoder cost is unmeasured~~ — MEASURED, report 18: 3 m 44 s and 2.89 GB at 4K, single-threaded.** 68× `cwebp` at 960×540, ~150× at 4K. It does **not** block the stage-4 applications, which all operate on an already-encoded partition, but it excludes upload-time encoding, interactive re-encode, and mobile/embedded entirely. Decode is unaffected and remains a strength. The number is soft — the encoder is single-threaded on 15 cores, prices all 20 marks when one is wanted, and has never been profiled — so it is a lever (**new P8**), not a wall.

## Roadmap

Four stages. **Nothing in stage 4 should start before stage 1 finishes** — every application rests on numbers that are one measurement away from existing.

### Stage 1 — prove the product is real (days, cheap)

| # | item | why now | cost |
|---|---|---|---|
| ~~P0b~~ | ~~Bytes at the capability rate~~ | **DONE — report 17. A dead heat: 25,399 B at 24.97 dB against WebP's 25,700 B at 24.99 dB, both resampled, −1.2%.** The format is at parity where its structure is useful | — |
| ~~P1~~ | ~~WebP + region-map sidecar~~ | **DONE — report 19. Holds: 44.1% smaller at 28.5 dB, 40.1% at the capability point, against the strongest sidecar — which turned out to be our own coder, beating the best off-the-shelf option by 2.46×.** 60–66% against what a consumer could assemble | — |
| ~~P6~~ | ~~Encoder cost~~ | **DONE — report 18. 3 m 44 s / 2.89 GB at 4K.** Excludes on-demand encoding; does not block the ranked applications | — |

### Stage 2 — make it an actual format (weeks)

| # | item | why |
|---|---|---|
| ~~P3~~ | ~~Fix the wall coder's legality (#12)~~ | **DONE — report 20. `potts.go` now reads `V(x+1,y)`; report 08 regenerated from the legal coder.** Cost +4.33% to +13.71% where CAE is chosen, 0.00% below ~6,400 regions. **The parity and sidecar headlines were already legal** — they use the interleaved coder, confirmed decodable |
| ~~P4~~ | ~~Build a real container and bitstream~~ | **DONE — report 21. SHPC v1, ~20 B of overhead, round-trips bit-exactly.** Parity is now measured, not plausible: **+0.930%** at 28.5 dB and **−1.097%** at the capability point, as real files |
| **P8** | **Profile and parallelise the encoder** | 3 m 44 s single-threaded on 15 cores, pricing 20 marks when one is needed, never profiled. Engineering, not research — and "3.7 minutes" kills adoption arguments before the byte numbers are heard |
| — | Fix the loop-count hole (P-02) | A decoder cannot tell when the loop list ends. Correctness, not optimisation |

### Stage 3 — prove it generalises (weeks)

| # | item | why |
|---|---|---|
| ~~P5~~ | ~~A second image~~ | **DONE — report 22, and it broke the headline.** Three Kodak images: **+9.4% to +71.5%** over WebP against +0.93% on Sierra. Parity is content-dependent, not a property of the format |
| **P5b** | **Full Kodak-24, and find the regime boundary** | Report 22 gives a *predictor* — top-100 region coverage, 90% on Sierra vs ~40% on Kodak. Establish where the crossover is, so the format's regime can be stated rather than guessed | now the top item |
| **P5c** | **Re-run the sidecar comparison (report 19) on Kodak** | Its 40–44% is a ratio and could hold, narrow or invert on busy content. **Must not be quoted as general until measured** |
| — | Boundary recall vs SAM / annotated ground truth | Report 14's residual gap: regions were judged meaningful by eye on three windows |
| — | Perceptual metrics (SSIMULACRA2, butteraugli) | PSNR only, and report 04 already found the two disagree on flat interiors |

### Stage 4 — build on it (only after stage 1)

Ordered by how much they exercise strengths that no raster format has:

1. **Selection primitives** — select-by-colour-proximity, region merge, contiguous-area grow. Pure graph queries over `share[]`, which already exists. This is the "remove background" / "select the sky" demo, and at 1,383 regions the sky is *2 ids*.
2. **Non-destructive editing** — recolour, adjust, replace per region with no generational loss. Strength 3, needs no new research.
3. **Cutout animation** — per-region transforms with stable ids. What `pixelize` and `sprites` already want.
4. **Semantic LOD for games** — the nested hierarchy already exists in `hdMarks`; needs the container, not research.
5. **Bounded-error archival** — needs the container plus a stated error guarantee.

### Deferred, with triggers in [`PARKED.md`](PARKED.md)

P7 curve-fitted boundaries (trigger fired, deferred on cost); the four remaining colour lenses (blocked until each can be measured against the post-RCT baseline); P-01 contour turns and P-02 loop channel (crossover moved against them); P-08 affine colour (expected value dropped after report 15).

## The shift this implies

The compression verdict does not change: **no rate, no resolution, not lossless where a shape representation is reliably smaller than a well-configured WebP** (README, unchanged). What changes is what that verdict *means*. Being smaller than WebP was never necessary for the product described here. Being *within a few percent* of WebP while carrying a segmentation that WebP cannot carry at any price is a different and much better claim — and after report 09 and report 11 it is nearly true.

The next unit of work should therefore be **P0**, not another byte lever. If the regions are not meaningful, the cheapest possible format is worthless. If they are, we are already close enough on bytes to stop optimising and start building.
