# 09 — Cross-domain fan-out (crypto, space, atoms)

Deliberately far-afield ideas — from disciplines with no connection to color —
mapped onto palette selection, implemented, and measured against the champion
(OKLab-matched PCA-divisive-init + Lloyd refine). Per the standing method, the
discards are kept with their numbers. Raw:
[09-crossdomain-data.txt](09-crossdomain-data.txt). Code: `bench/crossdomain.go`.

**Headline: one genuine win.** A **space-filling-curve initializer (crypto/database
locality hashing)** beats the champion at N=256 on **all six images**. The
astrophysics (MST/Friends-of-Friends) and statistical-mechanics (deterministic
annealing) transfers were measured and **discarded**.

## The idea catalogue (discipline → mapping)

| Discipline | Concept | Mapping to palette selection | Tested? |
|---|---|---|---|
| **Crypto / databases** | space-filling curves (Morton/Z-order, Hilbert), locality-sensitive hashing | linearize 3-D color preserving locality; cut the run into N equal-weight segments | ✅ **kept** |
| **Astrophysics** | Friends-of-Friends / MST halo finding (cosmic web) | single-linkage clusters of the color cloud; cut the N−1 longest MST edges | ✅ discarded |
| **Statistical mechanics (atoms)** | deterministic annealing — free energy, phase transitions | soft assignment with a cooling temperature; hardens into k-means at T→0 | ✅ discarded |
| Crypto / coding theory | lattice VQ (A*, D*, BCC lattices) | fixed perceptual-lattice codebook | ✗ (report 01: loses adaptivity) |
| Quasi-Monte-Carlo (finance/rendering) | low-discrepancy sequences (Sobol/Halton) | low-discrepancy centroid seeding | ✗ (init shown second-order; deprioritized) |
| Astrophysics | Voronoi binning (Cappellari–Copin, target-S/N) | adaptive palette allocation by "signal" | ✗ (complex; queued) |

## Results — mean ΔE2000 over six images (OKLab-matched, no dither)

| piece | N=16 | N=64 | N=256 | discipline |
|---|---|---|---|---|
| refine-oklab (prior champion) | 4.407 | **2.9083** | 1.9133 | — |
| refine (RGB-matched) | **4.292** | — | — | — |
| **spacecurve-refine** (Morton init + Lloyd) | 4.474 | 2.9120 | **1.8819** | crypto/db |
| spacecurve (no refine) | — | 3.267 | 2.084 | crypto/db |
| MST / Friends-of-Friends | — | 5.646 | 4.006 | astrophysics |
| MST + Lloyd refine | — | 3.434 | 2.154 | astrophysics |
| deterministic annealing | — | 4.133 | 3.802 | stat-mech |
| *pngquant (reference)* | 4.440 | 2.989 | 1.981 | — |

Per-image at N=256, **spacecurve-refine beats the champion on all 6** (athens 2.088<
2.092, liberty 2.098<2.131, monet 1.544<1.563, napoleon 1.893<1.943, pearl 1.455<
1.501, starry 2.205<2.249). Deterministic across runs (Δ≈0.001, float-order only).

## The keep — space-filling-curve initialization (crypto/databases)

**spacecurve-refine is the new best at N=256** (1.882 vs champion 1.913, vs pngquant
1.981 — **5% under the bar**), and ties the champion at N=64. The Morton Z-order
curve linearizes OKLab while preserving locality; cutting the curve into N
equal-weight segments gives a **spatially uniform** initial centroid set, which seeds
Lloyd in a better basin than variance-driven recursive splitting.

**Why it helps only at high N** — and is *worst* at N=16 (4.474): the benefit is
uniform *coverage*, which matters when there are many centroids to place (256) and is
counterproductive when there are few (16), where a variance-aware coarse split wins.
So the curve-init is a **high-N booster**, not a universal default. Mechanism is the
same family as the OKLab win: both help most where palette entries are dense.

This is the payoff of reaching outside color science: locality-preserving
space-filling curves (databases/GPU/crypto) are not in the color-quantization
canon, yet curve-init + refine is the best N=256 result in this whole record.

## The discards — measured, with reasons

- **MST / Friends-of-Friends (astrophysics) — DISCARD.** Catastrophic raw (5.6 / 4.0)
  and still bad after refine (3.4 / 2.2). Single-linkage **chains**: one bridge of
  intermediate colors fuses what should be separate clusters, so the cut leaves huge
  heterogeneous components. A known single-linkage failure mode; the cosmic-web
  halo-finder does not transfer to color. (Our Ward-linkage PNN in report 08 was the
  better agglomerative variant and *also* lost — agglomerative clustering is just not
  the tool here.)
- **Deterministic annealing (statistical mechanics) — DISCARD.** Much worse (4.1 /
  3.8). The principled escape-from-local-minima benefit did not materialize — exactly
  what report 08's multi-restart tie predicted: at N=256 the minima are shallow, so
  the expensive free-energy machinery buys nothing, and its soft-assignment phase can
  leave centroids under-separated if the cooling schedule isn't perfectly tuned.
  Even granting schedule sensitivity, a method that needs careful tuning to *maybe*
  match a cheap init+refine is discarded.
- **spacecurve without refine — DISCARD.** Worse than refine everywhere (3.27 / 2.08);
  the curve is a good *initializer*, not a finished palette. Refinement still carries
  the quality (consistent with reports 05/08).

## Updated stack (best measured, all beat pngquant)

| N | best config | mean ΔE2000 (vs pngquant) |
|---|---|---|
| 16 | RGB-matched refine | 4.292 (4.440) |
| 64 | OKLab-matched refine | 2.908 (2.989) |
| 256 | OKLab-matched **spacecurve-init** refine | 1.882 (1.981) |

A per-N dispatch (RGB→OKLab→OKLab+curve-init as N grows), or a single everywhere-
config (OKLab-matched refine) that already beats pngquant at every N with curve-init
as the high-N upgrade.

## Caveats / next

- Six paintings; **CQ100 scale-up** still the claim-maker (could shift the N=64 tie,
  not the N=256 6/6 result).
- Tiny float-order nondeterminism in refine → the **canonical histogram sort** (report
  05) makes it exact; required for Phase-5 golden tests.
- Untested cross-domain: Hilbert curve (better locality than Morton — may extend the
  curve-init win to lower N), Voronoi-binning, low-discrepancy seeding. Queued, lower
  priority than CQ100.

## Addendum — scaling to N=2048 (3 images)

Champions swept from N=64 to 2048 (pngquant caps at 256; ImageMagick is the only
external reference above). Mean ΔE2000; raw: [09b-scaling-sweep-data.txt](09b-scaling-sweep-data.txt).

| N | pngquant | ImageMagick | refine-RGB | refine-OKLab | scurve-OKLab |
|---|---|---|---|---|---|
| 64 | 3.090 | 3.406 | 3.088 | 2.980 | **2.976** |
| 128 | 2.494 | 2.750 | 2.602 | 2.416 | **2.412** |
| 256 | 2.037 | 2.275 | 2.111 | 1.968 | **1.947** |
| 512 | — | 1.813 | 1.687 | 1.564 | **1.557** |
| 1024 | — | 1.461 | 1.346 | 1.247 | **1.238** |
| 2048 | — | 1.156 | 1.055 | 0.978 | **0.972** |

Findings:
- **scurve-OKLab is best at every N from 64 to 2048** — including 64/128, where on
  this 3-image set it edges refine-OKLab (the 6-image N=64 tie was within noise). So
  there is **no real deficit at 64/128**; the only low-N case where curve-init/OKLab
  lose is N=16 (RGB-matched wins there, report 02/09).
- **The champion margins are small in absolute ΔE2000** (scurve vs refine-OKLab:
  0.003–0.021 — all *sub-JND*, invisible). At N≥64 every competent method is already
  in a low-error regime; the win is "match/beat the state of the art deterministically,"
  not a large visible jump.
- **We beat both incumbents at every N**: pngquant by ~0.07–0.11 (and it cannot exceed
  256 at all), ImageMagick by ~0.18–0.43.
- **OKLab-matched is the consistent lever** (refine-OKLab << refine-RGB everywhere,
  ~0.1–0.13); curve-init is a small, free extra on top.

Conclusion: the algorithm fan-out has saturated — new families either tie within noise
or lose. The remaining value is **validation at scale (CQ100)** and **shipping** (Phase
5), not more selection variants.
