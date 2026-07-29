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

- **[shapes-image-file-format/](shapes-image-file-format/)** — can an image
  format made of *shapes* rather than pixels beat WebP, PNG, JPEG or AVIF on
  size? Five rounds of measurement for
  [images](https://github.com/noelruault/images). **Result: no, with one narrow
  exception.** A Potts/Mumford-Shah region coder beats WebP by 1–6% below
  ~29.2 dB and loses above it; it never beats AVIF, trailing 8–52% across the
  whole range. Sixteen mechanisms — spanning physics, collective systems, vision
  science and spacecraft downlink — tested and killed with numbers, including
  the proof that "don't serialize, let the decoder regrow the regions" is a
  rename rather than a loophole. The one actionable win is an engineering one:
  **32,924 primitives where 1,685 suffice** at identical fidelity. Unusually,
  the record also carries its own casualty list — six claims the investigation
  produced, believed, published, and then falsified against its own
  measurements — and the practices that caught them.

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

## Parked ideas

Ideas set aside are recorded in [`PARKED.md`](PARKED.md) and in each study's own `PARKED.md`, with the number that parked them, **what they depend on**, and a concrete trigger for reviving them.

The dependency field is the point. An idea is usually parked because of a fact that is true at the time — a baseline, a threshold, a constant — and when that fact moves, the idea can become good again with nobody noticing. That has already happened here: a measured −6.40% improvement to the contour coder became worth zero when an unrelated commit improved a *different* coder and moved the crossover between them. Nothing about the parked work changed.

**Re-read a study's parked register whenever any baseline in that study moves.**
