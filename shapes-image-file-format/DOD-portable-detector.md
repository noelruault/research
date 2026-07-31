# Definition of Done — a portable detector matching Vision

**Set by the owner, 2026-07-31.** Written before the work, per `PREREGISTRATION.md`'s rule: a target chosen after seeing the result is not a target.

## The goal behind it

A **web editor** that opens an image, scans its objects, and edits their properties. That rules Apple's Vision out completely — not on licensing, but because it cannot run in a browser. So the detector must be portable and redistributable, and it must be good enough to replace Vision.

## Done means

**On the six photographs in the `bg-removal` folder, a portable detector produces the same result as Apple's Vision.**

## How "the same" is measured

Vision's masks for all six are already computed and stored, so it is a fixed reference, not a moving one.

**Primary metric: IoU between the portable detector's mask and Vision's mask, per image, at pixel level.**

| threshold | reading |
|---|---|
| **IoU ≥ 0.90 on all six** | **DONE.** The portable detector is a drop-in replacement for Vision |
| IoU ≥ 0.90 on ≥ 4 of 6 | Partial. Ship it, and record which images fail and why — content class matters more than the mean |
| IoU < 0.90 on ≥ 3 of 6 | **Not done.** Try the next model before concluding anything about the approach |

**Why 0.90 and not higher:** snapping *Vision's own* mask onto our partition already costs 1.7–5.2% IoU (reports 40, 43 — measured 0.948–0.983). A portable detector plus the same snap cannot beat that ceiling, so demanding 0.95+ would fail on our own quantisation rather than on the detector.

**Reported alongside, not as the headline:**
- IoU after snapping to the partition, both arms — the number that matters for the actual pipeline.
- Wall-clock per image. Vision is ~160 ms warm on the Neural Engine; a portable CPU/WASM model will be slower and that is expected, not a failure. **Speed is not part of the DoD.**
- Where they disagree (thin structures, dark subjects, fur, bokeh), because that is the useful finding regardless of the verdict.

## Constraints on the candidate

1. **Redistributable licence** — verified at the source, not assumed. The licence on model *weights* is sometimes not the licence on the code.
2. **Runs off macOS.** ONNX is the target format so the same weights can run server-side today and in `onnxruntime-web` later.
3. **No Apple weights.** They cannot be shipped, which is what started this.

## Explicitly not in scope

- Matching Vision's **speed**. Different silicon.
- Soft mattes. Everything here is a hard region-level selection; sub-region hair is `DESIGN-ALPHA.md` mode 2 and remains unbuilt.
- The web editor itself. This DoD covers the **detector** only; the WASM decoder and canvas UI are separate work.

## RESULT — met, 2026-07-31

**All three candidates cleared the bar.** Data: [`DOD-portable-detector-data.txt`](DOD-portable-detector-data.txt).

| model | mean IoU | ≥0.90 | warm ms | weights | verdict |
|---|---|---|---|---|---|
| **u2net** | 0.9541 | **6/6** | **163–246** | 176 MB | **DONE — the pick** |
| isnet-general-use | 0.9525 | **6/6** | 431–502 | 179 MB | DONE |
| birefnet-general | **0.9643** | **6/6** | 6455–8385 | 973 MB | DONE, but 50× slower |

**u2net matches Vision on speed as well as output** — 163–246 ms warm against Vision's ~160 ms on the Neural Engine — while running on ONNX Runtime, so the same weights go to `onnxruntime-web`. Speed was explicitly *excluded* from the DoD because a portable model was expected to be slower. It wasn't. Recorded because the expectation was wrong in our favour.

**Still open:** the upstream U-2-Net weights licence is **not** verified at source. rembg itself is MIT (verified), and it redistributes the weights from its own releases. The DoD demanded "verified at the source" — that row is not yet that, and it gates shipping, not experimenting.

## Honest caveat on the reference

Vision's mask is the **reference, not ground truth.** There are no human-drawn masks for these six images, so a portable detector could conceivably be *better* than Vision and score lower here. If a candidate misses the threshold, the disagreements get looked at before the candidate is blamed — a rule this study has needed before (falsifications #11, #14).
