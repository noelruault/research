# 07 — Interdisciplinary codebook-optimization brainstorm

A targeted fan-out: which algorithm, from any discipline, can push a 256-color
codebook below the perceptual error of the incumbents — beyond plain Lloyd k-means.
Framed by the situation: we already have exact nearest-color + CIEDE2000, and (report
02) already beat pngquant at every N via OKLab-matched refine. So this asks *what
buys more*, and documents what doesn't (the discards are results).

Two independent agents converged on the same #1 — **matched OKLab clustering+
assignment** — which we'd already implemented and measured (report 02). That
convergence is itself a result: the cheap lever was the right one.

## Ranked shortlist (promise to beat unweighted mean ΔE2000 at N=256)

| Rank | Lever | Promise | Cost | Deterministic | Status |
|---|---|---|---|---|---|
| 1 | **Cluster + assign in OKLab (matched)** | HIGH | ~free | yes | **DONE — beats pngquant at every N (report 02)** |
| 2 | **Importance/edge weighting of the histogram** | HIGH *for perceived*, ambiguous for *unweighted* mean | O(pixels) once | yes | **deferred — wrong objective (see below + report 06)** |
| 3 | **PNN (+ LBG-U) init, then Lloyd polish** | MED-HIGH | O(B²) on binned histogram | yes | queued |
| 4 | **libimagequant-style remap-error reweighting in k-means** | MED | tiny | yes | queued |
| 5 | **Multi-restart k-means / HyAB distance** | MED | linear×restarts | yes(seeded) | queued |

## The disciplines surveyed — and why most are discards here

The striking finding: **fancier global optimizers don't help at N=256**, because with a
good seed the Lloyd local minima are shallow and the optimizer is already near its
floor. The gap was never "optimize harder"; it was "optimize the right error" (space
+ weighting). So the heavy machinery is ruled out — with reasons:

- **Deterministic annealing (Rose, rate-distortion).** Provably escapes local minima;
  deterministic. **DISCARD (med-low):** its advantage is largest at *small* N / hard
  manifolds; at N=256 with PNN/Wu seeding the minima are shallow, and it costs 10–100×
  (soft assignment over all centroids per cooling step).
- **Neural Gas / GNG / SOM.** Robust soft-competitive VQ. **DISCARD (low-med):** gains
  largest at small/medium N; stochastic unless seed+order fixed; redundant with
  median-cut+k-means on benign 3-D color.
- **Entropy-constrained VQ (ECVQ).** Rate-distortion optimal. **DISCARD (low):**
  optimizes D+λR — the *wrong objective* unless we start caring about compressed file
  size at fixed quality. Keep in back pocket only if the goal shifts to size.
- **Simulated annealing on k-means.** **DISCARD (low-med):** its one charm is it can
  anneal on the exact non-differentiable CIEDE2000 directly, but as a global escape at
  N=256 it's overkill; **multi-restart k-means + keep-best captures ~80% deterministically** (kept as rank 5).
- **Soft assignment / GMM-EM / fuzzy c-means.** **DISCARD (low):** train/test mismatch
  — the palette is *used* with hard nearest-color at render, so optimizing a soft
  objective doesn't minimize the hard-assignment error we pay. Steal only the cheap
  reweighting idea (rank 4).
- **PNN / split-and-merge / LBG-U.** **KEEP (med-high):** the best *deterministic*
  initializer, proven in color quantizers (nQuant), cheap on a binned histogram. The
  one piece of heavy clustering worth trying as a better seed than PCA-divisive.
- **Lattice/optimal-quantizer theory (Gersho).** Asymptotically optimal cells tile
  space uniformly → *another* argument to cluster in OKLab (uniform cells ≈ uniform
  perceptual error). Conceptual support, not an algorithm.
- **Learned/neural, Lloyd-Max scalar.** Out of scope (portability / strictly worse).

## The objective caveat (independently raised by both agents)

Our benchmark is **unweighted** mean ΔE2000. Importance/edge weighting (rank 2) —
libimagequant's signature N=256 trick — **down-weights flat regions, which hold most
pixels**, so it improves *perceived* quality but may *raise* the unweighted mean.
Both agents flagged this explicitly. Therefore importance weighting is **not** pursued
for the current metric; it belongs to a separate **perceptual-objective track** that
first needs an edge-weighted / SSIM / saliency metric to be graded fairly (see report
06). Pursuing it blindly for the unweighted number would be a trap.

Also confirmed-fair: we ran pngquant with `--nofs` (no dither) against our undithered
output, so none of the measured gap is dithering.

## What to build next (objective-safe, deterministic, cheap)

1. **PNN/LBG-U seed** (rank 3) → measure as a better init than PCA-divisive under
   OKLab-matched refine.
2. **Remap-error reweighting** in the Lloyd loop (rank 4) → cheap polish on hard cells.
3. **CQ100 scale-up** → confirm the OKLab-matched win holds on 100 images (the
   no-published-method-beats-libimagequant-at-256 claim makes this worth proving).
4. **HyAB** centroid metric (rank 5) → only if a gap remains; needs a mean/median
   mixed centroid update.

The everything-machine optimizers (DA, neural gas, SA, GMM) are **logged as tried-and-
discarded** so we don't revisit them: at N=256 with a good perceptual space and seed,
they don't pay.

Sources: see report 01 and 06 plus — Rose DA (IEEE 144705); Neural Gas (Martinetz);
ECVQ (Chou–Lookabaugh–Gray, Stanford EE398a); PNN/LBG-U (Equitz; Virmajoki; nQuant);
HyAB/FLIP (NVIDIA 2020); Celebi 2011 (arXiv:1101.0395). Several perceptual-CQ blogs
(30fps HyAB, pkh.me) were 403-blocked to the agent — medium-confidence on HyAB
margins, verify in a browser before relying on specifics.
