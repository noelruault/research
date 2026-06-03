# 02 — Piece P1: color space (and the assignment-space fix)

The fair rematch report 05 demanded. Report 05 (D3) ruled OKLab *out* — but under
an **unfair test**: it clustered in OKLab yet *assigned* pixels in RGB, so the
palette was optimized for one geometry and used in another. This report fixes that:
**cluster AND assign in the same space (matched).** Raw output:
[02-pieces-color-space-data.txt](02-pieces-color-space-data.txt).

This matters because it is engine-compatible: Euclidean distance in OKLab is still
Euclidean, so pixelize's existing kd-tree works unchanged — just built over
OKLab-transformed palette entries and queried with OKLab-transformed pixels. No new
matcher, no brute-force ΔE2000 in the hot loop.

## Setup

Six paintings, no dither, mean CIEDE2000. Three configs, all using the
PCA-divisive-init + 10 k-means-refine recipe (report 05's winner), vs pngquant as
the bar:

- **refine (RGB)** — cluster in RGB, assign in RGB (the report-10 quality mode).
- **refine-oklab (matched)** — cluster in OKLab, assign in OKLab.
- **pca-oklab (matched)** — deterministic OKLab divisive, assign in OKLab.

## Result — mean ΔE2000 (lower = better)

| N | pngquant | refine RGB | **refine OKLab-matched** | pca OKLab-matched |
|---|---|---|---|---|
| 16 | 4.440 | **4.292** | 4.407 | 4.615 |
| 64 | 2.989 | 2.958 | **2.908** | 2.998 |
| 256 | 1.981 | 2.058 | **1.913** | 1.999 |

Per-image at N=256, **refine-OKLab beats pngquant on all 6** (athens 2.092 vs 2.180,
liberty 2.131 vs 2.231, monet 1.563 vs 1.622, napoleon 1.943 vs 2.018, pearl 1.501
vs 1.527, starry 2.249 vs 2.310).

## Findings

1. **Matched OKLab assignment beats pngquant at every tested N** (4.407<4.440,
   2.908<2.989, 1.913<1.981). The single config `refine-oklab` is now an
   everywhere-win over the quality bar — the result report 10 was missing at N=256.
2. **The benefit grows with N** — OKLab loses to RGB at N=16 (4.407 vs 4.292), ties-
   to-wins at N=64, wins decisively at N=256. **Mechanism:** at small N the palette
   entries are far apart (large color differences), the regime where OKLab is *not*
   especially uniform and RGB's ordering is fine; at large N entries are close (small
   differences), exactly where OKLab is perceptually uniform, so Euclidean-in-OKLab
   directly minimizes ΔE2000. This is the textbook small-vs-large-difference split,
   now measured on our corpus.
3. **Assignment space, not just clustering space, is the lever.** Report 05's D3 loss
   was entirely the RGB-assignment mismatch; with assignment matched, the verdict
   flips. Clustering-space and assignment-space must agree.
4. **Refinement still dominates the init** — `pca-oklab` (deterministic, no refine)
   trails `refine-oklab` everywhere, consistent with report 05.

## The updated stack (beats pngquant at every N)

- **N ≤ ~32 → RGB-matched refine** (best at N=16).
- **N ≥ ~64 → OKLab-matched refine** (best at N≥64; the crossover is ~N=32–64,
  to be pinned).
- A **single everywhere-config** (`refine-oklab`) already beats pngquant at all
  three N if we don't want a per-N switch; the per-N pick just adds a little at N=16.

Deterministic default stays `pca` (RGB) for reproducibility; the **perceptual
quality mode is OKLab-matched refine**, exposed via pixelize's pluggable distance
(an OKLab kd-tree), which the engine already supports.

## Engine note & caveats

- **kd-tree compatible:** build the tree over OKLab coords; same code, opt-in via
  DistanceFunc. The default fixed-palette path stays RGB-exact.
- **Determinism:** OKLab-matched refine inherits the histogram-order caveat (report
  05) — canonical histogram sort required for reproducibility.
- **Open:** pin the RGB→OKLab crossover N; test **HyAB** and **exact ΔE2000-greedy
  assignment** as an upper bound (slower, not kd-tree-able) to see how much headroom
  remains above OKLab-Euclidean; and validate at scale on CQ100. The brainstorm
  agents (libimagequant dissection + interdisciplinary codebook optimization) feed
  the next round — perceptual *error weighting* is the likely next lever.
