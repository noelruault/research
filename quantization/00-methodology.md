# 00 — Methodology: the puzzle strategy for `quantize`

This record builds pixelize's palette-derivation feature (workflow B, "turn any
image into N colors / merge similar colors") the same way
[nearest-color-scaling](../nearest-color-scaling/) built the matcher: **decompose
the pipeline into independent computational pieces, enumerate many ways to do each
piece (the popular ones AND ideas transferred from other disciplines), benchmark
every piece in isolation, then stack the winning pieces into the full solution and
benchmark the integration.** Every headline number traces to a reproducible
benchmark whose raw output sits in the matching `*-data.txt`.

The goal is not "a quantizer." It is **the most performant quantizer we can
assemble from the best piece at every step**, proven against the incumbents
(pngquant/libimagequant, ImageMagick, GIMP, Aseprite) on a public dataset.

## The pipeline, decomposed into pieces

"Reduce an image to N colors" is a chain of independent decisions. Each is a
*puzzle piece* with multiple candidate implementations we can swap and measure:

```
 image
   │
   ▼  [P1] COLOR SPACE        in which space do we measure "close"?
   │       RGB · linear-RGB · CIELAB · OKLab · YCbCr
   ▼  [P2] HISTOGRAM          how do we summarize the pixels?
   │       exact unique-map · 5-bit grid · 6-bit grid · octree buckets
   ▼  [P3] PALETTE SELECTION  how do we pick the N representatives?
   │       median-cut · Wu · octree · k-means · PNN/agglomerative
   │       · Jenks/Ckmeans-1D-on-axis · maximin/farthest-point · PQ-inspired
   ▼  [P4] SEEDING            (if iterative) how do we initialize?
   │       random · k-means++ · maximin · Wu-seeded
   ▼  [P5] REFINEMENT         do we polish the palette?
   │       none · Lloyd k-means · Elkan/Hamerly accelerated · mini-batch
   ▼  [P6] ASSIGNMENT         map every pixel to the palette  ← pixelize ALREADY OWNS THIS
   │       exact kd-tree (shipped) · 6-bit LUT fast-mode (shipped)
   ▼  [P7] POST              optional: merge-by-threshold (PNN) · dither
   ▼
 Palette[struct{}]  →  feeds the existing Apply() pipeline unchanged
```

The crux (proved in [nearest-color-scaling](../nearest-color-scaling/) and restated
in [background/survey.md](background/survey.md)):
**P6 is the operation pixelize already does exactly and fastest.** It is k-means'
inner loop (P3/P5) *and* the final mapping pass of every divisive method. So this
investigation only has to win at P1–P5; P6 is a solved, shipped dependency we
reuse. That is also why this is a *package inside pixelize*, not a new project.

## Why decompose instead of "just port libimagequant"

Because the pieces are independent, the best published tool is rarely the best at
*every* piece. libimagequant fixes P1=perceptual-weighted-RGB, P3=modified-median,
P5=k-means; ImageMagick fixes P3=octree. By measuring pieces separately we can,
for example, take Wu's selection (P3), seed k-means with it (P4), refine in OKLab
(P1+P5) — a combination no single incumbent ships. The puzzle is the point.

## Benchmark protocol

**Two levels, always:**

1. **Piece-level** — hold every other piece fixed at a baseline and vary one
   piece; attribute the delta to that piece alone. (Mirrors how report 09 isolated
   kd-tree variants.)
2. **Integration-level** — stack chosen pieces and measure the full pipeline,
   watching for interactions (e.g. a histogram precision that's fine alone but
   costs ΔE once k-means refines on it).

**Corpus.** Phase 1 uses the six in-repo paintings
(`docs/demo/inputs/{athens,liberty,monet,napoleon,pearl,starry}.jpg`) — the same
set the matcher benchmarks used, so numbers are comparable across records. Phase 2
(competition shootout) adds **CQ100** (100 images + 8,400 precomputed reference
quantizations with published per-image MSE) so we compare at N∈{4,16,64,256}
without re-running every tool. See [background/design.md](background/design.md).

**Color counts.** N ∈ {4, 16, 64, 256}. No dithering for palette-quality runs
(dither confounds intrinsic selection error); dither is measured separately in P7.

**Metrics (per image, then mean + 95th percentile across the corpus):**

- **RGB MSE / PSNR** — what the incumbents optimize; comparable to CQ100's
  reference numbers. The sanity bar: be at least competitive here.
- **Mean & p95 ΔE2000** (CIEDE2000 over CIELAB) — the *headline*. RGB-MSE rewards
  perceptually poor results (uniform desaturation can lower it); ΔE2000 tracks
  human-judged fidelity. This is where a perceptual P1 wins even at equal MSE.
- **Wall + CPU time**, and **determinism** (byte-identical palette across runs —
  required for golden tests and reproducible build maps).

**Discipline (from the matcher record):**
- Each report `NN-*.md` has a `NN-*-data.txt` with the verbatim runner output.
- The runner prints the machine, core count, Go version, date, and the exact
  piece configuration, so a row can be re-derived.
- A claim ships only with a measured delta behind it; "should be faster" is a
  hypothesis to test, never a conclusion.

## The harness

`bench/` is a **self-contained Go module** — it does NOT import or modify pixelize.
Experimental pieces live here and are benchmarked here; only a *chosen* piece gets
promoted into pixelize's `quantize` package (with its own tests), exactly as the
matcher research stayed out of the shipped binary. It measures palette quality
with an exact nearest-color assignment (stdlib `color.Palette.Index`, which is
bit-identical to pixelize's shipped kd-tree for opaque colors), so harness numbers
transfer directly to the engine.

`metrics.go` implements sRGB→linear→XYZ(D65)→CIELAB and CIEDE2000, self-tested
against the Sharma et al. reference pairs so the perceptual numbers are
trustworthy.

## Report plan

| # | Piece(s) | Question |
|---|---|---|
| 00 | — | this methodology |
| 01 | cross-disciplinary | what to borrow from VQ, ANN/PQ, cartography (Jenks), CVT, OKLab, … |
| 02 | P1 color space | RGB vs linear vs Lab vs OKLab — which space to select/measure in |
| 03 | P2 histogram | precision vs quality/speed |
| 04 | P3 baselines | median-cut, Wu, octree — establish the floor and the incumbents' level |
| 05 | P3 exotic | Jenks/Ckmeans-1D, maximin, PNN, PQ-inspired — the out-of-the-box pieces |
| 06 | P4 seeding | random vs k-means++ vs maximin vs Wu-seeded |
| 07 | P5 refinement | Lloyd vs Elkan/Hamerly vs mini-batch — quality per millisecond |
| 08 | P6 assignment | reuse pixelize kd-tree; Lab-space assignment cost |
| 09 | integration | stack the winners; the performant solution |
| 10 | vs competition | CQ100 shootout vs pngquant/ImageMagick/GIMP/Aseprite |

Reports 01, 04, 05 are in flight (cross-disciplinary agent + baseline harness).
This file and the harness skeleton come first so every later number has a home.
