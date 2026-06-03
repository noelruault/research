# research

Durable research records — the *evidence trails* behind shipped work, kept
separate from the application repos so the conclusions can be re-derived. Each
record is a numbered series of reports where every headline number traces to a
reproducible benchmark whose raw output sits in a matching `*-data.txt`, plus
the harness that produced it. Nothing here is imported or run by a product
binary; it is documentation and data.

## Records

- **[nearest-color-scaling/](nearest-color-scaling/)** — how
  [pixelize](https://github.com/noelruault/pixelize) maps every pixel to its
  nearest palette color as fast as possible without giving up correctness. The
  exact kd-tree branch-and-bound, the parallel scan, the run-length collapse,
  and the 6-bit fast-mode LUT that ship in pixelize, plus the reverse-
  engineering proving pixelize is both more correct *and* faster than
  ImageMagick's approximate remap.

- **[quantization/](quantization/)** — deriving a palette *from* an image
  ("turn any image into N colors", workflow B), built the puzzle way: the
  pipeline decomposed into pieces (color space, histogram, selection, seeding,
  refinement), every piece measured in isolation, winners stacked. Ships as
  pixelize's [`quantize`](https://github.com/noelruault/pixelize/tree/main/quantize)
  package. **Result:** beats ImageMagick's octree at every palette size and
  matches/edges libimagequant (pngquant) on CIEDE2000, validated on the Kodak
  suite. Documents the wins (OKLab-matched assignment; a space-filling-curve
  initializer at large N) *and* the many measured discards (PNN, multi-restart,
  HyAB, deterministic annealing, MST/Friends-of-Friends, …).

- **[compression-agent/](compression-agent/)** — a measurement-driven subagent
  that picks the right HTTP compression for a stack by benchmarking, not opinion.

The Aseprite *extension* is a build, not research — its planning (extension-quality
notes, the reverse-engineered UI catalogue) lives with the code in
[pixelize-aseprite/.plans/](https://github.com/noelruault/pixelize-aseprite/tree/main/.plans),
not here.

## Method

The shared method across records is a **fan-out + judge** loop: enumerate many
candidate approaches (the popular ones *and* transfers from other disciplines),
implement each as a benchmarkable piece, measure it against a fixed baseline on
a fixed corpus with a trustworthy metric, and **keep or discard with a number**.
Discards are first-class results — recorded with their measured reason so they
are not relitigated. Headline claims ship only with a measured delta behind them.

## Why it lives here

A raw research record — including notes on third-party source under its own
license — is documentation, not something a binary imports. The application
repos reference these records (pinned to a commit so the cited evidence stays
put) and keep only the *planning* that drives the build (e.g.
`pixelize/.plans/`). This repo is the permanent home for the evidence.
