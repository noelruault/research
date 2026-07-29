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
| B6b | The 512/960/1920 lossless rows are still `colorBytesLean`-priced | Same bug, other three resolutions; each is ~4% too high, and report 08's "ratio is stable across sizes" line rests on them | OPEN — one run each |
| B9 | **Apply the residual-context colour model** | Measured −10.33% of the colour bill at lossless (−9.60% of total), the largest single win found so far. Needs integrating and re-pricing | OPEN — pending the cross-channel agent, which may overlap with it |
| B7 | Generalisation: every result is one photograph | The frozen 16-tap template and any colour win may not transfer. Cheapest real test: Kodak-24 at one small size through the existing frontier | OPEN — blocks any "ship it" claim |

## Log — newest first

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

## How to resume

1. Read this file, then `README.md`, then `06-corrections-and-falsifications.md` (what is already dead).
2. Pick the top **OPEN** item in the bottleneck queue.
3. Reproduce the relevant baseline number first. Then implement the smallest honest variant beside it.
4. Measure at a low-rate point, a high-rate point, and lossless — a lever can matter at one end and be irrelevant at the other.
5. If it holds: apply, commit with the numbers in the message, push, and add a log entry here. If it does not: add the log entry anyway with the number that killed it, and mark the bottleneck closed-negative. **Both outcomes are results.**

Paths: repo `research/shapes-image-file-format`; Go lab `code/lab` (copy it out, `go build -o labx .`); working assets under the session scratchpad `hd/` (`src4k.png`, `renders4k/`, `ladder_*.txt`).
