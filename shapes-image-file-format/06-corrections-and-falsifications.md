# 06 — The thirteen claims this investigation killed

Every claim below was produced *by this investigation*, believed, written down, and in most cases committed to a repo README — then falsified by a later measurement in the same investigation. They are collected here because the failure pattern is more useful than any single number: **eight of the thirteen were a measurement compared against the wrong baseline, one was a measurement compared against itself run once, one was a bug class declared closed after fixing a single instance, one was a real result generalised past the one axis the eval had frozen, one was a coder that turned out not to be decodable at all, and one was a mechanism fitted to three data points and published as a predictor.** Note that #11 breaks the pattern in an instructive way: it is the first baseline error that flattered *against* the hypothesis, and it was caused by over-correcting #10.

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

## 7. "The merge is deterministic now"

**Claimed.** Implicitly, by #6: the heap comparator was the randomizer, it has a total order, the coder is deterministic.

**Why it was wrong.** The heap was one of **three** places where `range` over a Go map chose between equally good options. The other two were only found because pricing the coder at 3840×2160 forced a rewrite — `colorBytes` builds one `map[int32]int` per region to find each region's predictor, which at six million regions is a gigabyte of maps before any counting starts, so it was replaced with a sort over adjacency records. The rewrite disagreed with the original by 338 B on a 132,674-region partition. That looked like a bug in the rewrite. It was not: **six calls to the original `colorBytes` on one fixed partition inside one process returned six different answers**, spanning 224 B.

```
262835.47   262725.65   262836.15   262735.37   262949.40   262748.13    B
```

The predictor is the already-decoded neighbour sharing the longest wall, selected with `ln > bestLen` while ranging over a map. At fine partitions most adjacent pairs touch along a *single* crack edge, so ties are not an edge case — they are the common case, and the winner was whichever one Go's randomized iteration offered first. The same shape of loop sat in `paletteColorBytes` and in the Ising relaxation's candidate selection.

**Corrected.** All three now carry a total order: longest wall first, lowest region id on a tie. `colorBytes` returns an identical value on six consecutive calls, agrees to the last bit with the sorted rewrite on both a 132,674-region and a 7,040-region partition, and the relaxation fix leaves `frontier`'s output **byte-identical** to the committed data file.

**What it changed in the published numbers: nothing.** The frontier prices colours with `colorBytes2`, which predicts from the boundary-length-weighted mean of *all* decoded neighbours — a sum, with no argmax to break — so it was never exposed. The bug lived only in the paths the published headlines do not rest on. It would have contaminated the first new number it touched, which is exactly what the 4K work was about to produce.

**Lesson.** One instance of a bug class is evidence about the class, not about the instance. Having found once that map iteration order had leaked into a result, the correct next move was to grep every `range` over a map whose body picks a winner — not to fix the one that had already caused visible damage and call the category closed.

## 8. "The shape coder beats WebP below ~29.2 dB"

**Claimed.** Report 05, corrected and believed: at matched fidelity the region coder is 1–6% smaller than WebP below a crossover at ~29.2 dB.

**Why it was incomplete.** True, and measured on one image at one size: **512×288**. The eval was fixed early and never varied, which is what made every round comparable — and which meant resolution was never a variable at all. Report 08 puts the identical coder on the same picture at 3840×2160, 1920×1080, 960×540 and 512×288, and the deficit against WebP grows monotonically at every fidelity as resolution rises. See report 08 for the table.

**Lesson.** A fixed eval buys comparability across rounds and hides whatever it holds constant. Before generalising a win, vary the one axis the eval froze.

## 9. "1–6% better than WebP below 29.2 dB" — the baseline was WebP on its default setting

**Claimed.** Report 05, after correction #6 and believed for the rest of the investigation: below a crossover at ~29.2 dB the shape coder is 1–6% smaller than WebP at matched fidelity.

**Why it was wrong.** The sweep ran `cwebp -q N`. `cwebp`'s default method is **4**; `-m 6` costs encode time, changes nothing about decoding, and is what any build pipeline emitting WebP actually uses. On this image `-m 6` is **5.6–8.9% smaller** — bigger than the whole effect being claimed. `avifenc -s 4` was likewise not AVIF's best. So the comparison was against a WebP that had been left with bytes on the table, and the shape coder was credited with them.

**Corrected.** Re-running the identical sweep at `-m 6` and `-s 0`: the crossover falls from **29.17 dB to 26.09 dB**, the figure at the eval's own fidelity goes from **−1.9% to +2.7%**, and across the low band the sign alternates between adjacent samples — −2.1, −3.7, −0.9, +1.6, −3.2, −2.6, −1.0, +1.4. That is a wash, not a win. Both tables are kept side by side in report 05 and both are printed by `code/compare.py`.

**This is the same error as #1, #3, #4 and #5**, in its least obvious dress. Those compared against the wrong *thing*; this compared against the right thing turned down. A default flag is easy to read as neutral, and it is not — it is a configuration choice made on the encoder author's behalf by whoever picked a speed/size trade-off for the CLI, and it favoured the hypothesis.

**Lesson.** Steelman the baseline's *settings*, not just its identity. Before claiming a few percent over a shipping codec, spend ten minutes finding out whether its flags were left at defaults — the margin you are claiming is often smaller than the margin the defaults gave away.

## 10. "WebP cannot go this small" — the baseline was WebP with resampling forbidden

**Claimed.** Report 08 result 4, and the README headline it earned within the hour: at 3840×2160 `cwebp` bottoms out at `q0` = 85,102 B, so the shape coder's 19,819 B file is **4.3× smaller than anything WebP can emit**. Written up as the one capability the region coder has that WebP does not — the single survivor of a study that had killed everything else.

**Why it was wrong.** The floor is real, and it is the floor of `cwebp` *at a fixed encode resolution*. Nobody delivers a 20 KB rendering of a 4K photograph by turning quality down at 4K; they encode at a smaller size and let the client scale it up. `srcset` has been that pipeline for a decade. Its output is still 3840×2160 pixels on screen, scored against the same original on the same metric, so it was always in the same contest — the measurement had simply never been run, because the ladder's byte-matching searched quality and nothing else.

**Corrected.** Searching resolution and quality jointly at the shape coder's own two sub-floor byte targets:

| target | best WebP | its size | WebP PSNR | shape coder | verdict |
|---|---|---|---|---|---|
| 19,819 B | `-q 18 -resize 960 540` | 20,066 B | 24.54 dB | 21.99 dB | WebP **+2.55 dB** |
| 50,016 B | `-q 33 -resize 1280 720` | 49,774 B | 26.34 dB | 24.99 dB | WebP **+1.35 dB** |

This is not one lucky resolution. At 19,819 B **every rung that can reach the target wins** — 1280×720, 960×540, 640×360, 480×270 and 384×216, by +1.82 to +2.55 dB. (1920×1080 is already at `-q 1` by 31,922 B and cannot reach that size at all, which is the same floor effect one rung down.) At 50,016 B the win is narrower and does run out: the top four rungs take it by +0.19 to +1.35 dB, and the two most aggressive downscales *lose*, by −0.54 and −1.06 dB, because past some point the resample destroys more than the saved bitrate buys back. The rate ladder therefore quotes the best rung at each target, which is the steelman: `-q 18 -resize 960 540` and `-q 33 -resize 1280 720`.

Nor is the win an artefact of a good resampler. Repeated with ffmpeg's nearest-neighbour — the worst filter available, and worse than anything a browser would use — WebP still scores 24.17–24.29 dB across the three matched rungs, against the shape coder's 21.99. Data: [`08-rate-floor-steelman-data.txt`](08-rate-floor-steelman-data.txt).

**So the last thing this study claimed the shape idea could do, it cannot.** The rate slider's two smallest steps now compare against a real byte-matched WebP instead of a floor, and WebP wins them by a wider margin than it wins anywhere else on the axis — 2.55 dB at the bottom against 0.67 dB in the middle. The bottom of the rate range, which every previous round had identified as the shape coder's best ground, is where it loses worst.

**This is #9's error one level up.** #9 steelmanned the baseline's flags; this one needed its *pipeline* steelmanned. "`cwebp` cannot produce this file" was true of the command line that was run and false of the format, and the distance between those two statements is where most of this table lives.

**Lesson.** When a baseline appears to have a hard floor, establish whether the floor belongs to the format or to the invocation. A capability claim is a claim about what the other side *cannot* do — the hardest kind to verify, the easiest to want to believe, and the one this study fell for last.

## 11. "The deficit is U-shaped and the low-rate band is where it loses hardest" — only WebP was allowed to resample

**Claimed.** Report 08 result 4, written within an hour of #10: across the byte-matched rate ladder the deficit is −2.55, −1.35, −0.67, −2.39, −3.20 dB, so the low-rate band every earlier round called the shape idea's best hope is where it now does worst.

**Why it was wrong.** #10 established that below `cwebp`'s native floor a 20 KB file is reached by encoding at a lower resolution and upscaling. The ladder was then rebuilt to give WebP exactly that — `rate-build.sh` runs a joint resolution-and-quality search whenever the target sits below the floor. **It never offered the shape coder the same knob.** Rungs 1 and 2 pit a resampled WebP against a shape coder still encoding at native 3840×2160. The delivered pixel count is a choice *both* sides get to make, and only one side was making it.

**Corrected.** Giving the shape coder the same search and the identical `sips` upscale, at the same byte target:

| rung 1, ~20 KB | bytes | PSNR at 4K |
|---|---|---|
| shapes, native 4K — as published | 19,819 B | 21.99 dB |
| WebP `-q 18 -resize 960 540` ↑ | 20,066 B | 24.54 dB |
| **shapes, 960×540 (2,075 regions) ↑** | **20,618 B** | **24.59 dB** |

**That is a tie, not −2.55 dB**, and the U-shape's left arm goes with it. Rung 2 is **unresolved**: no scale-space mark lands within the ladder's ±5% byte tolerance of 50,016 B, and the nearest (53,121 B, +6.2%) scores 25.66 dB against WebP's 26.34 — so WebP probably still leads rung 2 by roughly half the published margin, but that is not byte-matched and interpolating it by hand is precisely falsification #6. It needs a merge run.

**#10 survives; its margin does not.** "Shapes reach a rate WebP cannot" is still dead. "And WebP wins there by 2.55 dB" is now dead too. At 20 KB both coders deliver 3840×2160 and they are level.

**This is the first error in this ledger that flattered *against* the hypothesis**, and it was produced by correcting the one before it. Seven earlier entries were baseline errors in the hypothesis's favour; the reflex built up to catch them steelmanned the baseline and forgot to steelman the subject. The failure mode is not bias toward a conclusion — it is asymmetric effort, applied wherever the last mistake was.

**Lesson.** After retracting a claim, check whether the correction is symmetric. Any knob you hand the baseline — encoder effort, encode resolution, delivery pipeline — the thing under test must be offered too, or the retraction becomes the next error.

## 12. The wall coder's published numbers came from a coder that cannot be decoded

**Claimed.** Implicitly, in every report from 02 onward: the crack-edge CAE coster prices a real bitstream.

**Why it was wrong.** `potts.go:311` reads `get(Hz, x+1, y)` into the `Hz` context — a crack edge to the right in the same row, not yet coded. The same context also reads `Hz(x-1,y)` and `Hz(x-2,y)`, so no scan order supplies all three: left-to-right lacks the right tap, right-to-left lacks the left pair. A decoder-side replay that rebuilds each context only from bits the schedule has actually reached reports **21,554 mismatching contexts at 512×288 and 51,995 at 960×540**.

**Corrected.** Repairing the tap costs **+3.4% of the wall bill at 6,417 regions, +4.6% at 11,121, +6.3% at 19,338, +11.9% at 96,359 and +12.7% at 710,144**. Wherever CAE is the chosen wall coder — which is everywhere above ~1,400 regions at 4K — the published wall figures are optimistic by that much, and report 08's tables have not yet been re-priced. Report 09 result 3 has the detail.

**Lesson.** A cross-entropy coster is not a codec, and nothing forces it to be implementable. If a pricing function is going to stand in for a bitstream across eight reports, it needs a decoder-side replay asserting every context is causal — a few dozen lines, written once, at the start.

## 13. "Parity holds where a hundred regions explain most of the picture"

**Claimed.** Report 22, on the strength of three Kodak images plus the Sierra wallpaper: the byte deficit tracks how much of the image a hundred regions cover — 90% on Sierra where the coder reaches parity, ~40% on Kodak where it loses — and that figure is measurable before encoding, making it a usable predictor of the format's regime.

**Why it was wrong.** Three data points. The mechanism was plausible, it matched report 04's isoperimetric argument, and it was asserted as a finding rather than a hypothesis.

**Corrected.** Across the full Kodak-24 set the correlation between top-100 region coverage and the byte delta is **+0.005**. It explains nothing. A second candidate — lossless PNG bytes per pixel as a complexity proxy — correlates −0.528, points the *wrong* way, and is contradicted outright by 15 of 24 Kodak images being *less* complex than the Sierra wallpaper while all of them lose by 19–72%.

**And the honest state is worse than "wrong predictor".** There is now no characterisation at all of when this format is competitive. Sierra is +0.93% at 4K and +1.3% at Kodak's own size, so it is not a resolution effect; it is an outlier for reasons this study has not identified.

**Lesson.** A mechanism that explains the data you have is not a predictor until it is tested on data you did not use to build it. Three points can support almost any story — and this one was mine, written into a report headline within minutes of seeing the third point.

## The pattern

| # | Claim | Failure mode |
|---|---|---|
| 1 | beats WebP 31% | wrong baseline — mismatched fidelity |
| 2 | at the entropy floor | wrong diagnosis — starvation read as saturation |
| 3 | geometry costs 19.2 KB | wrong baseline — measured my own weak component |
| 4 | renders at 8K free | wrong baseline — property was in the comparison, not the format |
| 5 | 9.8× smaller than source | wrong baseline — credited the downscale to the format |
| 6 | 3–9% at low rate | unreproduced single run + hand interpolation |
| 7 | the merge is deterministic now | fixed one instance, not the class — two more randomizers survived |
| 8 | beats WebP below 29.2 dB | true at the eval's size only — the frozen axis was never varied |
| 9 | 1–6% at low rate | wrong baseline — WebP was left on its default `-m 4` |
| 10 | WebP cannot go this small | wrong baseline — WebP was forbidden to resample |
| 11 | the low-rate band is where it loses hardest | wrong baseline — only WebP was allowed to resample, and this one flattered *against* the hypothesis |
| 12 | the CAE wall numbers price a bitstream | the coster was not decodable — one context tap reads a bit that has not been coded |
| 13 | top-100 coverage predicts the byte deficit | a mechanism fitted to three points; correlation +0.005 across twenty-four |

Eight of twelve are baseline errors, and every one of them flattered the hypothesis under test. None was caught by reasoning; each was caught by a later measurement that happened to overlap. The only structural defence found was the rule applied to the four investigating agents in report 04 — **reproduce the shared eval before your findings are believed** — which caught a fifth error in flight, when one agent's contradicting headline turned out to come from it silently substituting a different image.
