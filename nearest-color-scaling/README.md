# nearest-color-scaling

The complete research record behind **[pixelize](https://github.com/noelruault/pixelize)'s
nearest-color palette matcher** — how to map every pixel of an image to its
nearest color in a fixed palette, as fast as possible, without giving up
correctness.

This folder is the *evidence trail*, not the product. The product is the
matcher that ships in pixelize (the exact kd-tree branch-and-bound, the
parallel scan, the run-length collapse, the 6-bit fast-mode LUT). This is the
twelve-report investigation that decided which of those to build and proved,
with measured numbers, that they were the right calls.

## What this is

A numbered series of research reports (`01`–`12`), each paired with its raw
benchmark/data companion (`*-data.txt`), plus a reverse-engineering scratch
subdirectory. Together they form a self-contained record: every headline claim
traces to either a verbatim source quote, a cited library/paper, or a
reproducible benchmark whose raw output is in the matching `*-data.txt`.

Nothing here is compiled, imported, or shipped by any binary. It is documentation
and data — written to think the problem through and to keep the evidence so the
conclusions can be re-derived.

## Why it lives here (and not in pixelize)

This research was produced in sessions whose tooling was network-scoped to
`noelruault/pixelize` only, running in ephemeral containers. Committing into the
pixelize app repo was the only way to preserve the work durably at the time — but
a raw research record (including third-party source under its own license; see
below) does not belong in an application repo long-term. `noelruault/research`
is its permanent home.

The pixelize-side **planning** files that *consume* this research —
`.plans/00-overview.md` (the judged synthesis) and `.plans/01-execution-plan.md`
(the phased build plan) — deliberately stayed in pixelize, because they drive the
build. Only the raw research record moved here. pixelize points back to this
folder from `.plans/research/MOVED.md` and from its README benchmark section.

## Headline findings

- **pixelize can be both more correct *and* faster than ImageMagick.**
  ImageMagick's non-dithered `-remap` is measurably approximate at large
  palettes: it picks a non-nearest color on ~0.3% of pixels at 16 colors but
  ~13–22% at 162–256 colors, because it searches only the parent octree subtree,
  not the whole tree (report 01, reverse-engineered from IM 6.9.12 source; report
  04, measured). pixelize's exact kd-tree branch-and-bound is **bit-exact (0%
  non-nearest, verified vs brute force)** and roughly **30× faster than IM at 4K**.

- **No single algorithm wins every regime.** The right design is a *strategy
  selector* dispatching on `(palette size P, image size N, exact-vs-fast need)`,
  with cheap universal enhancements (run-length collapse, row parallelism, SoA
  buffers) layered underneath (report 00 synthesis; reports 06–11).

- **A precomputed inverse-colormap LUT makes per-pixel cost independent of
  palette size** — ~150–200× faster than a serial linear scan at 4K/256 colors —
  at the price of a small bounded approximation (~2–14% of pixels take a
  second-nearest color depending on LUT precision). That trade is why pixelize
  offers `-lut` only for batch/watch, never for a single exact image (report 03;
  report 11).

- **Several "obvious" optimizations were measured and rejected**, on purpose, so
  the design reads as deliberately finished rather than open: Hamerly +
  previous-pixel coherence pruning, Morton/Z-order LUT layout, and an
  SoA+AVX2 distance kernel all lost to the simpler shipped code on real images
  (report 12).

## The reports

| # | Report | What it establishes |
|---|--------|---------------------|
| 00 | *(synthesis — stays in pixelize, `.plans/00-overview.md`)* | The judged verdict-per-regime and dispatch table that drove the build |
| 01 | `01-imagemagick-reverse-engineering.md` | Line-by-line trace of IM 6.9.12 `-remap` / `quantize.c`; why its remap is approximate |
| 02 | `02-algorithms-and-libraries-survey.md` | Survey of search structures, 3D-LUTs, SIMD, metrics, parallelism; ranked shortlist for a Go impl |
| 03 | `03-experiments.md` | First throwaway-prototype bake-off across palette/image sizes vs IM and the then-current code |
| 04 | `04-informed-challenger.md` | Adversarial re-measurement that confirmed IM's approximation and the kd-tree's exactness |
| 05 | `05-cross-disciplinary-transfer.md` | Ideas borrowed from adjacent fields (ANN search, vector quantization, graphics) |
| 06 | `06-puzzle-exact-structures.md` | Exact-match data-structure variants explored |
| 07 | `07-puzzle-enhancement-combinations.md` | Combinations of the cheap enhancements |
| 08 | `08-scan-variants.md` | Linear-scan variants (serial, parallel, layout) |
| 09 | `09-kd-variants.md` | kd-tree variants and the prune-correctness fix |
| 10 | `10-runlength-variants.md` | Run-length short-circuit variants |
| 11 | `11-lut-fastmode-variants.md` | Fast-mode LUT precision/error variants |
| 12 | `12-phase3-enhancement-variances.md` | Close-out: the three enhancements measured and formally rejected |

Each `NN-*.md` report has a `NN-*-data.txt` companion holding its raw benchmark
output and tables.

## Third-party material — `01-imagemagick-reverse-engineering-scratch/`

> ⚠️ This subdirectory contains **verbatim extracts of ImageMagick's own source
> code** (`MagickCore/quantize.c`, ImageMagick 6.9.12-98), included **only as the
> analyzed subject** of report 01. They are **© ImageMagick Studio LLC**, licensed
> under the [ImageMagick License](https://imagemagick.org/script/license.php) — an
> Apache-2.0–style license — **not** under this repository's license. The files
> prefixed `im-` are upstream code; the `notes-*` files are original
> reverse-engineering notes. The directory's own `README.md` carries the full
> attribution and must travel with these files.

## Provenance

Imported from `noelruault/pixelize`, branch
`claude/extract-nearest-color-scaling-VtO3b` (pixelize@551d4f2), where the record
was staged under `.plans/research/`. Original authorship lineage traces to
pixelize branch `claude/readme-skill-agent-pixelize-I4GK3`
(commit `549498cb3aa8ff7b8d7d9a75160d18ad4c83a525`).

`AGENT-MIGRATION-INSTRUCTIONS.md` and `MIGRATION.md` in this folder are the
**historical migration runbook**, kept for context now that the move is complete.
They describe how the record reached this repo; the README you are reading is the
current front door.

## License

The original research reports, data files, and `notes-*` files are MIT (this
repository's license). The `im-*` files in
`01-imagemagick-reverse-engineering-scratch/` are third-party ImageMagick source
under the ImageMagick License — see that directory's `README.md`.
