# quantization

The research record behind **pixelize's [`quantize`](https://github.com/noelruault/pixelize/tree/main/quantize)
module** (shipped) — deriving a palette from an image ("turn any image into N
colors / merge similar colors", workflow B), built the puzzle way: decompose the
pipeline into pieces, enumerate many ways to do each (popular **and**
cross-disciplinary), benchmark each piece in isolation, then stack the winners and
benchmark the integration — proving it beats the incumbents on a public dataset.

Modeled on the sibling [nearest-color-scaling](../nearest-color-scaling/) record:
numbered reports, each headline number traceable to a reproducible benchmark whose
raw output sits in the matching `*-data.txt`.

## Contents

- **[00-methodology.md](00-methodology.md)** — the puzzle framework: the pipeline
  decomposed into pieces P1–P7, the two-level benchmark protocol, the report plan.
- **[01-cross-disciplinary-transfer.md](01-cross-disciplinary-transfer.md)** — what
  to borrow from vector quantization, ANN/PQ, cartography (Jenks/Ckmeans.1d.dp),
  CVT, OKLab, HyAB; the ranked shortlist of pieces to prototype; confirmed dead ends.
- **[02-pieces-color-space.md](02-pieces-color-space.md)** (+ data) — the matched-
  assignment breakthrough: clustering **and** assigning in OKLab beats pngquant at
  **every** N (the fair rematch report 05 demanded).
- **[04-pieces-selection-baselines.md](04-pieces-selection-baselines.md)** (+ data)
  — the popularity and median-cut floor every piece is measured against.
- **[05-pieces-fanout-judge.md](05-pieces-fanout-judge.md)** (+ data) — the 12-config
  selection fan-out (P3 × P1 × seeding) and the judged verdict: winners (init +
  k-means refine; PCA-divisive as the deterministic default) and discards (maximin
  under-performs; OKLab doesn't help under RGB assignment), plus the determinism
  finding (sort the histogram or seeded k-means drifts).
- **[06-competitive-teardown.md](06-competitive-teardown.md)** — source-level
  dissection of libimagequant (importance map, 0.57-gamma space, error-weighted
  refine) and which tricks are objective-safe to port vs which belong to a separate
  perceptual track.
- **[07-codebook-brainstorm.md](07-codebook-brainstorm.md)** — interdisciplinary
  fan-out (deterministic annealing, neural gas, ECVQ, SA, GMM, PNN): most ruled out
  at N=256 with reasons; PNN/error-reweighting kept.
- **[08-interdisciplinary-measured.md](08-interdisciplinary-measured.md)** (+ data) —
  the brainstorm's candidates *implemented and benchmarked*: PNN, multi-restart,
  error-weighted refine, HyAB — **all discarded with numbers** (none beats the
  OKLab-matched champion on unweighted mean ΔE2000).
- **[09-crossdomain-fanout.md](09-crossdomain-fanout.md)** (+ data) — far-afield
  transfers (crypto/databases, astrophysics, statistical mechanics): a **space-
  filling-curve (Morton) initializer wins at N=256** (beats champion 6/6 and pngquant
  by 5%); MST/Friends-of-Friends and deterministic annealing measured and discarded.
- **[11-kodak-validation.md](11-kodak-validation.md)** (+ data) — scale-up to the
  **Kodak-18** standard benchmark: beats ImageMagick comprehensively (18/18 at N≥16)
  and edges/matches pngquant at every N (CQ100 itself was unreachable — Mendeley off
  the allowlist).
- **[10-vs-competition.md](10-vs-competition.md)** (+ data) — the shootout: scored
  identically with CIEDE2000, our quality mode beats **ImageMagick** (octree) at
  every N, and — with OKLab-matched assignment (report 02) — beats **pngquant**
  (libimagequant) at **every** N. Harness: `bench/compare-quant.sh` + `emit`/`score`.
- **[bench/](bench/)** — the self-contained Go harness (imports nothing from
  pixelize): trustworthy metrics (MSE/PSNR + CIEDE2000 self-tested against Sharma),
  a `Quantizer` interface, and the pieces. `go test` checks the metric; `go run .`
  runs the corpus.

- **[background/](background/)** — the early scoping docs this record grew from: a
  broad prior-art `survey.md` and the initial `design.md` (the seed of the shipped
  package and the `.plans/quantize` execution plan).

(Report 03, histogram-precision, is the one planned-but-unwritten report; it did not
gate any decision.)

## Result, and where it shipped

The winning pipeline — **PCA-divisive init + weighted Lloyd refine, in a color space
chosen by palette size** (RGB for small palettes, OKLab for large; the measured
crossover is ~N=32–48) — shipped as pixelize's
[`quantize`](https://github.com/noelruault/pixelize/tree/main/quantize) package and
its `-palette auto:N` CLI flag. Output is deterministic (the histogram is sorted into
a canonical order). The space-filling-curve initializer is opt-in, hinted for N≥256.

**Measured quality** (mean CIEDE2000, scored identically, no dither): beats
ImageMagick's octree at every palette size (decisively — 18/18 Kodak images at N≥16)
and matches-or-edges libimagequant/pngquant at every size (the honest claim — a
genuine edge, not a rout; N=16 is roughly a tie). Validated on the **Kodak suite (18
images) + 6 paintings = 24 images**. True **CQ100** (100 images) was unreachable in
the build environment (Mendeley off the egress allowlist); the harness
(`bench/compare-quant.sh`, `emit`/`score`) takes any image directory, so CQ100 is a
corpus swap, not new code.

The execution plan that drove the build lives in `pixelize/.plans/quantize/` (repo
convention: research here, build plan there).
