# aseprite-plugin

The research record behind bringing **[pixelize](https://github.com/noelruault/pixelize)
to [Aseprite](https://www.aseprite.org/)** — both the extension that wraps the
engine and the *palette-derivation* ("merge similar colors") feature we plan to
add to the engine next.

This folder is the *evidence trail*, not the product. It exists so the design
calls — binary-backed extension, a `quantize` package built on pixelize's
existing nearest-color core, a CQ100/ΔE2000 benchmark — can be re-derived from
cited sources rather than taken on faith.

## What this is

A short numbered series, each citing primary sources (the official `aseprite/api`
and `aseprite/docs` repos, the Aseprite C++ source, the color-quantization
literature, and the open-source quantizers we benchmarked against):

- **[01-extension-quality.md](01-extension-quality.md)** — how the best public
  Aseprite extensions are built: manifest, plugin lifecycle, Dialog idioms, the
  script-security sandbox (read from `security.cpp`, not the folklore), packaging
  and CI. A checklist to separate a polished extension from a hobby script.
- **[02-quantization-survey.md](02-quantization-survey.md)** — the prior art for
  "derive a palette from an image": median cut, Wu, octree, k-means, and
  agglomerative merge-by-threshold; what the production tools actually use; and
  the analysis of Astropulse's (MIT, public) K-Centroid Downscale.
- **[03-quantize-design.md](03-quantize-design.md)** — the actionable output: the
  `quantize` Go package design, the CLI flag proposal (`-palette auto:N`,
  `-quantize ALGO`, `-merge DIST`), and a reproducible benchmark to *prove*
  better-than-incumbent quality.

## The one idea that ties it together

pixelize already ships the hard part of every color quantizer. "Reduce an image
to N colors" splits into two sub-problems: **(A) choose the palette** and
**(B) map each pixel to it**. (B) is a nearest-color query — and pixelize's
exact kd-tree matcher (the subject of the [nearest-color-scaling](../nearest-color-scaling/)
record) already does it faster and more accurately than ImageMagick. Every
quantizer needs (B): it is k-means' inner assignment loop *and* the final mapping
pass of median cut / Wu / octree (ImageMagick literally names that stage
"Assignment"). So the new `quantize` package is a thin layer of *palette-selection*
strategies on top of the *palette-assignment* core pixelize already owns. That is
why workflow B belongs **in** pixelize, as a package — not in a separate project.

## Why it lives here (and not in pixelize)

Same reason as the `nearest-color-scaling` record: a raw research record —
including notes on third-party MIT source — is documentation, not something a
binary imports. When we commit to building, the *execution plan* that drives the
build belongs in pixelize's `.plans/`; only this evidence trail lives here.

## Provenance

Produced 2026-06 via the deep-research harness (five parallel search angles →
source fetch → adversarial verification → synthesis). Three load-bearing,
flagged-uncertain claims were re-verified against primary sources before being
relied on: `app.fs.tempPath` exists (`aseprite/api`), K-Centroid is MIT © 2023
Astropulse (its `LICENSE`), and the sandbox gates `os.execute`/`io.open` while
removing `os.tmpname`/`os.exit` (`aseprite/api` README + `security.cpp`).
