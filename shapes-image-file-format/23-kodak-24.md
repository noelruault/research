# 23 — Zero wins in twenty-four, and the predictor I proposed is dead

**Question (P5b).** Report 22 measured three Kodak images, found the shape coder 9.4–71.5% behind WebP against +0.93% on the Sierra wallpaper, and proposed a predictor: parity holds where a hundred regions explain most of the picture (90% on Sierra, ~40% on Kodak). Run the full 24 and test that.

**Answer.** **Shapes lose on all twenty-four. Mean +27.8%, range +3.6% to +71.5%, zero wins.** And the predictor is falsified: its correlation with the byte delta across 24 images is **+0.005**. It was an artifact of three data points, and I proposed it.

Knobs: both coders native 768×512, **no resampling either side**, WebP `-m 6`, fidelity matched by bisecting `cwebp -q` to the shape render's PSNR, shape side as **real SHPC v1 containers**. Operating point is each image's scale-space mark nearest 4,500 regions, so every image is compared at a comparable structural budget.

## The sweep

| image | regions | PSNR | SHPC | WebP | delta | top-100 cov |
|---|---|---|---|---|---|---|
| kodim02 | 4,472 | 33.86 | 24,493 | 23,632 | **+3.6%** | 65.5% |
| kodim08 | 4,419 | 26.15 | 27,172 | 25,674 | +5.8% | 51.4% |
| kodim01 | 4,604 | 27.59 | 28,013 | 25,604 | +9.4% | 39.9% |
| kodim14 | 4,411 | 28.73 | 30,062 | 26,134 | +15.0% | 40.5% |
| kodim05 | 4,457 | 26.65 | 30,950 | 26,652 | +16.1% | 39.1% |
| kodim24 | 4,373 | 28.07 | 27,743 | 23,652 | +17.3% | 56.7% |
| kodim11 | 4,542 | 30.75 | 26,407 | 22,182 | +19.0% | 60.3% |
| kodim13 | 4,561 | 24.42 | 30,878 | 25,404 | +21.5% | 58.0% |
| kodim22 | 4,433 | 30.77 | 29,458 | 23,828 | +23.6% | 44.1% |
| kodim12 | 4,351 | 35.04 | 25,464 | 20,398 | +24.8% | 58.4% |
| kodim16 | 4,496 | 32.87 | 27,210 | 21,738 | +25.2% | 50.6% |
| kodim21 | 4,519 | 29.91 | 26,198 | 20,894 | +25.4% | 63.4% |
| kodim10 | 4,332 | 34.21 | 24,466 | 19,484 | +25.6% | 45.7% |
| kodim18 | 4,469 | 28.02 | 29,744 | 23,134 | +28.6% | 50.9% |
| kodim19 | 4,400 | 31.02 | 26,318 | 20,400 | +29.0% | 53.5% |
| kodim06 | 4,998 | 29.39 | 30,077 | 22,864 | +31.5% | 61.4% |
| kodim03 | 4,738 | 35.39 | 25,530 | 19,238 | +32.7% | 61.4% |
| kodim04 | 4,746 | 32.99 | 29,851 | 21,712 | +37.5% | 45.4% |
| kodim09 | 4,368 | 34.32 | 23,421 | 16,848 | +39.0% | 66.7% |
| kodim07 | 4,768 | 33.01 | 25,370 | 18,226 | +39.2% | 54.5% |
| kodim15 | 4,606 | 33.19 | 27,007 | 19,328 | +39.7% | 57.5% |
| kodim20 | 4,307 | 33.79 | 23,949 | 17,090 | +40.1% | 69.5% |
| kodim17 | 4,340 | 32.93 | 26,915 | 18,412 | +46.2% | 45.6% |
| kodim23 | 4,387 | 34.76 | 25,702 | 14,986 | **+71.5%** | 42.6% |

**n = 24, mean +27.8%, zero images where the shape coder is smaller.**

## It is content, not resolution

The obvious confound was size — Sierra was measured at 4K, Kodak is 768×512. Held content, varied resolution:

| | regions | PSNR | SHPC | WebP | delta |
|---|---|---|---|---|---|
| Sierra at 3840×2160 (report 21) | 11,121 | 28.51 | 132,301 | 131,082 | **+0.93%** |
| **Sierra at 768×512** — Kodak's own size | 4,497 | 30.39 | 27,130 | 26,788 | **+1.3%** |
| Kodak-24 mean at 768×512 | ~4,500 | — | — | — | **+27.8%** |

**Resolution is not the variable.** The Sierra wallpaper is exceptional at any size.

## Both predictors fail

| proposed predictor | correlation with byte delta |
|---|---|
| top-100 region coverage (report 22) | **+0.005** |
| lossless PNG bytes/pixel, as a complexity proxy | −0.528 |

The first is noise. The second is weak, and it points the **wrong way** — more complex images do *better*, not worse. Worse for that theory: **15 of the 24 Kodak images are *less* complex than Sierra by that measure** (Sierra 1.661 bpp against a Kodak mean of 1.631) and every one of them still loses by 19–72%.

**So there is no characterisation of when this format is competitive.** Sierra is an outlier for reasons not identified here.

## What this does to the record

**Report 22's mechanism is retracted.** It said parity "holds where a hundred regions explain most of the picture and fails where they explain about forty percent, and that figure is measurable before encoding." Measured across 24 images, that figure explains nothing. Logged as falsification #13.

**The compression verdict returns close to where the study began.** On standard photographic content the shape coder is 3.6–71.5% larger than a properly configured WebP at matched fidelity, with real files on both sides. The +0.93% headline is one unusual image.

**What still stands, and it is not nothing:**

- **Every coder improvement.** The cross-plane interleave, the cross-channel transform, the legality repair — all content-independent mechanisms measured on identical partitions. They are why the deficit is +27.8% rather than the +60%-ish the pre-fix coder would have shown.
- **The container.** SHPC v1 round-trips bit-exactly on all 24 images it had never seen.
- **The capability claims**, which never depended on bytes. kodim23 still yields 4,387 addressable regions that WebP cannot deliver at any file size — the format now costs 71.5% more to provide them on that image.

**Still unmeasured:** report 19's WebP+sidecar comparison on Kodak content. That margin is a ratio and could survive where parity did not — a raster+sidecar consumer pays for the region map too. It is the one remaining place the positioning could hold, and it has not been run (P5c).

## What I would tell someone starting this

Twenty-two reports of coder work were conducted on a single image that turns out to be unrepresentative, and the tuning was real — the coder genuinely improved, by mechanisms that transfer. But **no amount of coder work was ever going to be tested by that image**, and one afternoon on a standard corpus would have said so at any point in the previous two days.
