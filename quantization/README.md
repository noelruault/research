# quantization

The research record behind **pixelize's planned `quantize` module** — deriving a
palette from an image ("turn any image into N colors / merge similar colors",
workflow B), built the puzzle way: decompose the pipeline into pieces, enumerate
many ways to do each (popular **and** cross-disciplinary), benchmark each piece in
isolation, then stack the winners and benchmark the integration — proving it beats
the incumbents on a public dataset.

Modeled on the sibling [nearest-color-scaling](../nearest-color-scaling/) record:
numbered reports, each headline number traceable to a reproducible benchmark whose
raw output sits in the matching `*-data.txt`.

## Contents

- **[00-methodology.md](00-methodology.md)** — the puzzle framework: the pipeline
  decomposed into pieces P1–P7, the two-level benchmark protocol, the report plan.
- **[01-cross-disciplinary-transfer.md](01-cross-disciplinary-transfer.md)** — what
  to borrow from vector quantization, ANN/PQ, cartography (Jenks/Ckmeans.1d.dp),
  CVT, OKLab, HyAB; the ranked shortlist of pieces to prototype; confirmed dead ends.
- **[04-pieces-selection-baselines.md](04-pieces-selection-baselines.md)** (+ data)
  — the popularity and median-cut floor every piece is measured against.
- **[05-pieces-fanout-judge.md](05-pieces-fanout-judge.md)** (+ data) — the 12-config
  selection fan-out (P3 × P1 × seeding) and the judged verdict: winners (init +
  k-means refine; PCA-divisive as the deterministic default) and discards (maximin
  under-performs; OKLab doesn't help under RGB assignment), plus the determinism
  finding (sort the histogram or seeded k-means drifts).
- **[10-vs-competition.md](10-vs-competition.md)** (+ data) — the shootout: our
  quality mode beats **pngquant** (libimagequant) at N≤64 and **ImageMagick**
  (octree) at every N, scored identically with CIEDE2000; pngquant regains N=256.
  Harness: `bench/compare-quant.sh` + `emit`/`score` modes.
- **[bench/](bench/)** — the self-contained Go harness (imports nothing from
  pixelize): trustworthy metrics (MSE/PSNR + CIEDE2000 self-tested against Sharma),
  a `Quantizer` interface, and the pieces. `go test` checks the metric; `go run .`
  runs the corpus.

Reports 02, 03, 05–10 are planned in [00](00-methodology.md) and filled as pieces
are implemented and measured.

## Status

Methodology + harness stand up; CIEDE2000 validated; median-cut baseline measured
(mean ΔE2000 ≈ 4.86 at N=16 over six paintings). The execution plan that drives the
build into pixelize lives in `pixelize/.plans/` (per the repo convention — research
here, build plan there).
