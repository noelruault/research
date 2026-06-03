# 03 — `quantize` package design, CLI, and benchmark

The actionable output of this record: how to add palette derivation (workflow B,
"merge similar colors") to pixelize, the CLI surface, and the benchmark that
proves it beats the incumbents. Nothing here is built yet — this is the plan an
execution doc in pixelize's `.plans/` would drive.

## 1. Design principle: B is a layer, not a rewrite

pixelize already owns **palette assignment** — the exact kd-tree nearest-color
match (`kdtree.go`) and the pluggable `DistanceFunc` (`distance.go`, including a
perceptual CIEDE2000 option). Every quantizer needs that primitive (see
[02 §2](survey.md)). So the new package supplies only **palette
selection** and reuses the rest:

```
pixelize/
  distance.go        # DistanceFunc (Euclidean default, CIEDE2000 option) — REUSED
  kdtree.go          # exact nearest-color — REUSED as k-means inner loop + final map
  palette.go         # Palette type, Apply() — gains an Auto/Merge entry path
  quantize/
    quantize.go      # Quantizer interface + Generate(); implements draw.Quantizer
    histogram.go     # shared 3D RGB histogram (5-bit/chan grid) + weights
    wu.go            # Wu variance-min cut         (DEFAULT: deterministic, fast)
    median.go        # Heckbert median cut         (classic baseline)
    octree.go        # octree                      (streaming / parity with Aseprite)
    kmeans.go        # Lloyd + k-means++ init      (quality ceiling; reuses kdtree)
    merge.go         # agglomerative merge-by-T    ("merge similar colors")
```

The interface mirrors the stdlib so a derived palette also drops straight into
`image/gif` (pixelize already does GIFs):

```go
// Quantizer selects a palette of at most n colors from an image.
// It composes with pixelize's existing nearest-color matcher for the
// assignment pass, so callers get exact mapping for free.
type Quantizer interface {
    // Generate returns up to n representative colors for img.
    // dist is the metric used for selection and (for k-means) assignment;
    // pass pixelize's CIEDE2000 func for perceptual selection.
    Generate(img image.Image, n int, dist pixelize.DistanceFunc) pixelize.Palette
}

// Each concrete quantizer also satisfies stdlib draw.Quantizer:
//   func (q Wu) Quantize(p color.Palette, m image.Image) color.Palette
// so `gif.Options{ Quantizer: quantize.Wu{} }` just works.
```

k-means then reads as the thesis made literal — assignment is *the matcher we
already ship*:

```go
func (k KMeans) Generate(img image.Image, n int, dist pixelize.DistanceFunc) pixelize.Palette {
    hist := histogram.Build(img)            // weighted unique colors
    cents := kmeanspp(hist, n, dist)        // deterministic-seeded init
    for i := 0; i < k.Iters; i++ {
        tree := pixelize.NewKDTree(cents)   // REUSE: nearest-color over centroids
        sums := make([]accum, len(cents))
        for _, c := range hist {            // assignment == nearest-color query
            j := tree.Nearest(c.Color)      //   (the exact op pixelize is fast at)
            sums[j].add(c.Color, c.Weight)
        }
        cents = recenter(sums, cents)       // the only step k-means adds over B
    }
    return pixelize.Palette(cents)
}
```

`merge.go` ("merge similar colors") is agglomerative PNN that operates on *any*
palette — derived here or loaded from a file:

```go
// Merge collapses palette entries whose pairwise distance < threshold,
// merging by minimum distortion increase (PNN), until no pair is closer
// than threshold. Deterministic. Reuses dist for all comparisons.
func Merge(p pixelize.Palette, threshold float64, dist pixelize.DistanceFunc) pixelize.Palette
```

## 2. CLI surface

Additive to today's flags; no breaking changes. `-palette` gains two pseudo-values:

| Flag | Meaning |
|---|---|
| `-palette auto:N` | Derive an N-color palette from the image (e.g. `auto:16`). `auto` alone = a sensible default (16). |
| `-quantize ALGO` | Selection algorithm: `wu` (default) \| `kmeans` \| `median` \| `octree`. Only meaningful with `-palette auto:*`. |
| `-kmeans-iter K` | Iterations for `-quantize kmeans` (default ~10). |
| `-merge DIST` | Post-pass: merge palette colors closer than `DIST` in the active metric. Works on derived *and* loaded palettes. |
| `-distance METRIC` | Already implied by `DistanceFunc`: `euclidean` (default) \| `ciede2000` (perceptual). Drives both selection and assignment. |

Examples:

```sh
# Any photo → clean 16-color pixel art, palette derived from the image:
pixelize photo.jpg -size 96x96 -palette auto:16 -o art.png

# Highest quality, perceptual selection:
pixelize photo.jpg -palette auto:24 -quantize kmeans -distance ciede2000 -o art.png

# Load a big palette, then merge near-duplicates so the parts list is buildable:
pixelize logo.png -palette lego -merge 8 -pieces parts.csv -o mosaic.png
```

The whole feature reuses the existing `Apply` pipeline: derive/merge produces a
`Palette`, and everything downstream (dither, build-map, pieces, GIF, preview,
batch, watch) works unchanged. The Aseprite extension inherits all of it through
the binary — a new "Palette: Auto (N colors)" option in the dialog, no Lua algebra.

## 3. Default and rationale

- **Default `-quantize wu`** — deterministic (reproducible build maps and golden
  tests), near-linear, quality clearly above classic median cut, no init tuning.
- **`kmeans` for "best"** — seeded by Wu or k-means++ for determinism, refined with
  pixelize's kd-tree assignment. This is the libimagequant recipe (divisive init +
  k-means refine) and the route to beating RGB-MSE incumbents on ΔE.
- **`median`/`octree`** — baselines for the benchmark and parity with what
  ImageMagick/Aseprite do, so comparisons are apples-to-apples.

## 4. Benchmark — proving "better than anyone else"

Mirror pixelize's existing `bench/compare.sh` (the ImageMagick head-to-head), but
for *palette generation* quality. The design is reproducible because the dataset
ships reference outputs:

- **Dataset: [CQ100](https://data.mendeley.com/datasets/vw5ys9hfxw/3)** — 100
  permissively-licensed RGB images, and crucially **8,400 precomputed reference
  quantizations** (21 algorithms × 100 images × N∈{4,16,64,256}) with published
  per-image MSE. We can compare at the same N without re-running every tool.
  Optionally add the classic Kodak set + Lenna/Baboon/Peppers for continuity.
- **Color counts:** 4, 16, 64, 256. **No dithering** for the palette-quality
  comparison (dithering confounds intrinsic selection error).
- **Metrics, per-image and aggregated (mean + distribution):**
  1. **RGB MSE / PSNR** — comparable to CQ100's reference numbers and what the
     incumbents optimize. Sanity bar: we must be *at least competitive* here.
  2. **Mean & 95th-percentile ΔE2000** (CIELAB) — the *headline*. This is where a
     `-distance ciede2000` selection can win even at equal RGB-MSE, because RGB-MSE
     rewards perceptually-poor results (e.g. uniform desaturation) while ΔE2000
     tracks human-judged color fidelity.
  3. **SSIM / MS-SSIM** — structural sanity check.
- **Compare against (pinned versions, scripted, fixed seeds):**
  pngquant/libimagequant (the quality bar), ImageMagick `-colors N +dither`
  (octree), GIMP Script-Fu convert-indexed (median cut), and Aseprite CLI
  (`-b --color-mode indexed`) for the pixel-art-relevant comparison.
- **To claim a win credibly:** lower **mean ΔE2000** at each fixed N across the
  full CQ100 set, report the **per-image win rate**, and publish the harness +
  exact tool versions/flags. Optimizing selection under a ΔE2000 objective is the
  most defensible path to beating RGB-MSE-minimizing incumbents on perceptual
  fidelity — and it reuses the CIEDE2000 `DistanceFunc` pixelize already has.

Why this is winnable: the incumbents minimize RGB MSE; **none of the common ones
selects under a perceptual objective by default.** pixelize already carries an
exact CIEDE2000 matcher. Wiring it into selection (not just assignment) is the
differentiator — measured, not asserted.

## 5. Open questions for the execution plan

1. **Histogram precision** — 5-bit/channel (32,768 bins) is the literature default;
   confirm it doesn't cost ΔE at N=256 on photographic input.
2. **k-means determinism** — Wu-seeded vs k-means++ with a fixed seed; golden tests
   demand byte-stable output, so pick one and pin it.
3. **Metric for `-merge`** — default threshold units (raw Euclidean vs ΔE2000);
   ΔE2000 is more intuitive ("merge colors within 2 JNDs") but slower.
4. **Alpha** — libimagequant is alpha-aware; decide whether B handles transparency
   or documents straight-alpha only for v1.
5. **Where selection runs** — on the resized image (fewer pixels, faster) or the
   original (more faithful palette)? Probably the resized target, matching how a
   user thinks about "this sprite's colors".

## Sources

- CQ100 dataset + paper: https://data.mendeley.com/datasets/vw5ys9hfxw/3 ·
  https://www.spiedigitallibrary.org/journals/journal-of-electronic-imaging/volume-32/issue-3/033019/cq100--a-high-quality-image-dataset-for-color-quantization/10.1117/1.JEI.32.3.033019.full
- CIEDE2000 standard: https://cie.co.at/publications/colorimetry-part-6-ciede2000-colour-difference-formula
- Lab-vs-RGB selection caveat: https://30fps.net/pages/median-cut-lab-problem/
- libimagequant (median-cut + k-means refine): https://pngquant.org/lib/
- Go stdlib quantizer interfaces: https://pkg.go.dev/image/draw · https://pkg.go.dev/image/gif
- go-quantize (fast Go median cut): https://github.com/ericpauley/go-quantize
