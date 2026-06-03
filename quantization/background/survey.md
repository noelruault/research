# 02 — Color-quantization survey

Prior art for "derive a palette from an image" (workflow B), and where pixelize
can credibly beat the incumbents. Two sub-problems recur throughout; keep them
separate:

- **(A) Palette selection** — choose N representative colors.
- **(B) Pixel assignment** — map every pixel to its nearest palette entry.

(B) is the operation pixelize already does exactly and fast (the kd-tree matcher
of the [nearest-color-scaling](../nearest-color-scaling/) record). The thesis of
this whole effort: **every quantizer needs (B)**, so B is the shared core and the
new work is only A.

## 1. The algorithm families

| Method | Palette selection (A) | Speed | Quality | Deterministic | Needs a final (B) pass? |
|---|---|---|---|---|---|
| **Median cut** (Heckbert 1982) | divisive; split box on longest axis at the population median | fast | good; wastes entries on flat regions | yes | yes |
| **Wu** (Graphics Gems II, 1991) | divisive; split at the **variance-minimizing** cut using O(1) cumulative moments | very fast | very good (optimizes SSE locally) | **yes** | yes |
| **Octree** (Gervautz–Purgathofer 1988) | hierarchical; insert by RGB bits, merge least-populous leaves | very fast, streaming, bounded memory | fair (merges by population, not error) | yes | yes ("Assignment" stage) |
| **k-means / Lloyd** | iterative: assign to nearest centroid, move centroid to cluster mean | slow unless accelerated | **best** (directly minimizes SSE) | **no** (random init) | built into every iteration |
| **Agglomerative / PNN** | bottom-up: merge the two closest clusters until N (or until no pair < threshold T) | slow naive (O(N²)–O(N³)) | high with a distortion-cost merge | yes | yes |

Notes that matter for our design:

- **Wu is the deterministic sweet spot**: near-linear in pixels, no randomness,
  quality clearly above classic median cut. Good *default*.
- **k-means is the quality ceiling** but is non-deterministic under random init;
  fix with **k-means++** or a deterministic init, and run it on the histogram
  rather than raw pixels.
- **"Merge colors closer than T"** is precisely **single-linkage agglomerative
  clustering with the dendrogram cut at height T**. The quality-oriented variant
  (PNN) merges by *minimum distortion increase* rather than raw distance, avoiding
  single-linkage "chaining". This is the literal implementation of the user's
  "merge similar colors" idea — and it works on a *derived* or a *loaded* palette.

## 2. The two technical hinges (verified)

**Hinge 1 — k-means' assignment step *is* a nearest-color query.** In Lloyd's
algorithm each color is assigned to the nearest centroid by Euclidean distance;
the current centroid set is just a candidate palette. So pixelize's exact
nearest-color routine *is* k-means' inner loop. The assignment step dominates
cost (O(n·K) per iteration), so accelerating nearest-centroid lookup — a kd-tree
over the K centroids, rebuilt per iteration, answering each query in ~O(log K) —
speeds the whole algorithm. Colour is only 3-D, so kd-trees are ideal here and
escape the high-dimensional curse. (Celebi, *Improving the Performance of K-Means
for Color Quantization*, arXiv:1101.0395; Kanungo et al., *An Efficient k-Means
Clustering Algorithm*, PAMI 2002.)

**Hinge 2 — the divisive methods still need a final (B) pass.** Median cut, Wu,
and octree produce a palette but must then map each pixel to it. The box *mean* is
not guaranteed to be the nearest palette entry for a color near a box boundary, so
a true nearest-color pass (an inverse colormap / kd-tree) is standard and is what
guarantees minimal mapping error. ImageMagick names this stage **Assignment**
explicitly. (Leptonica color-quantization page; ImageMagick `quantize` docs.)

Conclusion: **pixelize's kd-tree matcher is the workhorse for all of A's
strategies** — k-means inner loop, and the final mapping pass for Wu/median/octree.
This is the architectural reason B is a *package inside pixelize*, not a new repo.

## 3. What the production tools actually use

- **libimagequant / pngquant** — *modified median cut* (split to minimize variance
  from the median, with perceptual weighting) **+ k-means (Voronoi-iteration)
  refinement**. The quality bar to beat. Its internal error is gamma-corrected,
  alpha-aware, importance-weighted. (https://pngquant.org/lib/)
- **ImageMagick `-colors`** — **octree** ("adaptive spatial subdivision":
  Classification → Reduction → Assignment), optional Floyd–Steinberg. Widely
  criticized for posterizing and no alpha. (https://imagemagick.org/script/quantize.php)
- **GIMP "convert to indexed → generate optimum palette"** — median-cut-based.
  *(Primary GIMP source not loaded — secondary sources only; flagged.)*
- **Aseprite "create palette from sprite" / RGB→Indexed** — **octree** internally
  (`rgbmap="octree"`), with a `fitCriteria` selectable distance space
  (`linearizedRGB`, `cielab`) and proposals to add more. A script can drive this
  directly via `app.command.ColorQuantization{ algorithm="rgb5a3" }` and
  `app.command.ChangePixelFormat{ format="indexed", rgbmap="octree",
  fitCriteria="cielab" }` — *no Lua reimplementation needed in-app*. *(Exact
  default + option names vary by version and were not fully source-verified;
  confirm against the target version. Sources:
  [aseprite/aseprite#3394](https://github.com/aseprite/aseprite/issues/3394),
  [ChangePixelFormat API](https://www.aseprite.org/api/command/ChangePixelFormat),
  the official `tests/scripts/color_quantization.lua`.)*

The headline gap: the incumbents optimize **RGB MSE**. None of the common ones
optimize a **perceptual** objective by default. That is our opening (see
[03](design.md) and the benchmark).

## 4. Astropulse K-Centroid Downscale — the closest neighbor (MIT, public)

Source: [github.com/Astropulse/K-Centroid-Aseprite](https://github.com/Astropulse/K-Centroid-Aseprite),
**MIT © 2023 Astropulse** (license re-verified directly). Distributed paid/PWYW on
itch.io + Gumroad; the same code is mirrored free on GitHub.

Algorithm (`scripts/scaler.lua`): divide the source into a `targetW × targetH`
grid of tiles; run **k-means per tile** (random seed locked for determinism, fixed
iteration count, squared-Euclidean RGB distance); output each tile's **largest
centroid** (most-common color). Dialog (`extension.lua`): *Centroids* 2–16
(default 2), *Iterations* 1–20 (default 3), output size, lock-ratio.

Two takeaways:

1. **It validates the thesis.** Its inner loop is exactly the nearest-color
   assignment pixelize already does — and the author notes it is *"slow, several
   seconds per image"* precisely because it runs per-pixel in Lua. A binary-backed
   Go path with a kd-tree closes that gap.
2. **It is a downscaler, not a palette reducer.** It picks a representative color
   *per output pixel*; it does not produce a fixed N-color palette, a build map, or
   a parts list. Our workflow B (derive a global N-color palette) and pixelize's
   mosaic outputs remain unserved by it.

## 5. Prior art to borrow from

**Lua (if we ever want an in-process fallback):**
- [PG1003/bitmap](https://github.com/PG1003/bitmap) `src/bitmap/color.lua` — clean,
  dependency-free pure-Lua median cut.
- [FeelTheFonk/SDDj](https://github.com/FeelTheFonk/SDDj) — an Aseprite extension
  whose dialog already offers `kmeans / median_cut / octree / octree_lab`; the best
  template for a palette-derivation UI.
- [Reference Color Extractor](https://community.aseprite.org/t/script-reference-color-extractor-frequency-diversity-k-means/28258)
  — Frequency / Diversity / K-Means palette extraction from a reference image.

**Go (our actual target):**
- Stdlib gives **interfaces, not a quantizer**: `draw.Quantizer` (no concrete impl
  in stdlib), `color.Palette` (`Convert`/`Index` = nearest-color), and only the
  *fixed* `palette.WebSafe` / `palette.Plan9`. `image/gif` takes a
  `draw.Quantizer` and falls back to static Plan9 if nil — i.e. it *expects* you to
  bring an adaptive one. (https://pkg.go.dev/image/draw, /image/color/palette, /image/gif)
- [ericpauley/go-quantize](https://github.com/ericpauley/go-quantize) — fast
  weighted median cut, **implements `draw.Quantizer`**, ~20 ms vs hundreds of ms
  for alternatives in its own benchmark. The model for an efficient Go median cut.
- [soniakeys/quant](https://pkg.go.dev/github.com/soniakeys/quant) — readable
  `median` and `mean` quantizers, both `draw.Quantizer`.
- [libimagequant](https://github.com/ImageOptim/libimagequant) (C) — the quality
  reference (median-cut init + k-means refinement + perceptual weighting).

## Sources

- Survey: https://link.springer.com/article/10.1007/s10462-023-10406-6
- Celebi, k-means for CQ: https://arxiv.org/pdf/1101.0395
- Kanungo et al., efficient k-means: https://www.cs.umd.edu/~mount/Projects/KMeans/pami02.pdf
- Leptonica: http://www.leptonica.org/color-quantization.html
- ImageMagick quantize: https://imagemagick.org/script/quantize.php
- pngquant/libimagequant: https://pngquant.org/lib/
- Wu's quantizer: https://gist.github.com/bert/1192520
- Octree (Dr. Dobb's): http://collaboration.cmc.ec.gc.ca/science/rpn/biblio/ddj/Website/articles/DDJ/1996/9601/9601f/9601f.htm
- Single-linkage clustering: https://en.wikipedia.org/wiki/Single-linkage_clustering
- K-Centroid (MIT): https://github.com/Astropulse/K-Centroid-Aseprite
- go-quantize: https://github.com/ericpauley/go-quantize
- soniakeys/quant: https://pkg.go.dev/github.com/soniakeys/quant
- Aseprite quantization: https://github.com/aseprite/aseprite/issues/3394 · https://www.aseprite.org/api/command/ChangePixelFormat
