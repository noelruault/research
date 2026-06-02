# 05 — Selection fan-out + judge

The puzzle fan-out for piece P3 (palette selection), crossed with P1 (color
space) and P4 (k-means seeding): **12 quantizer configurations, every one measured
on the same six-image corpus at N∈{4,16,64,256}, all results in one table so the
judge sees winners and losers together.** Raw output:
[05-pieces-fanout-data.txt](05-pieces-fanout-data.txt).

Per the brief: the discards matter as much as the winners. Ruling a piece out —
with the measured reason — is a result we keep, so we don't re-litigate it.

## The full board — mean ΔE2000 (↓) / p95 ΔE2000 (↓)

| configuration | N=4 | N=16 | N=64 | N=256 |
|---|---|---|---|---|
| popularity *(floor)* | 22.42 / 53.3 | 17.06 / 44.4 | 14.15 / 40.9 | 12.01 / 37.5 |
| median-cut *(classic)* | 7.96 / 17.8 | 4.85 / 11.2 | 3.28 / 7.78 | 2.25 / 5.64 |
| divisive · axis-aligned · rgb | 7.73 / 16.8 | 4.76 / 10.3 | 3.28 / 7.42 | 2.30 / 5.33 |
| **divisive · pca · rgb** | **7.18** / 16.2 | **4.55** / 9.89 | 3.23 / 7.43 | 2.29 / 5.56 |
| divisive · pca · oklab | 7.37 / 16.7 | 4.73 / 10.8 | 3.21 / 7.40 | 2.23 / 5.24 |
| kmeans · random · rgb | 7.24 / 15.9 | 4.50 / 9.54 | 3.16 / 7.14 | 2.21 / 4.98 |
| kmeans · kmeans++ · rgb | 7.26 / 15.3 | 4.53 / 9.62 | 3.12 / 7.07 | 2.13 / 4.95 |
| kmeans · maximin · rgb | 7.27 / 15.6 | 4.68 / 9.69 | 3.28 / 7.23 | 2.40 / 5.41 |
| kmeans · maximin · oklab | 7.40 / 15.9 | 4.78 / 10.6 | 3.28 / 7.32 | 2.31 / 5.11 |
| **kmeans[median-cut] · rgb** | 7.22 / 16.2 | 4.46 / 9.61 | **3.08** / 7.18 | **2.09** / 5.09 |
| **kmeans[divisive·pca·rgb] · rgb** | **7.17** / 16.0 | **4.44** / 9.51 | 3.12 / 7.07 | 2.15 / 5.03 |
| kmeans[divisive·pca·oklab] · oklab | 7.38 / 16.0 | 4.57 / 10.0 | 3.13 / 7.14 | 2.12 / **4.84** |

(All seeded-k-means rows carry the determinism caveat below; treat their last
digit as ±0.05 noise. Divisive rows are bit-stable.)

## Winners

1. **Init + k-means refine wins at every N — and the init barely matters.** The two
   refined configs (`kmeans[divisive·pca·rgb]` and `kmeans[median-cut]`) are best or
   tied-best from N=4 to N=256 (e.g. N=16: 4.44 / 4.46). They converge to ≈the same
   place regardless of which decent init seeds them. **The refinement is the value,
   not the init.** This is exactly the libimagequant recipe (divisive init + k-means
   refine), now measured and confirmed on our corpus.
2. **k-means++ alone is the simplest near-best** — within ~0.02–0.05 ΔE of the
   refined winners at every N (N=256: 2.13 vs 2.09), with no separate init pass. The
   value-for-complexity champion.
3. **PCA-divisive is the deterministic non-iterative champion.** `divisive·pca·rgb`
   beats classic median cut at N≤64 (N=16: 4.55 vs 4.85) and ties at N=256, with
   zero iterations and **bit-stable** output. It is the obvious deterministic default
   and the best init to seed refinement with.
4. **Two cheap divisive upgrades both pay:** variance-selection > population-selection
   (axis-aligned-variance 4.76 vs median-cut 4.85 at N=16) and PCA-axis > coordinate-
   axis (4.55 vs 4.76). They compound.

## Discards (kept, with the measured reason)

- **D1 — popularity.** 3–7× the error of any real method at every N. Floor only.
- **D2 — maximin seeding. RULED OUT as a primary seeder.** The research (Celebi)
  flagged deterministic Gonzalez maximin as the best-evidenced recipe; **our
  measurement disagrees for mean-ΔE on photos.** At N=256 maximin is the *worst*
  non-popularity method (2.40 rgb / 2.31 oklab) and it never wins at any N. Reason:
  farthest-point seeding optimizes *coverage* (k-center), so it spends palette
  entries on rare extreme/outlier colors and starves the dense regions that dominate
  average error. Good for "miss no color," bad for "minimize average error." A
  surprising, valuable negative — do not seed with maximin for this objective.
- **D3 — OKLab clustering under RGB assignment. RULED OUT for now; needs a fair
  rematch.** Clustering in OKLab does not help and usually hurts slightly
  (`divisive·pca·oklab` 4.73 vs `·rgb` 4.55 at N=16). Reason: the harness (like
  pixelize) **assigns** pixels in Euclidean RGB, so a palette optimized for OKLab
  geometry is mismatched at assignment time. This confirms the documented "Lab
  doesn't automatically help" caveat — *but the test is not yet fair.* Nuance worth
  keeping: at N=256 the OKLab configs win on **p95** (best p95 4.84), so perceptual
  space helps the worst regions at high N. **Required follow-up (report 02):** test
  OKLab clustering *with OKLab assignment*, the only apples-to-apples comparison.
- **D4 — random seeding.** Competitive once refined (4.50 @ N=16) but non-
  deterministic and never better than k-means++. Dominated; discard as a seeder.
- **D5 — classic median cut as the default.** Beaten by PCA-divisive at every N≤64.
  Keep only as the reference baseline and as one viable refinement init.

## Cross-cutting finding — determinism

`divisive·*` is deterministic (probe: `divisive·pca·rgb` = 4.547 on two runs,
identical). **Seeded k-means is *not*, despite a fixed RNG seed** (probe: k-means++
= 4.533 then 4.486). Cause: `histogram()` materializes the point list by iterating a
Go map, whose order is randomized per process, so any order-sensitive step
(k-means++ sampling, maximin tie-breaks, random) drifts. **Engine requirement:**
sort the histogram into a canonical order before any order-sensitive step. With that
one fix, k-means++ and refinement become reproducible — mandatory for golden tests
and stable build maps.

## The stack (puzzle assembly → what pixelize should ship)

- **Deterministic default → `divisive · pca · rgb`** (variance-selected, principal-
  axis, RGB). Reproducible, fast, non-iterative; beats classic median cut.
- **Quality mode → `divisive·pca·rgb` init + k-means refine in RGB**, with the
  canonical histogram sort for determinism. Best mean ΔE at low/mid N, ties the
  field at high N. The confirmed libimagequant-style recipe.
- **Ruled out of the default path:** maximin seeding (D2) and OKLab-under-RGB-
  assignment (D3, pending the fair rematch). k-means++ stays as the simplest
  init-free option once the determinism fix lands.

## Caveats & what this does NOT yet prove

Six-painting corpus (photographic), Euclidean-RGB assignment, fixed 10 Lloyd
iterations, ΔE2000 over the whole image. Open, queued:

- **CQ100 + incumbents** (report 10) — the real shootout vs pngquant/ImageMagick/
  GIMP/Aseprite; this record only ranks *our* pieces against each other.
- **OKLab with matched assignment** (report 02) — the fair D3 rematch.
- **Wu** as an explicit named baseline (the divisive·variance rows approximate it).
- **Refinement-iteration sweep** (report 07) — quality per Lloyd pass; is 10 needed?
- **Optimal 1-D (`Ckmeans.1d.dp`) split** for PCA-divisive (research 01 piece #2) —
  does a provably-optimal split beat the median split along the principal axis?
