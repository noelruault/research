# 08 — Native resolution, and the lossless end

**See it:** [1:1 mirror at 4K](https://claude.ai/code/artifact/4ca0a875-4377-457a-8ace-9e83deaa896f) — drag-to-wipe comparison at true native pixels, plus the ladder chart. Local copy: [`hd-mirror.html`](hd-mirror.html).

**Question.** Every round of this investigation ran on one image at 512×288. Fixing the eval is what made the rounds comparable — and it froze resolution, so nothing measured could say whether the results were about shapes or about small pictures. This round varies the frozen axis, and adds the one operating point never tested: **lossless**, where WebP is bit-exact and there is nothing to trade.

**Answer.** Both questions come out against the shape coder, and the second one badly.

## What was run

The same photograph as every earlier round — the macOS Sierra wallpaper — at its native **3840×2160**, plus resamples to 1920×1080, 960×540 and 512×288 so all four rungs hold content constant and vary only size.

Pricing is **identical to report 04's `frontier`**: the rate-distortion merge, six sweeps of Ising wall relaxation, the cheaper of the CAE and contour wall coders, and `colorBytes2` for colours. This matters — a first pass used merge-only with the weaker single-neighbour colour predictor and came out 31% worse, which would have measured a coder this study never published. The agreement check: on the original eval image the new path interpolates to 12,341 B at 28.66 dB against the published 12,202 B, a 1.1% difference explained by different scale-space mark spacing.

WebP is `cwebp -m 6`, AVIF is `avifenc -s 6`, JPEG XL is `cjxl -e 7`. Per report 06 #9, leaving a codec on its defaults is a way of losing an argument you have already won.

Data: [`08-native-resolution-data.txt`](08-native-resolution-data.txt), [`08-resolution-ladder-data.txt`](08-resolution-ladder-data.txt). Code: `code/lab/hd.go`, `code/hd/`.

## Result 1 — lossless: the shape coder ties PNG and loses to everything since

To reproduce all 8,294,400 pixels exactly, the region coder must transmit the exact partition: every 4-connected run of identical pixels as its own region.

| encoder | setting | bytes | vs WebP |
|---|---|---|---|
| JPEG XL | `cjxl -d 0 -e 7` | 6,468,598 | 0.84× |
| **WebP** | `cwebp -lossless -z 9` | **7,718,506** | — |
| AVIF | `avifenc --lossless` | 11,969,137 | 1.55× |
| **shape coder** | exact region partition | **12,159,385** | **1.58×** |
| PNG | as delivered | 12,278,280 | 1.59× |

**1.58× WebP, 1.88× JPEG XL, and a dead heat with PNG.** And the shape coder is the flattered side even here: its figure is contour + colour cross-entropy with no container and no header, against real files.

The mechanism is visible in the breakdown, and it is not the one report 04 predicted:

| | |
|---|---|
| distinct colours | 529,557 |
| regions in the exact partition | 6,356,392 |
| pixels per region | **1.305** |
| crack edges | 14,508,938 |
| **walls** | 822,369 B — 0.4534 bits/edge |
| **colours** | 11,337,015 B |

**The geometry is nearly free and irrelevant.** When almost every pixel boundary is a wall, the wall map is nearly constant and codes at under half a bit per edge — 803 KB for fourteen and a half million edges. **93% of the bill is colour.** The partition has degenerated to 1.3 pixels per region, at which point the "region coder" *is* a raster coder: one colour per pixel, predicted from a single neighbour, where WebP brings fourteen spatial predictors, a cross-channel colour transform, LZ77 with 2D distance mapping, a colour cache, and a per-tile meta-Huffman segmentation it gets for free.

This is worth stating plainly because it is the cleanest statement of the whole investigation's finding: **at the exact end there is no geometry to exploit, so an explicit-geometry format degenerates into a worse raster format.** The ratio is stable across sizes — 1.48× at 512×288, 1.41× at 960×540, 1.40× at 1920×1080, 1.58× at 3840×2160.

## Result 2 — the low-rate wash gets worse with resolution

Report 05, corrected twice, ends at "a wash below ~28 dB on this image at 512×288". Here is the same comparison at four sizes, read at fixed PSNR so the same picture quality is bought at every rung. Positive means the shape coder needs **more** bytes than WebP.

| | 512×288 | 960×540 | 1920×1080 | 3840×2160 |
|---|---|---|---|---|
| **28.7 dB** | +0.2% | +2.3% | +6.8% | **+19.3%** |
| **30.0 dB** | +6.5% | +8.9% | +13.4% | **+32.7%** |
| **31.5 dB** | +9.2% | +15.6% | +24.4% | **+48.6%** |
| **34.0 dB** | −1.8% | +18.8% | +39.5% | **+70.8%** |

Monotone at every fidelity. (The −1.8% at 512×288/34.0 dB is a single point at the top of that rung's measured WebP range, where the sampling is sparse; the three rungs above it are unambiguous.)

**This is the opposite of what the isoperimetric argument predicts.** Boundary length grows with the linear dimension and area with its square, so a 16× pixel count should cost only 4× more boundary — the perimeter tax that sinks the shape coder ought to *weaken* at 4K. It does weaken. The deficit still grows, because WebP's cost per pixel falls **faster**:

| at 28.7 dB | 512×288 | → 3840×2160 | growth |
|---|---|---|---|
| shape coder | 9,399 B | 163,471 B | ×17.4 |
| WebP | 9,382 B | 137,033 B | ×14.6 |

for a 56.25× increase in pixels. Both improve with resolution; WebP improves more. A larger picture gives a block codec more correlated neighbourhood to predict from, longer LZ77 matches, and more tiles to segment its entropy image over. It gives the shape coder a slightly better area-to-perimeter ratio, and nothing else. Region count at matched fidelity rises from 1,044 to 11,121 across the ladder — sublinear in pixels, which is the isoperimetric gain showing up exactly as predicted, and not enough.

## Result 3 — where the deficit actually lives

At a realistic web budget the comparison can be held constant two ways, and both are worth reading.

**Matched size.** Shape coder 203,511 B @ 29.44 dB against WebP `-q 14` 200,024 B @ 30.38 dB — near enough iso-byte, and WebP is 0.94 dB ahead. **Matched quality.** WebP `-q 8` reaches 29.52 dB, within 0.08 dB of the shape coder, in **165,042 B** — **18.9% fewer bytes** for the same measured result.

Measured on three 448×448 native windows instead of the whole frame:

| window | content | shapes | WebP, matched size | gap | WebP, matched quality | gap |
|---|---|---|---|---|---|---|
| Sunlit ridge | dense texture, hard shadow terminator | 26.01 dB | 26.44 dB | −0.43 dB | 25.57 dB | **+0.44 dB** |
| Snow and rock | flat fields split by sharp dark edges | 26.82 dB | 27.39 dB | −0.57 dB | 26.61 dB | **+0.21 dB** |
| **Sky** | **smooth gradient, film grain** | **36.90 dB** | **42.29 dB** | **−5.39 dB** | 41.61 dB | **−4.71 dB** |

The matched-quality column is the sharper statement of the same fact: **spending 19% fewer bytes, WebP is still 4.7 dB ahead on the gradient and slightly *behind* on both texture windows.** The shape coder is not generally worse. It is competitive on exactly the content a region model is for, and it loses the whole contest in the sky.

**Almost the entire deficit is one content type.** On texture and on sharp-edged structure — the cases a region model is built for — the shape coder is within 0.6 dB of WebP. On a smooth gradient it is 5.4 dB behind, because a piecewise-*constant* model has exactly one way to draw a ramp: stack flat bands and pay for every boundary between them. A DCT spends one low-frequency coefficient.

That failure is nameable and therefore looks fixable, which is why report 04 already priced the fix: per-region **affine** colour closes most of the gradient gap and costs more in coefficients than it saves. The bands are the symptom. The explicit boundary is the disease.

## Caveats, load-bearing

- **The source is a q95 JPEG.** At native resolution it carries JPEG artefacts and film grain that a downscale averages away, and noise favours a transform codec (which buries it in high-frequency residual) over a region codec (which must spend a region on every blob). This confounds "resolution" with "noise". The ladder controls it partially — all four rungs come from that same 4K file, so the effect is monotone across rungs that share content and share the noise's origin — but a genuinely grain-free 4K master could behave differently. **Not verified.**
- **One image.** No Kodak, no CLIC, no BD-rate. Everything here is one photograph with sky, rock, snow and grain.
- **PSNR only**, and result 3 is precisely a case where PSNR and an eye would disagree about *which* failure is worse: banding in a sky is more visible than its dB cost implies.
- **The shape coder still has no container.** Every number above is an idealised cross-entropy. Building a real bitstream moves the shape column up, never down.

## What this changes

Report 05's crossover is gone twice over: once because WebP was configured below its shipping settings (report 06 #9), and again because whatever survived that was a small-image effect. **There is no measured regime on this image — no rate, no resolution, not lossless — where a shape representation is reliably smaller than a well-configured WebP.**

What survives is narrower and unchanged by any of this: at 24–28 dB the two representations cost about the same, and in that band the geometry is cheap enough to keep the properties that were never about bytes — editability, restyling, resolution independence where the geometry is authored above the pixel grid, and semantic addressability. That is the same conclusion report 04 reached from four directions. This round removes the one number that had been arguing otherwise.
