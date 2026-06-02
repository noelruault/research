# 01 — Cross-disciplinary transfer

Where the pattern "reduce N points to K representatives, then assign each to its
nearest representative" appears in other fields, and which of those approaches are
worth transferring to color quantization (CQ). Sources are cited inline; claims
that rest on search snippets rather than fetched primary text are flagged
`[snippet]` (the agent's WebFetch was 403-blocked environment-wide, so primary-PDF
mechanism details should be re-verified before publishing specific numbers).

## The unifying insight

**Color quantization *is* 3-D vector quantization.** Lloyd's algorithm (signal
processing) = LBG (Linde–Buzo–Gray, compression) = k-means (statistics) =
centroidal Voronoi tessellation (computational geometry) are the same algorithm
rediscovered in four fields. In every one, the inner "assign each point to its
nearest representative" step is exactly the operation pixelize already ships as an
exact, fast kd-tree matcher. So most of what follows **reuses our existing
primitive** and only adds new ways to *choose* and *refine* the representatives.

(VQ: [py-lbg](https://github.com/internaut/py-lbg), [IJCSIT review](https://www.ijcsit.com/docs/Volume%202/vol2issue6/ijcsit2011020620.pdf);
CVT≡Lloyd: [Wikipedia CVT](https://en.wikipedia.org/wiki/Centroidal_Voronoi_tessellation),
[SIAM](https://epubs.siam.org/doi/10.1137/040617364).)

## Ranked shortlist (what to prototype)

Ordered by promise for our goal — best perceptual quality (low CIEDE2000) at high
speed, deterministically, reusing the kd-tree.

1. **Maximin (Gonzalez) seeding + frequency-weighted histogram + a few Lloyd
   passes.** The best-*evidenced* "k-means done right for CQ" recipe. Maximin is
   deterministic if seed 1 is fixed (e.g. the most frequent color), selects ~one
   center per natural cluster, and its inner loop is our nearest-color query.
   Coreset/histogram weighting cuts the point count 10–100×.
   ([Celebi, IMAVIS 2011 / arXiv:1101.0395](https://arxiv.org/pdf/1101.0395);
   [coreset CQ, IEEE SMC 2018](https://dl.acm.org/doi/10.1109/SMC.2018.00361))

2. **PCA-axis median cut with provably-optimal 1-D splits (`Ckmeans.1d.dp`).**
   Replace median cut's greedy "longest axis, cut at median" with: project a box's
   colors onto their principal axis, and split at the point a dynamic program
   proves optimal (Wang & Song's exact 1-D k-means; the modern, faster Jenks /
   Fisher natural-breaks). Deterministic, near-optimal, **non-iterative** — a real
   differentiator vs random-seeded k-means. Wu's classic quantizer is the adjacent
   production proof (variance moments + DP).
   ([Ckmeans.1d.dp, R Journal](https://digitalcommons.unl.edu/r-journal/285/);
   [Wu](https://gist.github.com/bert/1192520))

3. **Cluster/seed in OKLab, evaluate in CIEDE2000.** OKLab (Ottosson 2020) is cheap
   (matrix → cbrt → matrix), in CSS Color 4, with better hue linearity than CIELAB.
   Crucial caveat: naively switching median cut to **CIELAB does *not* reliably beat
   RGB** — axis-aligned splits interact badly with Lab's non-cubic gamut; OKLab's
   better cell shapes mitigate this but it is *slightly weaker than CIELAB for ΔE
   specifically*. So: cluster in OKLab for geometry/speed, but keep CIEDE2000 as the
   *evaluation* metric, never the inner loop.
   ([Ottosson](https://bottosson.github.io/posts/oklab/);
   [30fps: why CIELAB doesn't help median cut](https://30fps.net/pages/median-cut-lab-problem/);
   [Levien critique](https://raphlinus.github.io/color/2021/01/18/oklab-critique.html) `[snippet]`)

4. **HyAB as the clustering/centroid metric.** `ΔE_HyAB = |ΔL| + √(Δa²+Δb²)` —
   city-block lightness + Euclidean chroma — is purpose-built for the *large* color
   differences that dominate palette clustering (the regime CIEDE2000 was *not* fit
   for), and is far cheaper than CIEDE2000. **Integration friction (the main risk):**
   HyAB is non-Euclidean, so a vanilla Euclidean kd-tree won't return exact HyAB
   nearest neighbors. Options: a custom bound, or use HyAB only for centroid choice
   while assignment stays Euclidean-in-OKLab. Must be A/B'd.
   ([30fps: HyAB k-means](https://30fps.net/pages/hyab-kmeans/);
   [Abasi 2020](https://onlinelibrary.wiley.com/doi/abs/10.1002/col.22451) `[snippet]`)

5. **Weighted sort-means / Hamerly bounds for the Lloyd refinement.** Triangle-
   inequality accelerations that give *exact* k-means with **zero quality change**,
   deterministic. Color is 3-D with K≤256 — Hamerly's sweet spot. Weighted
   sort-means is published specifically for CQ histograms. For K=256 these may beat
   repeated kd-tree queries by reusing bounds across iterations (centers barely move
   late in convergence) — worth a head-to-head vs the kd-tree.
   ([Weighted Sort-Means, arXiv:1011.0093](https://arxiv.org/pdf/1011.0093);
   [Hamerly geometric methods](https://cs.baylor.edu/~hamerly/papers/sdm2016_rysavy_hamerly.pdf))

6. **(Stretch) Superpixel pre-pass / spatial color quantization.** SLIC superpixels
   give a spatially-aware coreset; Puzicha's scolorq jointly optimizes palette+dither
   for stunning low-K output. Both add cost; scolorq *abandons* per-pixel-exact
   assignment, so it can only be an optional separate mode, never the core.
   ([Superpixel CQ, PMC](https://pmc.ncbi.nlm.nih.gov/articles/PMC9416436/);
   [scolorq / rscolorq](https://github.com/okaneco/rscolorq))

## Confirmed dead ends

- **Product / Residual / Additive Quantization (PQ/RQ/AQ, FAISS).** Their entire
  reason to exist is *high* dimensionality (D=128…2048); color is D=3, where a flat
  256-entry codebook is searched exactly and trivially. The one transferable idea —
  OPQ's "rotate to decorrelate axes before splitting" — is obtained directly via PCA
  (piece #2). No evidence any CQ work fruitfully borrowed PQ.
  ([PQ explainer](https://www.pinecone.io/learn/series/faiss/product-quantization/))
- **Lattice VQ** — fixed codebook, gives up image adaptivity. Wrong tradeoff.
- **CAM16-UCS / ICtCp** — more accurate than OKLab but much costlier, no evidence of
  CQ benefit worth the speed.

## How this reshapes the piece list (P3–P5 in [00](00-methodology.md))

- **P3 selection** gains two strong non-trivial pieces beyond the median/Wu/octree
  baselines: **maximin-seeded k-means** (#1) and **PCA + Ckmeans.1d.dp median cut**
  (#2).
- **P1 color space** becomes a first-class variable: RGB vs OKLab for *selection*,
  with CIEDE2000 fixed as *evaluation* (#3); HyAB as an alternative selection metric
  with a known kd-tree caveat (#4).
- **P5 refinement** gets a concrete accelerator to benchmark against the kd-tree:
  weighted sort-means / Hamerly (#5).

The harness ([bench/](bench/)) already measures the median-cut baseline (report
[04](04-pieces-selection-baselines.md)); these pieces slot in as additional
`Quantizer` implementations and color-space options, each benchmarked in isolation
before stacking.
