# Pre-registration — the three measurements that decide the niche

**Written before any of these runs. Committed first, on purpose.**

This study's method rule #1 is *name which knobs each side of a comparison may vary, before the run*. That error was made twice, the second time in the session that documented it. This file extends the rule from knobs to **metrics and decision thresholds**, because the next three measurements are the ones that choose what this format is for, and a threshold chosen after seeing the data is not a threshold.

Reports 29, 30 and 31 will cite this file. If a result lands outside what is written here, the rule is: **publish the number and the fact that it fell outside the registered rule** — never quietly re-draw the line.

## Why now

The byte question is closed (report 23: zero wins in 24 against WebP). The surviving claim is report 24: against WebP **plus a region-map sidecar**, 24 wins of 24, mean +30.5%. Report 28 attached a number to *why* that baseline is fair — a client-side mask keeps only 24–40% of its boundaries across two deliveries — but drew its own boundary explicitly: the claim **does not apply** to one-shot local selection where any plausible segmentation is acceptable.

That carve-out is where the two candidate niches split:

- **Authored assets (gaming, animation)** need region identity stable across clients, re-encodes and time. Report 28 supports this directly.
- **Photo editing** is largely one-shot local selection, and its free alternative is not our merge — it is a modern automatic segmenter, which this study has **never measured against**.

M1 and M2 harden the claim we already have. **Only M3 decides the niche.** That asymmetry is the reason this file exists.

## Standing invariants

All three inherit `AUTORESEARCH.md`'s invariants — reproduce before believing, pay for everything, steelman both sides, run it twice, the killed list is binding. Plus:

- **A perfect result is a broken test.** Report 28's first run scored 100.0000% because it compared a file with itself. Any score at a boundary of its range gets checked before it gets written down.
- **Stage explicit paths.** No `git add -A`.

---

## M1 — Report 28 Test 2, extended to the corpus

**Question.** Does a client's own segmentation of the decoded pixels match the *transmitted* partition?

**Status today.** n = 2. Boundary Jaccard **0.4069** (kodim01) and **0.2213** (kodim23), labelled indicative in report 28.

**Protocol.** The same ten Kodak images Test 1 uses (01, 02, 04, 06, 09, 12, 15, 18, 21, 23). Client-side arm: this study's own merge run on the WebP-decoded pixels at matched fidelity — deliberately the *generous* case, unchanged from report 28. Regions matched by nearest region count, since a client has no fidelity target. Code exists: `28-freemask/pcmp.go`.

**Metric.** Boundary Jaccard, primary. Pair agreement reported alongside but **not** as the headline — report 28 already established it reads 85–87% because interior pixel pairs are non-boundary in every plausible segmentation.

**Pre-registered decision rule.**

| outcome | reading |
|---|---|
| All ten land in a band comparable to Test 1's 0.238–0.402 | Report 24's baseline is corpus-backed. Stop defending it; it is done |
| **Any image ≥ 0.60** | The shipped partition is close to what a client would compute anyway on that content. The capability claim needs re-argument, and the content property that produced it becomes the finding |
| Mean ≥ 0.60 | The capability argument is falsified in its current form. Report 24 stands on bytes; report 13's identity argument does not |

**What this does not decide.** Nothing about segmentation *quality*. A mask can be perfectly stable and still follow the wrong things.

**Cost.** Low. Code and corpus exist; encode is ~10 s per image at 768×512 (report 25).

---

## M2 — The free mask with a segmenter that is not ours

**Question.** Report 28 handed the client *our own merge*. Its own caveat: this bounds free-mask agreement **from above**. What does a realistic client get?

**Protocol.** Repeat Test 1 (stability across two deliveries) with the client arm replaced by an off-the-shelf segmenter. At least one of SLIC or Felzenszwalb. Both deliveries segmented by the *same* library and settings — this measures instability from the **delivered bytes**, not from tool diversity, which would be a different and easier claim.

**Knobs, named in advance.** The off-the-shelf segmenter is tuned to produce a region count within ±20% of our merge's at the matched-fidelity mark. Neither side gets a knob the other is denied. Segmenter version and parameters recorded in the data companion.

**Metric.** Boundary Jaccard, as M1.

**Pre-registered decision rule.**

| outcome | reading |
|---|---|
| Agreement **drops** below report 28's 0.330 mean | Expected. The finding strengthens in its realistic case |
| Agreement roughly unchanged | Fine. The generous case was not generous; report 28's number stands as-is |
| Agreement **rises** materially | Treat as a suspected broken test first, not a result. Check for a self-comparison, a shared cache, and matched inputs before writing anything down |

**What this does not decide.** Same as M1 — stability, not quality.

**Cost.** Low. One Python dependency; no new corpus.

---

## M3 — Boundary recall against human ground truth. **This one decides the niche.**

**Question.** Report 14 judged the regions meaningful **by eye, on three windows of one photograph**. That is the weakest evidence in this record, and the entire photo-editing case rests on it. Are the regions actually good, measured against human-drawn boundaries, and how do they compare to what a photo editor could run locally for free?

### The metric, and why it is not the conventional one

This is the most important paragraph in this file, and it is the one most likely to be read as self-serving. Registering it before the run is the only defence.

For a **selection** use case the two error directions are not symmetric:

- **Over-segmentation is free.** Report 14 found the sky is 2 regions at 1,383. Selecting it is a union of two ids. Splitting one object across many regions costs the user nothing.
- **Under-segmentation is fatal.** If a single region straddles the horizon, no sequence of selections recovers the boundary. The information is gone.

The standard boundary F-measure penalises over-segmentation, which is a cost this format does not pay. Scoring on it would fail a test we never needed to pass.

**Primary metrics, registered:** boundary **recall** at fixed region count, and **under-segmentation error**.

**Also published, unconditionally:** the conventional boundary **F-measure** on the same runs. It penalises us for something harmless, and it is the number a sceptical reader will ask for. Reporting only the flattering metric would make the flattering metric worthless. Both go in the report, with this justification attached.

### Protocol

**Corpus.** Requires human boundary annotations, which Kodak does not have — this is a **different corpus from every other report in this study**, and that discontinuity must be stated in report 31 rather than glossed. BSDS500 is the obvious candidate and the first thing to check.

> **TODO before the run — do not assume.** Verify the corpus exists, is fetchable, its licence permits this use, and what its annotation format actually is. Record the resolved answer here as an amendment. If no annotated corpus is usable, M3 cannot run as written and that is itself a finding to report, not a reason to substitute a weaker test.

**Our arm.** The merge's scale space, not a single point. We have 227 → 1,383 → 11,121 → … already built in `hdMarks`; that gives a **recall-vs-region-count curve** for free. Report the whole curve.

**Comparison arm.** The strongest freely available automatic segmenter at the time of the run, in its automatic (no-prompt) mode, since a photo editor's "free mask" is unprompted.

> **TODO before the run — do not assume.** Do not bake a model name in from memory. Check what the current strongest freely available option is when the run happens, record the exact model, version and weights in the data companion, and state the check in the report. A stale comparison arm is a steelmanning failure, which this study has already committed once (falsification #11).

**Matching.** SAM-family generators emit a variable mask count; our merge emits a curve. Compare at the nearest region count, and plot both. Neither side is scored at a region count the other was denied.

### Pre-registered decision rule

Anchored on a number already in the record: at 11,121 regions the median region is **34 px of texture speckle** (report 14). Past roughly that density, "selecting a region" stops meaning anything to a user, so recall bought by going finer is not recall a photo editor can spend.

| outcome | reading |
|---|---|
| Our curve reaches the comparison arm's recall at a region count within **~2×** of its mask count, with under-segmentation error no worse | **Photo editing is live.** The branch is viable and gets its own roadmap |
| We need **an order of magnitude** more regions to match its recall | **Photo editing is dead as a selection pitch.** The regions are cheap and stable but not the thing a user wants to select. This is an asset format |
| We match on recall but lose badly on under-segmentation | The worst case for us, and the most useful to know: the regions merge across things users care about. Photo editing dead, and it is a warning for the asset path too |
| We beat the comparison arm on recall at comparable counts | Check the test before celebrating. Then this is the strongest result in the study and the roadmap changes |

**What this does not decide.** Nothing about the gaming/asset branch. That rests on stable identity (M1, M2) and primitive count — 32,924 rects to 1,685 regions at matched fidelity — neither of which requires the regions to be semantically correct. **A dead M3 does not weaken the asset case.** It removes a second option.

**Cost.** Materially higher than M1 and M2: a new corpus, model weights, and a comparison arm to verify rather than assume. Budget a day, not an afternoon.

---

## Out of scope for this phase

Deliberately not running, so nobody reads the absence as an oversight:

- Perceptual metrics (SSIMULACRA2, butteraugli). Real gap, unrelated to the niche question.
- Any byte lever. The queue is closed and the closures are proofs, not hunches — see `HANDOFF.md`.
- The region editor (H1), a decoder outside Go (H3), truncation (H4), encoder memory (H5). Queued, after Phase 1.

## Phase 1, which starts regardless of the outcome

**SHPC v2 — alpha and a colour-space tag.** Both are prerequisites for either niche, so they are not gated on M1–M3.

Design calls recorded here so the reasoning is not reconstructed later:

1. **Alpha is per-region, not per-pixel.** One value per region alongside colour — cheap against three channels of colour, and it is what a cutout asset wants: a hard exact silhouette at alpha 0. Per-pixel alpha breaks the piecewise-constant model outright.
2. **The merge does not know about alpha, and that is a trap.** The merge is driven by colour SSE, so nothing currently stops it merging across a cutout boundary and smearing a silhouette. The intended fix is a hard constraint — never merge across an alpha edge. **Untested. It needs a check before it is believed**, and it is exactly the class of defect that looks fine until a sprite has a halo nobody can explain.
3. **Version bump, not a chunk system.** SHPC v1 is `magic "SHPC"(4) · version(1) · uvarint W, H, nregions, wallLen, colourLen · coef(8) · wall chunk · colour chunk` (report 21). The version byte already exists; two new fields bump it to v2. A TLV chunk list is the right answer when a *third* optional field appears, not before.
4. **Known chain.** The colour chunk needs external `brotli` at both ends. Harmless now; it lands on H3 (a decoder outside Go) when that starts.

## Amendment rule

This file is **append-only below this line**. To change a protocol, a metric or a threshold: append a dated amendment stating what changed, why, and **whether any data had been seen at the time**. Never edit a rule in place. An amendment written after seeing a result is not disqualifying — concealing that it was is.

### Amendments

#### Amendment 1 — 2026-07-30 — alpha is reopened; both approaches documented, neither chosen

**No data had been seen. M1, M2 and M3 have not run.** This amends Phase 1 only.

**What changed.** Phase 1 item 1 recorded a decision — "alpha is per-region, not per-pixel" — on a design argument, before any measurement. The owner's instruction is to research and document **both**, and to record them so either can be picked up later. That is the correct call: item 1 was a preference stated as a conclusion, which is the same defect report 26 identified in `bitsPerEdge` ("a tuning parameter that looks like a measurement").

**Where it lives now.** [`DESIGN-ALPHA.md`](DESIGN-ALPHA.md) — approach A (per-region flat), approach B (per-pixel plane), C (a header mode field carrying both, which is also the answer to "can alpha be optional?" — yes, mode 0), and D (three recorded-not-pursued ideas). Research items A1–A5, plus a pilot for A3 with its caveats. Phase 1 item 1 above is **superseded**; items 2, 3 and 4 stand.

**One correction to the text above.** Item 1 said per-pixel alpha "breaks the piecewise-constant model outright". Too strong: colour would stay piecewise-constant and only alpha would change representation. A hybrid, not a contradiction — harder to justify, not impossible.

**One thing that got sharper, not weaker.** Item 2's hazard — the merge is colour-SSE-driven and would happily dissolve a silhouette — stands, and is now A1, the cheapest item in the study. It is still **untested**, and the possibility that the merge already preserves silhouettes without any constraint is a better outcome than implementing one.

**Nothing in M1, M2 or M3 is affected.** Their protocols, metrics and thresholds are unchanged.

**Also added this session, and it governs the above:** [`PRINCIPLES.md`](PRINCIPLES.md). Principle 7 — *other formats inspire us, they do not set our bar* — is why the alpha question was re-asked from scratch rather than answered by copying PNG. Its stated exception is why M3 still compares against the strongest free segmenter: where a free alternative competes for the same user, that is product competition, and refusing to measure it would be avoiding the result rather than pioneering.

#### Amendment 2 — 2026-07-30 — Phase 1 alpha is built; A1's proposed fix is retracted on measurement

**Still no data seen for M1, M2 or M3.** They have not run and their protocols, metrics and thresholds are untouched. This amendment records Phase 1 work only.

**Built.** Alpha now travels the whole pipeline — load → merge → container → decode — as **SHPC v2 mode 1** (per-region flat, `DESIGN-ALPHA.md` approach A). Mode 2 (per-pixel plane) is reserved in the header and rejected by the decoder, because A3 has not shown that real game art needs it. Round-trips bit-exactly on three sprites, alpha included.

**Phase 1 item 2 is retracted on measurement.** It said the merge would need an explicit "never merge across an alpha edge" constraint. A1 showed the merge was never the defect — `load()` discarded alpha and premultiplication turned transparency into black — and **A1b now shows the constraint is unnecessary**: carrying alpha alone takes silhouette dissolution from 16–62% to **0.00% at every usable mark**. The only residual is 10.66% at one sprite's coarsest rung, where the merge sees the alpha difference and trades it away on rate-distortion grounds. That is a priced decision, not a loss. The constraint stays available as an override for extreme coarsening; it is not needed to make the format correct.

**Phase 1 items 3 and 4 stand and are now measured.** The version bump cost **exactly the one predicted byte** on an opaque image (816 B vs v1's 815 B), and v1 files still decode bit-exactly under the new decoder. The `brotli` dependency noted in item 4 is unchanged and now also carries the alpha chunk.

**The guarantee every published number rests on, verified rather than asserted.** An image with no alpha takes the identical path it took before alpha existed: the merge carries alpha as a fourth SSE channel, and a *constant* fourth channel contributes exactly zero to every `dSSE` term. Checked against a pre-change binary built from a git worktree — **13 of 13 renders byte-identical** across a full scale-space — and locked by `TestAlphaOpaqueIsInert`.

**Still open in Phase 1:** the colour-space tag. Not started.
