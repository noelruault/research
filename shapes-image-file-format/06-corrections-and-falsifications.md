# 06 — The six claims this investigation killed

Every claim below was produced *by this investigation*, believed, written down, and in most cases committed to a repo README — then falsified by a later measurement in the same investigation. They are collected here because the failure pattern is more useful than any single number: **five of the six were a measurement compared against the wrong baseline, and the sixth was a measurement compared against itself run once.**

## 1. "Beats WebP by 31%"

**Claimed.** An order-2 context coder on the palette index map reached 17.4 KB against WebP's 25.2 KB.

**Why it was wrong.** Lossy-versus-lossless framing error. 17.4 KB reproduces our *own quantized grid* — losslessly coding our own quantization error, a fidelity nobody asked for — while WebP was measured in lossless mode on the same grid. At matched fidelity on the *original* image, WebP q34 = 12.35 KB @ 28.63 dB and AVIF q30 = 8.9 KB @ 28.71 dB.

**Corrected.** 17.4 KB is **1.41× behind WebP and 1.95× behind AVIF**, not 31% ahead.

**Lesson.** A comparison is only meaningful at matched fidelity on the same source. "Lossless of my own lossy output" is a fidelity you invented and only you are being scored on.

## 2. "Order-3 overfits — context dilution — we are at the entropy floor"

**Claimed.** Order-3 (18.3 KB) is worse than order-2 (17.4 KB), therefore the context is diluted and 17.4 KB is the floor.

**Why it was wrong.** Order-3 has 4,096 contexts over 147,456 samples — about 36 samples each. That is *starvation*, not saturation, and the two have opposite remedies. The test that distinguishes them: if it were dilution, mixing could not recover it either.

**Corrected.** Online logistic mixing of orders 0–4 (`code/mix/`) gives 16.2 KB, below order-2's 16.9 KB standalone. The model was starved. The raster baseline is 16.2 KB.

**Lesson.** "It got worse when I added information" has two explanations with opposite fixes. Distinguish them with an experiment, not with the more flattering label.

## 3. "Geometry alone costs 19.2 KB — that is the wall"

**Claimed.** A geometry-cost experiment (`code/geom/`) with colour free measured boundaries at 19.2 KB, which put shapes structurally out of reach.

**Why it was wrong.** Artifact of my own segmenter. The Felzenszwalb segmentation produced jagged, high-perimeter boundaries, and boundary cost is proportional to perimeter.

**Corrected.** A proper energy-minimizing segmenter (Potts merge + zero-temperature Ising wall relaxation) costs **9.0 KB** for the geometry at the same fidelity. The relaxation alone — straightening walls without changing the region count — cut bytes 15%.

**Lesson.** When a negative result depends on a component you wrote in an afternoon, the result is about your component. Measure the ceiling with the best available implementation before declaring a wall.

## 4. "It renders at 8K for the same bytes"

**Claimed.** A 96px flat-art cover is 9 KB gzipped and renders at 8K for the same 9 KB, while an 8K PNG would be megabytes.

**Why it was wrong.** Fake vectorness. A rect cover of a *quantized grid* has no sub-grid geometry, so upscaling it is mathematically identical to nearest-neighbour upscaling of a tiny PNG — which is far smaller than 9 KB.

**Corrected.** On 4×-upscaled pixel art the region coder does win (5,422 B vs order-2's 6,657 B vs WebP-lossless's 9,246 B), but at native resolution it loses (1,386 B vs 1,173 B). Resolution independence is a genuine win only when the geometry is authored *above* the pixel grid, which a cover of a raster never is.

**Lesson.** Check whether the property you are selling is in the representation or in the comparison. "Scales for free" is only real if there is sub-pixel information to scale.

## 5. "9.8× smaller than the source"

**Claimed.** The project README reported wins up to 31× against the original JPEG.

**Why it was wrong.** The pipeline downscales and quantizes before covering. Comparing the output against the full-resolution original credits the resize and the palette reduction to the format, which did neither as a compression act.

**Corrected.** Against **the same pixels** — the identical downscaled, quantized image encoded losslessly by WebP — the format is **6.9× larger**, on every input tested including flat art. The sign flips.

**Lesson.** The baseline must be the same pixels. If your pipeline throws information away before measuring, the discard is not a compression win.

## 6. "3–9% better than WebP at low rate", and the 8.6% bottom-end figure

**Claimed.** Report 05's first draft: the shape coder beats WebP by 3–9%, bottoming at 2,765 B vs 3,024 B (−8.6%) at 23.99 dB, with the crossover at ~28.4 dB.

**Why it was wrong.** Two compounding errors.

*The measurement was nondeterministic.* The region merge builds its candidate heap by ranging over a Go map, and Go randomizes map iteration order. Equal-key merge candidates — common early in the coarsening, when many pairs share an identical `dSSE/dL` — therefore popped in a different order each run, producing a different scale-space. Spread at the coarse end was up to **7% in bytes**. The headline had been taken from a single lucky run.

*The comparison was interpolated by eye.* Codec bytes at each shape PSNR were read off the sweep by hand rather than computed.

**Corrected.** Fixed by giving the heap comparator a **total order** — ties on key break on `(a, b)` — which makes the merge deterministic; verified identical across three consecutive runs. The comparison was rewritten as `code/compare.py`, which interpolates the codec curves at every measured PSNR with no extrapolation. Corrected result: **1–6% over WebP, crossover at ~29.2 dB**. The deterministic run also reproduces report 04's headline exactly (1,685 regions @ 28.66 dB, 11.9 KB), which is what confirms the fix rather than merely quieting the noise.

**Lesson.** Run it twice before you publish it once. A result you have not seen reproduce is a sample, not a measurement — and `range` over a Go map is a silent randomizer that will not announce itself.

## The pattern

| # | Claim | Failure mode |
|---|---|---|
| 1 | beats WebP 31% | wrong baseline — mismatched fidelity |
| 2 | at the entropy floor | wrong diagnosis — starvation read as saturation |
| 3 | geometry costs 19.2 KB | wrong baseline — measured my own weak component |
| 4 | renders at 8K free | wrong baseline — property was in the comparison, not the format |
| 5 | 9.8× smaller than source | wrong baseline — credited the downscale to the format |
| 6 | 3–9% at low rate | unreproduced single run + hand interpolation |

Five of six are baseline errors, and every one of them flattered the hypothesis under test. None was caught by reasoning; each was caught by a later measurement that happened to overlap. The only structural defence found was the rule applied to the four investigating agents in report 04 — **reproduce the shared eval before your findings are believed** — which caught a fifth error in flight, when one agent's contradicting headline turned out to come from it silently substituting a different image.
