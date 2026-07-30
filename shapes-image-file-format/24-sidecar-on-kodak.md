# 24 — The positioning survives the corpus that killed the parity claim

**Question (P5c).** Report 23 found the shape coder loses to WebP on all 24 Kodak images, mean +27.8%, and left one thing unmeasured: report 19's comparison against **WebP plus a region-map sidecar** — the baseline that matches the product, since a consumer wanting the same capability has to pay for the mask too. That margin is a *ratio*, and on busy content the region map gets dearer for both sides. It was the one place the positioning could still hold. Run it on all 24.

**Answer.** **It holds on all twenty-four.** Mean **+30.5%**, range **+18.7% to +37.8%**, twenty-four wins out of twenty-four.

Knobs: WebP `-m 6` at matched fidelity — the same files report 23 produced. The sidecar gets the **best of** our interleaved boundary coder and the best generic label-map encoding (`xz -9e` or `brotli -q11` on the raw 16-bit label plane). Shape side is the **real SHPC v1 container**. Same operating point as report 23: each image's mark nearest 4,500 regions.

## The two comparisons side by side

| image | SHPC | WebP | vs WebP alone | raster + sidecar | **vs raster+sidecar** |
|---|---|---|---|---|---|
| kodim02 | 24,493 | 23,632 | +3.6% | 39,407 | **+37.8%** |
| kodim08 | 27,172 | 25,674 | +5.8% | 42,878 | **+36.6%** |
| kodim01 | 28,013 | 25,604 | +9.4% | 44,641 | **+37.2%** |
| kodim14 | 30,062 | 26,134 | +15.0% | 46,177 | +34.9% |
| kodim05 | 30,950 | 26,652 | +16.1% | 47,332 | +34.6% |
| kodim24 | 27,743 | 23,652 | +17.3% | 40,875 | +32.1% |
| kodim11 | 26,407 | 22,182 | +19.0% | 39,039 | +32.4% |
| kodim13 | 30,878 | 25,404 | +21.5% | 46,256 | +33.2% |
| kodim22 | 29,458 | 23,828 | +23.6% | 43,456 | +32.2% |
| kodim12 | 25,464 | 20,398 | +24.8% | 37,263 | +31.7% |
| kodim16 | 27,210 | 21,738 | +25.2% | 40,936 | +33.5% |
| kodim21 | 26,198 | 20,894 | +25.4% | 37,975 | +31.0% |
| kodim10 | 24,466 | 19,484 | +25.6% | 34,939 | +30.0% |
| kodim18 | 29,744 | 23,134 | +28.6% | 42,596 | +30.2% |
| kodim19 | 26,318 | 20,400 | +29.0% | 37,977 | +30.7% |
| kodim06 | 30,077 | 22,864 | +31.5% | 43,159 | +30.3% |
| kodim03 | 25,530 | 19,238 | +32.7% | 35,402 | +27.9% |
| kodim04 | 29,851 | 21,712 | +37.5% | 41,926 | +28.8% |
| kodim09 | 23,421 | 16,848 | +39.0% | 31,599 | +25.9% |
| kodim07 | 25,370 | 18,226 | +39.2% | 33,639 | +24.6% |
| kodim15 | 27,007 | 19,328 | +39.7% | 36,450 | +25.9% |
| kodim20 | 23,949 | 17,090 | +40.1% | 32,167 | +25.5% |
| kodim17 | 26,915 | 18,412 | +46.2% | 36,496 | +26.3% |
| kodim23 | 25,702 | 14,986 | **+71.5%** | 31,617 | **+18.7%** |

| | mean | range | wins |
|---|---|---|---|
| vs WebP alone | +27.8% | +3.6% .. +71.5% | **0 / 24** |
| **vs WebP + sidecar** | **+30.5%** | +18.7% .. +37.8% | **24 / 24** |

*Sierra, for comparison: +0.93% vs WebP alone, +44.1% vs WebP+sidecar.*

## Why it survives when parity did not

**The sidecar is expensive, and it is expensive for the same reason the shape coder is.** On busy content the region map costs more — but the raster consumer pays that cost *on top of* a full set of pixels, while the shape coder pays it *instead of* them. The worse the content is for boundaries, the worse it is for both sides, so the ratio is far more stable than the difference.

The correlation makes it precise: **corr(WebP-alone delta, sidecar margin) = −0.967**. The two move together almost perfectly. kodim23 is the extreme of both — worst against WebP alone at +71.5%, worst against the sidecar at +18.7% — and it is still an 18.7% win.

**Our boundary coder is 2.14× better than the best generic label-map encoding on average**, holding across all 24 images. Report 19 found 2.46× on the Sierra image; that result generalises too. Pricing the sidecar with our own coder remains the steelman, not self-dealing — a consumer using downloadable tools does considerably worse than these numbers show.

## What the study can now claim

**Against WebP alone the format is not competitive on photographs.** Zero wins in 24, mean +27.8%. Report 23 stands.

**Against the baseline that matches what it delivers, it wins everywhere measured** — 24 of 24 Kodak images plus the Sierra wallpaper, margins from +18.7% to +44.1%, never once losing.

Those are not in tension. They say the format is a poor *image codec* and a good *structured-image format*: if you want pixels, use WebP; if you want pixels **and** a segmentation, this is the cheapest way measured to get both, by roughly 30%.

That is the first claim in this study to survive a corpus rather than rest on one image.

## Caveats

- **Both sides here deliver the same partition.** If a consumer would accept a coarser or approximate mask — one produced by re-segmenting on the client, say — their sidecar gets cheaper or free, and this comparison does not apply. Report 13's argument for why a *transmitted* partition is worth paying for (deterministic, identical on every client, authored rather than recomputed) is a capability argument, not a bytes one, and it is untested against a re-segmentation baseline.
- **No dedicated modern region-map codec was benchmarked** — JBIG2 and CM-class coders are still not installed. "Nothing off-the-shelf beats our coder" rests on two encodings per image, not a survey. Unchanged from report 19, and still **unverified**.
- One operating point per image (~4,500 regions), one metric (PSNR), 8-bit RGB, no alpha.
- The raster+sidecar figure combines a real WebP file with a **cross-entropy** wall cost for the sidecar when ours wins. That flatters the sidecar's *shape-coder-based* variant slightly — but the generic sidecar numbers are real files and the margin holds against those too, at 2.14× the cost.
