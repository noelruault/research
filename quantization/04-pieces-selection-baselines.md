# 04 — Piece P3: selection baselines

Establishes the floor for palette selection: the trivial **popularity** baseline
and classic **median cut**. Every cross-disciplinary piece from
[01](01-cross-disciplinary-transfer.md) (Wu, maximin+Lloyd, PCA+Ckmeans, OKLab,
HyAB) is measured as a delta against median cut here. Raw output:
[04-pieces-selection-baselines-data.txt](04-pieces-selection-baselines-data.txt).

## Setup

- Corpus: the six in-repo paintings
  (`pixelize/docs/demo/inputs/{athens,liberty,monet,napoleon,pearl,starry}.jpg`).
- Assignment: exact nearest-color (`color.Palette.Index`, bit-identical to
  pixelize's shipped kd-tree for opaque colors), so numbers transfer to the engine.
- Metrics: RGB MSE (↓), PSNR dB (↑), mean & p95 CIEDE2000 (↓). CIEDE2000 validated
  against the Sharma reference pairs (`metrics_test.go`).
- No dithering (palette-quality isolation).

## Results (mean over 6 images)

| N | quantizer | MSE | PSNR | mean ΔE2000 | p95 ΔE2000 |
|---|---|---|---|---|---|
| 8 | popularity | 6147.63 | 12.94 | 21.972 | 52.779 |
| 8 | **median-cut** | **189.46** | **25.68** | **6.189** | **14.093** |
| 16 | popularity | 3487.17 | 14.49 | 16.989 | 44.344 |
| 16 | **median-cut** | **98.62** | **28.43** | **4.857** | **11.197** |
| 32 | popularity | 2919.54 | 15.72 | 15.447 | 42.982 |
| 32 | **median-cut** | **53.15** | **31.09** | **3.925** | **9.098** |

## Reading

- **Popularity is a non-starter** — 30–60× the MSE of median cut, because keeping the
  N most-frequent colors ignores everything in the tail (skies, gradients), so large
  image regions map to a wildly wrong color (athens/starry are the worst). It is here
  only as the floor.
- **Median cut is the real baseline.** mean ΔE2000 ≈ 4.9 at N=16. A ΔE2000 of ~2.3 is
  roughly "just noticeable", so at N=16 the average pixel is a few JNDs off — exactly
  the headroom the perceptual pieces target.
- **The target.** Reports 05–09 must drive that 4.857 down (and the p95 of 11.2,
  which captures the worst regions) without losing determinism or speed. libimagequant
  (median-cut init + k-means refine) is the external bar; pieces #1/#2 from
  [01](01-cross-disciplinary-transfer.md) are our routes to it and past it.

## Next

- Add **Wu** (variance-min cut) — expected to beat median cut at equal cost; the
  honest "classic best non-iterative" baseline.
- Add **maximin-seeded k-means** and **PCA + Ckmeans.1d.dp** as the first
  cross-disciplinary pieces, each as a `Quantizer`, measured as a delta against this
  table.
- Add an **OKLab** color-space toggle to selection and re-measure (evaluation stays
  CIEDE2000).
