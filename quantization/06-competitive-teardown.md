# 06 — Competitive teardown: what libimagequant actually does

A source-level dissection of libimagequant/pngquant (the quality bar) to understand
*why* it was winning at N=256 before our OKLab-matched fix, and which of its
remaining tricks are worth porting — and, importantly, which optimize a **different
objective** than our benchmark measures. Sources are libimagequant's Rust source
(`github.com/ImageOptim/libimagequant/src/`), pngquant.org/lib, and Material Color
Utilities; claims unverifiable from source are flagged below.

## Its pipeline (the parts that matter)

1. **Perceptual working space — not sRGB, not linear.** Everything (variance,
   clustering, diff) happens in a custom gamma space, exponent `INTERNAL_GAMMA=0.57`
   (`src/pal.rs`), ~halfway between sRGB and linear. This is a cheap stand-in for a
   perceptual metric.
2. **Adaptive-posterization histogram** (`src/hist.rs`) — near-full color precision
   unless the histogram explodes, then drops ≤3 bits. Not a fixed 32³ grid.
3. **Per-pixel importance / contrast map** (`src/image.rs`, `contrast_maps()`) — a
   second-difference edge detector → importance `z=(1−max(min(h,v)))⁴` scaled to
   **[80, 256]**, used as a per-pixel **count boost** in the histogram. **Flat pixels
   get ~⅓ the weight of edge pixels.** This is the headline N=256 trick.
4. **Variance-based median cut with error feedback** (`src/mediancut.rs`) — split the
   max-weighted-variance box; box score is multiplied up when its worst pixel error
   exceeds a ramping MSE target (re-split the badly-represented).
5. **Adaptive Voronoi/k-means refine** (`src/kmeans.rs`) — Wu/median-cut seed, a
   small adaptive iteration count, early-stop on `|Δpalette_error|<limit`, with a
   per-color weight `(2w+pw)(0.5+error)` that pulls harder on badly-approximated
   colors.
6. **Per-channel weights** R0.5/G1.0/B0.45 and an **alpha-aware "worst background"
   diff** (`Σ max(blackΔ², whiteΔ²)`) for semi-transparent pixels.
7. **Edge-modulated serpentine Floyd–Steinberg** at remap.

## Why it won at N=256 (pre-fix), decomposed

At N≤64 the contest is the coarse partition — everyone is similar. At N=256 the
contest is **where to spend the marginal colors** and **matching the metric's
perceptual geometry**. libimagequant's edge there is three things:

| Their trick | What it buys at 256 | Our status |
|---|---|---|
| 0.57 perceptual space | spacing that CIEDE2000 rewards | **Neutralized** — our OKLab-matched assignment (report 02) is *more* perceptual and now beats them |
| importance/contrast map | spends marginal colors on edges, not flat gradients | **Not ported** — but see the objective caveat below |
| error-weighted refinement | refines the few hard cells | partially (our k-means refine; no explicit error weight yet) |

So the OKLab-matched fix (report 02) already explains and reverses the gap. The
importance map is the remaining distinctive lever — but it is **not** an unambiguous
win for *our* number:

## The objective caveat (a real research point)

Our benchmark is **unweighted mean CIEDE2000** over all pixels. libimagequant's
importance map deliberately **down-weights flat regions** — which contain *most*
pixels. Spending fewer palette entries on the many flat pixels to serve the few edge
pixels will, if anything, **raise** unweighted mean ΔE2000 while improving
**perceived** quality (edges look crisp, banding hides where the eye doesn't look).

So importance weighting and edge-aware dithering optimize a *different objective*
than we currently score. Two consequences:

1. **For the agreed metric (unweighted mean ΔE2000), we have already beaten the bar**
   at every N via OKLab-matched refine (report 02). Chasing the importance map for
   *this* number would likely backfire.
2. **For product quality (what the user sees)**, importance weighting + edge-aware
   dither are real wins — but they need a **perceptual benchmark to measure**
   (edge-weighted ΔE, SSIM/MS-SSIM, or a saliency-weighted error), not the flat mean.
   That is a deliberate, separate research track, not a free addition.

This is the kind of result the fan-out is for: a trick the incumbent uses, evaluated
and **scoped out of the current objective with a reason**, not blindly copied.

## Porting verdict (ranked)

- **Already done / better:** perceptual space → OKLab-matched assignment beats the
  0.57-gamma approximation (report 02).
- **Worth porting for the *current* metric:** error-weighted refinement (#5) — pulls
  on hard cells without changing the objective; cheap. Median-cut/Wu seeding (#4) we
  already do (PCA-divisive init).
- **Defer to a perceptual-objective track (needs a new metric first):** the
  importance/contrast map (#3) and edge-aware dithering (#7). High *perceived* value,
  but they trade against unweighted mean ΔE2000.
- **Only if we add transparency:** alpha-aware worst-background diff (#6).

## Peer check

Material Color Utilities = Wu + Wsmeans (k-means in **CIELAB**, Wu-seeded, ≤10 iters,
triangle-inequality speedup) — confirms "perceptual-space + good seeding + refine" is
the modern recipe, which is exactly our OKLab-matched refine. MCU has **no** importance
map or alpha handling (it's for theme extraction). The agent found **no published
2018–2025 method with a verified CIEDE2000 win over libimagequant at N=256** — which
makes our OKLab-matched result worth validating at scale (CQ100) before claiming it.

## Next experiments queued

1. **Error-weighted refinement** — port #5, measure ΔE2000 delta (objective-safe).
2. **CQ100 scale-up** — confirm the OKLab-matched win holds on 100 images, not 6.
3. **Perceptual-objective track** (separate) — add an edge-weighted/SSIM metric, then
   evaluate the importance map + edge-aware dither honestly under it.
4. **Interdisciplinary round** — deterministic annealing / neural-gas / PNN-merge from
   the second brainstorm agent (pending), to push unweighted mean further.

Sources: libimagequant `src/{pal,hist,image,mediancut,kmeans,quant,nearest,remap}.rs`
(github.com/ImageOptim/libimagequant); https://pngquant.org/lib/ ; Material Color
Utilities (github.com/material-foundation/material-color-utilities); Celebi 2011
(arXiv:1101.0395).
