# shapes-image-file-format

The complete research record behind an attempt to build **an image format made of shapes instead of pixels** — reduce a picture to a small palette, cover it with coloured regions, and ship the geometry — and the measured answer to whether that can beat WebP, PNG, JPEG or AVIF on file size.

**The answer is no, with one narrow exception that is not worth adopting a format for.** This folder is the evidence trail for that conclusion, including the six claims the investigation produced and then falsified against its own measurements.

## What this is

Five rounds of measurement, each with its report and its raw data companion, plus a sixth report that is the honest record of what went wrong. The work was done for [`noelruault/images`](https://github.com/noelruault/images), a project that converts rasters into rectangle covers, and it settles the compression question that project kept re-asking.

Nothing here is compiled, imported, or shipped by any binary. The `code/` directory holds the runnable experiments so every number can be re-derived.

## Headline findings

- **A shape representation cannot beat AVIF at any rate on the eval image** — it trails by 8–52% across the whole measured range, and the reason is structural rather than an engineering gap.

- **It can beat WebP, by 1–6%, below ~29.2 dB.** Above that crossover WebP pulls ahead and the gap widens with rate. This appears to be the first matched-fidelity measurement of a region coder against a modern codec; the classical "geometry wins at low bitrate" literature benchmarks against JPEG-2000, not VP8/AV1-era codecs. *(That novelty claim is unverified — see the caveat in report 04.)*

- **The mechanism is an isoperimetric argument with a sign change.** Boundary cost grows as √(region count) while a block codec keeps paying per-block mode and coefficient bits that do not shrink as fast under deep quantization. So shapes win where regions are few, and lose as soon as fidelity forces the region count up. Measured at the operating point, region walls are **2.22× longer** than compact cells of the same areas — and that excess *is* the image information, which is why emergent (cheap) geometry is also uninformative geometry.

- **The shipped rect cover is leaving an order of magnitude on the table inside the shape idea.** A photo that a greedy cover splits into **32,924 rects** is **1,685 regions** under an energy-minimizing Potts merge at identical fidelity, and 153 regions at a coarser one. That is a 9× byte reduction and a 20–85× primitive reduction, and it is the one actionable engineering result here.

- **Sixteen mechanisms were tested and killed with numbers**, across physics, collective systems, vision science and spacecraft downlink — including "don't serialize, let the decoder regrow the regions", which is provably a rename: any region reconstructible from decoded pixels is a function of the causal past, so it *is* a context model and is bounded by `H(X | causal past)`.

## The reports

| # | Report | What it establishes |
|---|--------|---------------------|
| 01 | [`01-encoder-bakeoff.md`](01-encoder-bakeoff.md) | Baseline bake-off: the shipped SVG rect cover is the **worst** of seven encoders; a plain indexed PNG beats every shape encoder |
| 02 | [`02-webp-dissection.md`](02-webp-dissection.md) | WebP's four transforms ported one at a time; the lever turns out not to be shapes, and the path that makes shapes small converts them into a raster codec |
| 03 | [`03-shared-prior.md`](03-shared-prior.md) | Dictionary and cached-corpus priors, including a best-case corpus built to favour shapes: **1.02× for shapes, 1.01× for WebP** — the hoped-for asymmetry does not exist |
| 04 | [`04-adjudication.md`](04-adjudication.md) | Four independent adversarial investigations, sixteen killed mechanisms, and the four agreeing explanations for the ceiling |
| 05 | [`05-low-rate-crossover.md`](05-low-rate-crossover.md) | The one place shapes win: below ~29.2 dB, by 1–6% over WebP, never over AVIF |
| 06 | [`06-corrections-and-falsifications.md`](06-corrections-and-falsifications.md) | The six claims this investigation produced and then killed, and the measurement that killed each |
| 07 | [`07-method-what-worked.md`](07-method-what-worked.md) | The practices that caught real errors here — fixed eval, reproduce-before-believing, regenerate-don't-transcribe, run-it-twice — written for whoever runs the next one |

Data companions: [`03-corpus-dictionary-data.txt`](03-corpus-dictionary-data.txt), [`04-region-merge-frontier-data.txt`](04-region-merge-frontier-data.txt), [`05-codec-rd-sweep-data.txt`](05-codec-rd-sweep-data.txt), [`05-matched-fidelity-comparison-data.txt`](05-matched-fidelity-comparison-data.txt).

## The eval

One fixed evaluation throughout, so every round is comparable:

- **Image:** macOS Sierra wallpaper at 512×288 (147,456 px).
- **Metric:** RGB PSNR, the definition used by the `images` harness, applied identically to external codecs so nothing is measured on a friendlier metric than anything else.
- **External codecs:** `cwebp`, `avifenc`/`avifdec`, `pngquant`, `sips`, at their own quality settings, sweeping the full range.
- **Shape coder:** Potts / piecewise-constant Mumford-Shah region merge (Koepfler-López-Morel scale-space) plus zero-temperature Ising wall relaxation, with crack-edge boundaries and region colours both entropy-coded.

## Reading this critically — what the numbers are not

These are load-bearing caveats, not disclaimers. Every headline above inherits them.

- **The shape coder's bytes are an idealised lower bound.** They are the cross-entropy of an adaptive binary arithmetic coder — no container, no header, no real bitstream. WebP and AVIF numbers are whole files. The AVIF column in report 05 discounts its measured 297 B container floor to keep the comparison from flattering us; WebP's ~44 B floor is left in.
- **PSNR only.** At 24–26 dB the two failure modes are not comparable: the shape render posterizes into flat regions, WebP goes blurry and blocky. MS-SSIM or a subjective test could plausibly rank them differently, and report 04 already found flat interiors losing badly on smooth content (MS-SSIM 0.902 vs AVIF's 0.983 on sky) while winning on texture.
- **One image.** No Kodak, no CLIC, no BD-rate. A single 512×288 photograph.
- **Reproducibility was not free.** The region merge was nondeterministic for most of this investigation (Go randomizes map iteration; equal-key merge candidates popped in varying order), with up to 7% spread in bytes at the coarse end. Report 06 covers the fix and what it invalidated.

None of that is fixed by more effort on the encoder. It is fixed by building a real container and running a multi-image perceptual sweep — which would be the work required to make any of this publishable, and which was not done.

## Verdict

**The compression ambition is parked.** A shape representation ties or slightly beats WebP below ~29.2 dB and loses to AVIF everywhere by 8–52%. Beating the second-best codec by a few percent in a band where the best codec leads by half again is not a reason for anyone to adopt a format.

What survives is not a bytes claim: ~1,700 named, editable, independently animatable regions instead of 32,924 anonymous rects; immediate-mode drawing with no decode step; per-region addressability; a truncatable progressive stream; and per-pixel error bounds. Those are real properties no raster codec offers, and they happen to matter most in exactly the low-rate band where the byte numbers are also least bad.

## Provenance

Produced in a single investigation for [`noelruault/images`](https://github.com/noelruault/images), where the working record lived at `research/RESULTS.md`. The planning files that *consume* this research stayed in the projects they drive: [`pixelize/.plans/shape-export.md`](https://github.com/noelruault/pixelize/blob/main/.plans/shape-export.md) and [`sprites/.plans/image-to-rects.md`](https://github.com/noelruault/sprites/blob/main/.plans/image-to-rects.md).
