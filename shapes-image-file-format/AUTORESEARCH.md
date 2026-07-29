# Autoresearch ledger — shapes image format, bottleneck programme

**This file is the handover.** An agent with no memory of the conversation should be able to read this file alone, pick the top open item, and continue without asking anything. Update it in the same commit as the work it describes. Newest results go at the top of the log.

## Mission

Attack the shape coder's remaining gap to WebP **one bottleneck at a time**, biggest share of the bill first. Every claimed improvement is measured on the real eval, applied and committed if it holds, reverted if it does not. A clean negative is a successful iteration; do not tune until a number looks good.

The compression *verdict* is settled and is not what this programme is trying to overturn — see `README.md`. What is open is whether the **coder** is anywhere near its own floor. So far it demonstrably was not: report 09 found a 12.6% win and a decodability bug in a component eight reports had assumed was mature.

## Invariants — violate these and the result is worthless

- **Eval**: the macOS Sierra wallpaper. `src4k.png` (3840×2160), plus 1920/960/512 resamples, in `$SCRATCH/hd/`. PSNR is this study's RGB definition — use `labx psnr`, never your own.
- **Reproduce before believing.** Run the unmodified coder and match the published number *before* changing anything. If you cannot reproduce, that is the finding; stop and report it.
- **Price variants side by side on identical partitions.** Add a pricing function; never replace the baseline. A change that alters the partition invalidates the comparison unless the baseline is re-run on the new partition too (falsification #3).
- **Pay for everything.** Side information, learned parameters, mode flags, second passes — all counted in the same total. Most dead "wins" here died when the side channel was priced.
- **Steelman both sides.** Any knob given to WebP (`-m 6`, encode resolution, delivery pipeline) must be offered to the shape coder too. Falsification #11 is what happens otherwise.
- **Causality.** Every context tap must be supplied by a real scan order. Assert it with a decoder-side replay. Falsification #12 is what happens otherwise.
- **Run it twice.** Go map iteration has produced three separate nondeterminism bugs here (#6, #7).
- **The killed list is binding**: sixteen mechanisms in report 04, twelve claims in report 06. Re-deriving a dead idea is this study's most common failure.
- Shell loops under `bash`, not zsh.
- **Build the package before committing agent code.** Independent agents write independent files that collide; `code/lab` shipped broken in `a6d1c73` because nobody compiled it.
- **Check the exit status you think you are checking.** `cmd 2>&1 | head && echo OK` reports `head`'s status, not the compiler's. This has produced two false green results today.

## Where the bill sits at 3840×2160 — this drives priority

| regions | PSNR | total | walls | colour | wall coder actually used |
|---|---|---|---|---|---|
| 227 | 21.99 | 19,819 B | 96.3% | 3.7% | **contour** |
| 1,383 | 24.99 | 50,016 B | 91.6% | 8.4% | **contour** |
| 11,121 | 28.51 | 153,190 B | 79.0% | 21.0% | CAE |
| 96,359 | 32.74 | 533,107 B | 51.8% | 48.2% | CAE |
| 710,144 | 40.42 | 2,413,389 B | 28.7% | 71.3% | CAE |
| 6,356,392 | exact | 11,654,978 B | 7.1% | **92.9%** | CAE |

## Targets

| goal | requirement |
|---|---|
| match WebP at 28.7 dB at 4K | cut 26,438 B = 16.2% of total (**8.3% still to find** after report 09) |
| match WebP at high-rate steps | close −2.39 dB @ 533 KB, −3.20 dB @ 2.41 MB |
| match WebP lossless | cut 33.8% of total ≈ 36% of the colour bill (was 36.5% before the B6 correction) |
| beat AVIF anywhere | 30–50% — treat as out of reach, do not aim here |

## Priorities changed on 2026-07-29 — read report 13 first

The programme was optimising bytes against WebP. Report 13 reframes it: the comparison that matches the product is **WebP + a region-map sidecar**, against which the shape coder appears **41% smaller** (153,190 B vs 258,080 B at 28.5 dB) — hypothesis, not result, because that sidecar figure is our own illegal wall coder.

**The top item is no longer a byte lever.** It is **P0: does the segmentation follow objects or illumination?** Twelve reports measured whether regions are cheap; none measured whether they are *right*, and every capability claim depends on it. If regions shatter the sky into bands and merge a person with the wall, the cheapest possible format is worthless.

| P | item | why it outranks bytes |
|---|---|---|
| ~~P0~~ | ~~Measure segmentation quality~~ | **DONE — report 14. Answered positively.** Heavy-tailed, object-tracing; sky is 2 regions at the capability point. Replaced by P0b below |
| **P0b** | **Benchmark bytes at the CAPABILITY operating point (1,383 regions, 24.99 dB), not just 28.5 dB** | Report 14: capability is best at 227–1,383 regions, byte work optimised 11,121. The benchmark that matches the product does not exist in nine reports |
| P1 | WebP + sidecar benchmark, properly steelmanned | The comparison that matches the product; would be the study's strongest result |
| P2 | Adopt RCT (B10) | +8.3% → ~+4.7% at the mid-axis. *In progress* |
| P3 | Fix wall-coder legality (#12) | The record contains numbers no decoder can produce; P1 inherits the error |
| P4 | Real container | Without a bitstream there is no format |
| P5 | Second image, then a corpus (B7) | Blocks every generalisation claim |
| P6 | Encoder cost | Determines which applications are reachable. Trivial, never measured |
| P7 | Curve-fitted boundaries | Only worth it if P0 says regions are meaningful |

The byte queue below stays valid but is now subordinate to the above.

## Bottleneck queue — work the top open item

| # | bottleneck | why it is worth doing | status |
|---|---|---|---|
| B1 | **Colour coder** — mean predictor, order-0 residual, **no cross-channel transform** | 48–71% of the bill at high rate, 93% at lossless. The most primitive component in the pipeline against WebP's 14 predictors + cross-colour transform | **OPEN — next up**, awaiting fan-out results |
| B2a | Contour coder — **junction map** | — | **CLOSED, NEGATIVE.** It is 2.7–11.2% of the contour bill and already at its floor; widening buys −0.076% of the file at best. See log |
| B2b | Contour coder — **turn stream** | — | **CLOSED, NEGATIVE IN PRACTICE.** −6.40% of turns is real and free, but report 09's interleave collapsed the contour coder's band from ~6,400 regions to ~849, so it now pays only below 24.25 dB. ~2% of learnable slack remains against an oracle |
| B8 | Contour coder — **loop channel** | — | **CLOSED WITHOUT WORK.** Same collapse: its 3.6–7.4% shares sat at 1,383–6,417 regions, which the interleave has since handed to CAE. Worth <0.1% of any file the coder would actually emit |
| B3 | **`bitsPerEdge = 1.73`, `bitsPerReg = 25.0`** (`potts2.go:15`) measured at 512×288, drive the RD merge key and Ising λ at every resolution | Actual cost is 1.22–1.61 bits/edge at 4K rungs and 0.4534 at lossless. The 4K scale-space is therefore not the coder's own RD frontier — the shape coder is undersold at the resolution where the verdict was sharpened | OPEN — must re-run baseline on re-tuned partitions or reproduces #3 |
| B4 | **Re-price report 08 against a legal wall coder** | #12: published CAE numbers are optimistic by 3.4–12.7%. Report 08's tables are flagged but not corrected | OPEN — bookkeeping, no research risk |
| B5 | **Rung 2 of the rate ladder** | No mark within ±5% of 50,016 B; needs a merge run at ~7,800 regions to settle whether WebP's lead there is real | OPEN — one run |
| B6 | Inconsistent pricing at the lossless row | — | **CLOSED, APPLIED.** Real, verified twice, and it moved a published headline: lossless total 12,159,385 → **11,654,978 B**, 1.58× → **1.51×** WebP. Shapes now sit *under* AVIF and 5% under PNG |
| B6b | The 512/960/1920 lossless rows | — | **CLOSED, APPLIED.** All four re-priced with `colorBytes2`: 1.400× / 1.352× / 1.352× / 1.510× against WebP-lossless, where the record said 1.48 / 1.41 / 1.40 / 1.58. Curve shape unchanged |
| B10 | **Adopt the cross-channel transform (plus report 12's 8-byte chroma coefficient) in `colorBytes2` and re-price the record** | **−28.0%** of the colour bill, and every colour figure in reports 04–09 was measured without it. Re-pricing must be checked against a general compressor, not only modelled — report 12 showed the two rank differently | **OPEN — top of the colour queue, and the last colour item** |
| B9 | Residual-context colour model | Was queued as the largest colour win at −10.33%. On top of RCT it is worth **+4.7 pp**, not 10.3 | OPEN — do after B10, and quote the stacked figure |
| B7 | Generalisation: every result is one photograph | The frozen 16-tap template and any colour win may not transfer. Cheapest real test: Kodak-24 at one small size through the existing frontier | OPEN — blocks any "ship it" claim |

## Log — newest first

### 2026-07-29 — P0 answered: the regions are meaningful (report 14)

Recovered the partitions from the published renders and counted regions inside three named windows.

| regions | PSNR | **sky** | ridge | snow | top-100 cover | median region |
|---|---|---|---|---|---|---|
| 227 | 21.99 | **2** | 21 | 19 | **99.4%** | 1,716 px |
| 1,383 | 24.99 | **2** | 84 | 76 | **90.0%** | 223 px |
| 11,121 | 28.51 | 11 | 633 | 452 | 65.2% | 34 px |

**Heavy-tailed at every level** — largest region is 3,450× the median at 1,383 regions, where uniform banding would give ~1×. A hundred regions describe 99.4% of a 4K photograph at 227 regions and 90% at 1,383. Visually: the ridge window splits into shadowed rock / lit snow / sunlit peak along the ridgeline and shadow terminator; the sky splits along a contour tracing the cloud edge. At 11,121 the sky bands into 11, and they follow cloud *form*, not horizontal slices.

**The pessimistic reading in report 13 W1 was wrong.** Report 04's observation was incomplete rather than wrong: the sky is split by brightness, but into 2 regions at the capability point — a usable failure, since "select the sky" is a union of two ids.

**The finding worth more than the answer: the capability point and the byte point are different operating points.** Capability is best at 227–1,383 regions (21.99–24.99 dB). Byte competitiveness was optimised at 11,121 (28.51 dB), where the median region is 34 px of texture speckle. Nine reports optimised a rate the applications do not want. **Nobody has ever benchmarked this format against WebP at 1,383 regions**, because 24.99 dB was assumed too low to ship — an assumption that only holds if fidelity is the product.

Limits: three windows, one photograph, contents named by eye rather than against annotated ground truth. No boundary-recall number against a human or SAM segmentation. Strong indicator, not proof.

### 2026-07-29 — CORRECTION: cancelling lenses 3–6 was not supported by the measurement

I closed the colour lens programme after report 12 on the grounds that only ~1% of headroom remained and that colour had become bookkeeping. Challenged on it, and the challenge is right on both counts.

**The floor is not proven.** Report 11 states its own limit explicitly: *brotli is an upper bound on achievable, never a lower bound on entropy — the true floor is below 6,904,345.* The ~1.2% headroom I quoted came from a static oracle **inside the same family**: predict, then code a residual. A lens proposing a different *representation* — spectra's low-dimensional reflectance manifold is the live example — is not bounded by that oracle at all. I inferred a ceiling from a measurement that does not establish one.

**And colour is not bookkeeping at the mid-axis.** Recomputing: after report 09's interleave, 28.7 dB stands at +8.3% over WebP. Colour is ~32,143 B of that 153,190 B rung, and RCT+brotli takes −15.4% of it = −4,942 B, moving **+8.3% → +4.7%**. Colour is worth roughly **a third of the entire remaining mid-axis gap**. The correct claim is "walls must close most of it", not "colour cannot help".

Evidence against my own reasoning, which I should have weighted: the light lens was the one I expected least from and it produced both a real lever and the modelled-vs-real ranking finding — which the floor ladder did not predict and which is worth more than the lever.

**Lenses 3–6 are restored to the queue**, ordered by whether they propose a different representation (not bounded by the residual floor) or a better model (bounded by it):

1. **spectra** — reflectance is 3–7 dimensional; a per-image colour manifold is a different representation, so the residual-family floor does not bound it. Highest value of the four.
2. **matter** — coarse-to-fine cascade over the merge hierarchy that already exists; targets the lossy end, where colour is 21–71% of the bill.
3. **vision** — perceptual; lossy end only, and report 04 already killed foveation/CSF as codec-agnostic preprocessing.
4. **biology** — weakest prior; report 04 killed PDE inpainting, L-systems and stigmergy.

**Sequencing, and the reason it is not cancellation:** each must be measured against the **post-RCT** baseline (6,898,336 B lossless), not the pre-RCT 10,832,609 — otherwise every one of them will claim a win that RCT already banked. So they run after B10 lands, one at a time. Running spectra now would produce a number nobody could use.

### 2026-07-29 — light lens closed negative; and modelled bytes do not rank like real bytes (report 12)

**The structural claim is dead.** Dividing region colours by a smooth illumination field *raises* their joint entropy 5.9% (13,135,022 → 13,914,233 B) and grows the alphabet 50%, and **every shuffled control beats the real field** — destroying the spatial correspondence helps, which it could not if illumination were being removed. After brotli every transmitted-field arm is a net loss: poly2 +0.016%, grid64 +0.638%, textbook Retinex +13.76%. Not a rename of report 04's killed coarse-field mechanism, and shown rather than asserted: it fails in the 21-parameter form and fails *monotonically worse* as the field gets finer.

**What survived is 8 bytes.** A two-coefficient chroma refinement to RCT: 6,904,345 → **6,898,336 B (−0.087%)**, decode-verified. Re-verified outside the agent's harness byte-exact, with both controls costing *more* than doing nothing (`a` negated +0.342%, `a` on the wrong channel's ratio +0.537%) — so it is information, not capacity. Entire win is the blue channel (`a_B` = +0.136, `a_R` = −0.014), which on a sky photograph is plausible and fragile. Belongs with B10, not as its own mechanism.

**The finding that outlives the lens:** `k`-only improves the order-0 model by 0.04% and makes brotli **0.60% worse**. The lossless residual stream is 37% zeros; brotli lives on the exact-hit rate, so a refinement that lowers residual *variance* while lowering *exact hits* trades away more LZ matches than it wins. **Any colour lever ranked on modelled bytes can invert once a real compressor is the reference — report 11's own ladder ranks these two backwards.** Every bespoke colour figure in reports 04–09 is a modelled number and none has been checked against a general compressor on the same stream. Report 11 now carries this caveat.

**Lenses 3–6 were cancelled here, and that was reversed within the hour as unsupported. See the correction entry above this one.**

### 2026-07-29 — B6b closed: all four lossless rows re-priced

Re-ran the lossless stage at every resolution with the corrected `hd.go` (`colorBytes2`, not `colorBytesLean`), and measured `cwebp -lossless -z 9` at each size directly rather than reusing a remembered figure.

| size | shapes (corrected) | WebP-lossless | ratio | published | delta |
|---|---|---|---|---|---|
| 512×288 | 243,481 B | 173,906 B | **1.400×** | 1.48× | −0.080 |
| 960×540 | 815,037 B | 603,038 B | **1.352×** | 1.41× | −0.058 |
| 1920×1080 | 3,143,329 B | 2,325,106 B | **1.352×** | 1.40× | −0.048 |
| 3840×2160 | 11,654,978 B | 7,718,506 B | **1.510×** | 1.58× | −0.070 |

Every row was 0.05–0.08 too high. The qualitative claim survives intact: the ratio is roughly flat across sizes and clearly worst at native resolution, which is what report 08's argument rests on. Report 08's line now carries the corrected figures.

Note the internal consistency check: 1.48 × 173,906 = 257,381 B, which is 5.4% above the corrected 243,481 — the same order as the colour-coder correction that caused it. The old numbers were wrong in exactly the way the fix predicts.

### 2026-07-29 — the floor is not the problem; a missing transform is (report 11)

**Verdict: −36% of the colour bill is reachable and has already been reached.** A decode-verified stream of **6,904,345 B (−36.26%)** exists, produced by handing the residuals to off-the-shelf `brotli -q11`. A WebP tie on colour alone needs 6,896,137 B — the gap is **8,208 B, 0.076% of the colour bill**.

**Independently verified before recording**: re-ran `brotli -q 11` on the agent's dumped residual stream outside its harness and got **6,904,345 B** to the byte; their decoder replay rebuilds all 24,883,200 samples with max |Δ| = 0.

**The finding that reorders this queue:** the coder sits **0.02% above the static ideal of its own model** and **36% above the floor of its own data**. It is a competent implementation of the wrong model. The entire deficit is one missing **cross-channel transform (G, R−G, B−G)** — worth **−28.0% alone**, roughly three times B9, which was queued as the largest colour lever. And they do not add: residual context on top of RCT is worth **+4.7 pp**, not 10.3. Every colour lever measured so far was competing for slack a single missing transform already accounted for.

**The regimes have swapped owners.** At the lossy point (11,121 regions) the raster predictor goes *positive* (+0.9%) and the context ladder starves; RCT still gives −15.4% of colour, but colour is 21% of the bill there, so that is **−3.2% of total against the 8.3% still needed at 28.7 dB**. Colour owns the lossless end and can nearly close it. Walls own the middle and colour cannot help there.

**Not claimed, deliberately:** not a WebP tie — 822,369 + 6,904,345 = 7,726,714 vs 7,718,506 is 1.001×, but that wall figure is the coster #12 showed is not decodable and its legality cost at lossless was never tabulated. brotli is an upper bound on achievable, not a lower bound on entropy; no CM/PAQ-class coder was installed, so the true floor is *below* 6,904,345. Oracle rows are not floors — a junk control of 7,920 meaningless contexts reaches 9,051 B at the lossy point, so no wide-context static number there is quotable.

### 2026-07-29 — spend limit raised; the six colour lenses run SEQUENTIALLY, floor first

The multi-disciplinary colour lenses that died on the spend limit are being re-run **one at a time**, not fanned out, so a single agent's cost is bounded and each result can redirect the next.

**Order is deliberate and is not the order they were originally written in.** The information-theory lens runs *first and alone*, because it measures `H(region colours | partition)` — the floor every other colour lens is competing for. If that floor sits above the −36% of the colour bill the lossless target needs, the colour programme is finished on arithmetic and the remaining five lenses are cancelled unrun. Floors before levers.

Queue, in order, each gated on the previous:
1. **information theory** — the entropy floor. *(running)*
2. light / illuminant × albedo separation — the one with a real structural claim: natural images are a smooth illumination field times a piecewise-constant albedo field, which is the structure a region coder should want.
3. spectra — reflectance is low-dimensional (3–7 basis functions); measure this image's actual region-colour dimensionality.
4. vision — chroma acuity and opponency: how few bits a region colour perceptually needs. Note report 04 already killed foveation/CSF as codec-agnostic preprocessing.
5. matter — coarse-to-fine cascade over the merge hierarchy, which already exists; conditional entropy of a child colour given its parent.
6. biology — structural colour, Turing patterning. Ranked last: report 04 already killed PDE inpainting, L-systems and stigmergy, so the prior on this lens is worst.

An outside review had recommended cutting 5 and 6 entirely. They are kept but ranked last, so if the floor result cancels the phase they simply never run.

### 2026-07-29 — B2b/B8 closed: report 09 ate the contour coder's territory

The turn stream is 77–94% of the contour bill, so it looked like the largest unexamined component in the pipeline. Three free context taps — the junction map of the three candidate targets, hard exclusions from decoder-known structure, and absolute direction of travel — take **−6.40% of turns at 11,121 regions**. A junk control of 864 equal-sized contexts *costs* +0.49%, so it is information, not capacity.

**It is worth almost nothing, because of our own previous result.** Report 09's interleave cut CAE enough to move the CAE/contour crossover from ~6,400 regions down to between 849 and 1,383 — verified directly against report 09's table:

| regions | base CAE | interAsym | contour | chosen before → after |
|---|---|---|---|---|
| 849 | 47,181 | 37,728 | 37,518 | contour → contour |
| 1,383 | 55,565 | **44,726** | 45,797 | contour → **CAE** |
| 6,417 | 96,087 | **81,522** | 93,577 | contour → **CAE** |

So contour is now chosen only below ~849 regions at 4K — **21.99 to 24.25 dB, below any usable fidelity**. The turn lever is worth 1.0–1.2% of the file in a band nobody operates in, and exactly 0.00% everywhere else. Had it been measured before report 09 it would have been worth −2.76% at 6,417 regions.

**This is the programme's first case of one result devaluing another.** Improving CAE did not just help CAE — it shrank the region of the ladder where the contour coder is used at all, and retroactively made an entire queue item worthless. Bottleneck priority computed from a static bill table is wrong the moment any component improves; B2 was ranked #2 on numbers that report 09 had already invalidated.

Other findings, recorded because they cost nothing to keep:
- **My brief's premise was false.** I told the agent the turn stream should be dominated by long straight runs after the Ising relaxation. It is not: p(straight) = 0.536 and the mean straight run is **1.157** — relaxed walls are 45° staircases at pixel level, not axis-aligned. Explicit run-length contexts are **+23% worse**. Do not brief a hypothesis as a fact.
- **Causality clean.** A decoder state machine rebuilds the crack graph from (junction bitmap, direction bits, turn symbols) alone and asserts equality — `reconstruct=true` at all 10 operating points. No #12 here. One unpaid hole: the loop count is never transmitted.
- **Near its floor anyway.** Static conditional entropy at 227 regions: published order-3 = 17,956 B, a 132,192-context oracle = 17,232 B. **~4% of slack against an oracle, ~2% against anything learnable online.**
- **The small eval mis-ranks context depth, again.** `ord5` is −1.07% at 4K/227 but +0.12% at 960/239. Third instance of falsification #2's shape.

### 2026-07-29 — the fan-out died on the account's monthly spend limit

7 of 14 workflow agents failed with `You've hit your monthly spend limit`: **all six multi-disciplinary colour lenses** (vision, light, spectra, biology, matter, information) **and the synthesis**. The multi-disciplinary colour research was never run. What survived is the three wall levers and the four compression-native colour levers, already logged.

**Consequence for whoever picks this up: spawning subagents will fail until the limit is raised.** The remaining queue items are all doable solo, and B9 is the one with a measured number behind it.

### 2026-07-29 — B6 applied: the lossless row was priced with a weaker coder than its own table

`hd.go:140` priced the lossless row with `colorBytesLean` (single longest neighbour) while every lossy rung uses `colorBytes2` (boundary-weighted mean of all decoded neighbours) — which is also the method report 08 *states* for the whole table. The study was comparing itself against a weaker version of itself on exactly one row.

**Verified independently before applying**: `hdcheck` confirms `colorBytes` ≡ `colorBytesLean` = 11,337,015.3004 B (the published figure), and a patched build prices the same partition with `colorBytes2` at **10,832,608.78 B**, matching two agents that found it separately. Block regenerated by the tool, not edited by hand.

| | published | corrected | change |
|---|---|---|---|
| lossless colour | 11,337,015 B | **10,832,609 B** | −4.45% |
| lossless total | 12,159,385 B | **11,654,978 B** | −4.15% |
| vs WebP | 1.58× | **1.51×** | — |
| vs AVIF (11,969,137) | 1.016× — above | **0.974× — below** | ranking changed |
| vs PNG (12,278,280) | 0.990× — dead heat | **0.949×** | — |

This one flatters the hypothesis, which is why it was verified twice before being applied. Still pending: the rate ladder's step 7 and the mirror page carry the old number and need regenerating.

### 2026-07-29 — fan-out results: four levers measured, none close the gap

| lever | component | best delta on its own bill | delta on total | where |
|---|---|---|---|---|
| **cross-plane interleave** | walls | **−12.64%** | −9.98% | 28.51 dB |
| context mixing (6 templates) | walls | −5.23% | −4.13% | 28.51 dB |
| wider context (16-bit) | walls | −6.22% | — | 28.51 dB |
| **residual context** | colour | **−10.33%** | −9.60% | lossless |
| RD choice of 5 predictors | colour | −5.24% | −4.87% | lossless only; **zero by 28.5 dB** |
| coding order (6 traversals) | colour | −2.06% | −1.47% | 40.42 dB; ~0 at both ends |

- **Walls: interleave still wins.** Mixing is worth −5.23% and decomposes as −1.3 pp estimator swap, −3.9 pp mixing proper; a single wider template alone gets two thirds of it. Mixing independently re-confirmed #12 with a runnable causality check.
- **Both wall levers are worth exactly 0.00% below ~6,400 regions**, where the contour coder is already cheaper. The low-rate arm can only be attacked through B2b.
- **Colour: the predictor is not the lever, the residual model is.** The boundary-weighted mean is already the best single predictor everywhere except near-lossless; five-way RD selection buys −5.24% at lossless and *nothing* by 28.5 dB. Conditioning the residual on prediction spread × previous residual buys twice that.
- **Ordering is free and nearly worthless** — three orders of magnitude short of the targets at both ends.
- Every colour lever's payoff tracks **pixels per region**, not resolution, and at 512/960 the same contexts go *positive* above ~4 px/region. A small-image study would have concluded they were worthless. They were starved — falsification #2's shape again, and the reason 4K was mandatory.

### 2026-07-29 — B2a: contour junction map is at its floor (clean negative)

**The measurement that matters is the split, which had never been taken.** `vertBits` was three channels, not one: the junction bitmap, four presence bits per special vertex, and a flat per-loop cost. At 3840×2160:

| regions | contourB | juncB | junc% | turnB | **turn%** | loopB | loop% |
|---|---|---|---|---|---|---|---|
| 227 | 19,079 | 510 | 2.67% | 17,989 | **94.3%** | 266 | 1.4% |
| 1,383 | 45,797 | 2,611 | 5.70% | 40,199 | **87.8%** | 1,658 | 3.6% |
| 6,417 | 93,577 | 10,452 | 11.17% | 71,819 | **76.8%** | 6,927 | 7.4% |

Verified independently: the four channels reconcile to `contourB` on every row, and the totals match the published ones.

- **Junction map: negative, and structurally so.** Greedy widening saturates at 16 bits for **−0.82%** of the map = **−0.076% of the file**. It is already at its floor: at 227 regions the adaptive 10-bit coder costs 510 B against a memoryless 503 B and an enumerative `log2 C(N,k)` floor of 502 B — **the context model costs 1.4% *more* than having no context at all**, because 1,024 models are learning to describe 244 junctions.
- **It cannot repeat report 09's win, for a reason worth keeping.** Junction count is set by region count; turn count scales with the linear dimension. So the junction map's share *shrinks* with resolution — 13.95% of the contour bill at 512×288 down to 2.67% at 4K. Sparsity binds, not sample count: 7,687 positive samples over 65,536 models. 4K is where this component matters **least**, exactly inverting report 09's finding for CAE.
- **Causality: clean.** Decoder replay over all 8.29M positions reports **0 mismatching contexts** at four resolutions, against CAE's 21,554 at 512×288. `contour.go` does not have `potts.go:311`'s disease.
- **New decodability gap (small).** The loop channel sends no count and no end-of-loops flag, so a decoder cannot tell when the loop list stops. Repair costs 10.8 B at 227 regions to 277 B at 6,417 (0.06–0.30%). Recorded, not fixed.
- **The contour coder's band is wider than published.** Against a *legal* CAE (#12: +4.6% at 11,121) the 11,121 rung reads CAE 126,615 B vs contour 126,291 B — contour is chosen up to ~11,000 regions, not ~6,400. More of the ladder depends on this coder than report 09 assumed.

Data: `10-contour-junction-map-data.txt`. Code: `code/lab/contourctx.go`, `code/lab/selectedj.go`.

**Also fixed here: `code/lab` did not compile.** Report 09 committed `crossplane.go` and `wallctx.go` from two independent agents into one package; both declare `tap` and `crackPlanes`. The repo's promise that every number can be re-derived was broken from `a6d1c73` until now. Pure rename, no behaviour change. **Build the package before committing agent code** — added to the invariants.

### 2026-07-29 — report 09: wall coder, two findings (commit `a6d1c73`, pushed)

- **Cross-plane interleave wins: −12.6%** of the wall bill at 11,121 regions (121,047 → 105,752 B). `interVH` gets −11.1% at the *same* 10/10 context width and same 2,048 models; `base12` with 4× the models buys 1.1%. **Schedule, not capacity.** Coding Hz first instead is +10.4% worse. Win band ~25–31.5 dB; reverses at fine partitions (+12.6% at 3.4M regions, +1.7% lossless). At 28.7 dB the WebP deficit goes **+19.3% → +8.3%**.
- **Context width is real but half as good: −6.2%**, and mostly outside the band where CAE is chosen. Settles a review question: the same frozen 16-bit template is **+2–4% worse at 512×288 and −10.3% at 4K**, so the 10-bit context was right for the old eval and under-conditioned at native resolution. Static `H(X|ctx)` falls to 20 bits while adaptive cost turns at 16 → ceiling is model-learning cost.
- **#12: the published CAE coder is not decodable.** `potts.go:311` reads `Hz(x+1,y)` (uncoded) alongside `Hz(x-1,y)`/`Hz(x-2,y)`. Replay: 21,554 bad contexts at 512×288, 51,995 at 960×540. Legality costs +3.4% to +12.7% of the wall bill.
- **#11: the rate ladder was one-sided.** Only WebP got a resolution search below its floor. Given the same knob the shape coder ties rung 1 — **24.59 dB at 20,618 B** vs WebP 24.54 dB at 20,066 B. The published −2.55 dB is retracted.
- Not done: mixing arm still running; report 08 tables not re-priced; template tested on one image only.

## Parked work

Anything set aside lives in [`PARKED.md`](PARKED.md) with its revive trigger. **Re-read it whenever a baseline here moves** — that is when a parked entry silently becomes valuable. Two entries are already live candidates:

- **P-01, contour turn coding** — a verified, free −6.40% that is worth ~0 only because report 09 moved the CAE/contour crossover. Fixing #12 widens the contour band again and may already have moved it back. **Not recomputed.**
- **P-07, curve-fitted boundaries** — parked conditional on P0, and **P0 has since been answered positively** (report 14). Its stated condition is satisfied and it has not been re-evaluated.

## How to resume

1. Read this file, then `README.md`, then `06-corrections-and-falsifications.md` (what is already dead).
2. Pick the top **OPEN** item in the bottleneck queue.
3. Reproduce the relevant baseline number first. Then implement the smallest honest variant beside it.
4. Measure at a low-rate point, a high-rate point, and lossless — a lever can matter at one end and be irrelevant at the other.
5. If it holds: apply, commit with the numbers in the message, push, and add a log entry here. If it does not: add the log entry anyway with the number that killed it, and mark the bottleneck closed-negative. **Both outcomes are results.**

Paths: repo `research/shapes-image-file-format`; Go lab `code/lab` (copy it out, `go build -o labx .`); working assets under the session scratchpad `hd/` (`src4k.png`, `renders4k/`, `ladder_*.txt`).
