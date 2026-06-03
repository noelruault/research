# 08 — Interdisciplinary pieces, measured (and discarded)

Report 07 ranked interdisciplinary candidates on *literature reasoning*. A discard
isn't earned until it's measured, so this report **implements the tractable ones and
benchmarks them against the champion** (OKLab-matched PCA-divisive init + Lloyd
refine, report 02) on the six paintings, OKLab-matched assignment, no dither. Raw:
[08-interdisciplinary-measured-data.txt](08-interdisciplinary-measured-data.txt).
Implementations: `bench/extras.go` (+ error-weight flag in `kmeans.go`).

**Result: none reliably beats the champion. All discarded — with numbers.**

## Mean ΔE2000 over six images (lower = better)

| piece | N=64 | N=256 | verdict |
|---|---|---|---|
| **refine-oklab (champion)** | 2.9083 | **1.9133** | the bar |
| PNN init + refine | **2.9000** | 1.9940 | **DISCARD** |
| multi-restart keep-best (R=8) | 2.9179 | 1.9159 | **DISCARD** |
| error-weighted refine | 2.9373 | 1.9269 | **DISCARD** |
| HyAB metric | 2.9814 | 1.9569 | **DISCARD** |
| PNN init only (no refine) | 3.4965 | 3.0535 | **DISCARD** |
| *pngquant (reference)* | 2.9891 | 1.9813 | champion beats it at both N |

## The discards, each with its measured reason

- **PNN (Pairwise Nearest Neighbor) init — DISCARD.** The best-known deterministic VQ
  initializer (rank 3 in report 07). It marginally *wins* at N=64 (2.9000 vs 2.9083,
  inside the noise band) but **loses clearly at N=256** (1.9940 — worse than even
  pngquant). Cause: PNN needs a binned start to be tractable (`pnnCap=2000`), and
  binning to ~2000 colors before merging to 256 throws away the precision that a
  full-histogram PCA-divisive init keeps at high N. The one interdisciplinary piece
  that *could* have been a keep is sunk by the tractability prepass it requires. Not
  worth its complexity.
- **Multi-restart keep-best — DISCARD.** The deterministic substitute for simulated
  annealing (rank 5). Worse than the champion at N=64 (2.9179) and a statistical tie
  at N=256 (1.9159 vs 1.9133, +0.0026). **This is the measured confirmation of report
  07's thesis:** at N=256 with a good seed, k-means' local minima are shallow, so
  escaping them buys nothing. Eight restarts for ~zero gain — discard (and, by
  extension, the heavier escape methods DA / neural gas / SA that target the same
  shallow minima at higher cost).
- **Error-weighted refinement — DISCARD (for this metric).** libimagequant's
  hard-cell-reweighting trick makes our number *worse* at both N (2.9373 / 1.9269).
  **Exactly the objective caveat from report 06, now measured:** it pulls palette
  toward high-error edge colors, improving *perceived* quality while raising
  *unweighted* mean ΔE2000. Correctly belongs to a separate perceptual-objective
  track, not this one.
- **HyAB metric — DISCARD.** Worse at both N (2.9814 / 1.9569). HyAB is tuned for
  *large* color differences; dense palettes (N≥64) live in the small-difference
  regime where OKLab-Euclidean is the better proxy for ΔE2000. The literature's
  large-difference advantage doesn't apply here — measured.
- **PNN init only (no refine) — DISCARD.** Far worse (3.50 / 3.05); reconfirms report
  05 that the refinement, not the init, carries the quality.

## What this settles

1. **The champion stands.** OKLab-matched PCA-divisive-init + plain Lloyd refine is
   best-or-tied at every N; no interdisciplinary refinement improves it on unweighted
   mean ΔE2000. We tried the shortlist and it held.
2. **"Optimize harder" is a dead end at N=256** — multi-restart ties, error-weighting
   hurts. The N=256 win came from the right *space* (OKLab-matched), not a better
   optimizer. The heavy escape methods (deterministic annealing, neural gas, GMM,
   ECVQ) are left literature-discarded; the multi-restart result is the cheap
   empirical proxy that they wouldn't pay either.
3. **The only remaining lever is a different objective.** Every measured attempt to
   beat the champion on *unweighted mean ΔE2000* failed; the one trick that does more
   (error weighting) optimizes *perceived* quality. So further gains require the
   perceptual-objective track (report 06) with its own metric — not more clustering
   cleverness here.

## Caveats

- Six paintings; CQ100 scale-up still pending (it could shift the hair's-breadth
  N=64 ordering, not the N=256 verdicts).
- PNN was tested binned (`pnnCap=2000`); an un-binned fast-PNN (kNN-graph) might
  recover N=256 quality, but that's a large build for a piece already beaten by the
  simpler champion — explicitly not pursued.
- "Discard" here means *for unweighted mean ΔE2000*. Error-weighting and HyAB may
  win under a perceptual/edge-weighted metric; that's the separate track.
