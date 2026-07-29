# 05 — The low-rate crossover: the one place shapes beat WebP

**Question.** Report 04 swept the frontier down to 256 regions and stopped, because the fixed eval sat at 28.6 dB. That left the bottom of the rate curve unmeasured — and the bottom is exactly where WebP is weakest. Does the picture change there?

**It does.** This is the only measured case in the whole investigation of a shape representation beating a shipping codec on bytes at matched fidelity.

## Method

- Potts+Ising frontier extended from 8,000 down to 153 regions ([`04-region-merge-frontier-data.txt`](04-region-merge-frontier-data.txt)). The merge floors out around 150 regions on this image; coarser marks produce nothing new.
- `cwebp` swept q0–q50 and `avifenc` q0–q38, in one run, on the same 512×288 image, measured with the project's own RGB PSNR ([`05-codec-rd-sweep-data.txt`](05-codec-rd-sweep-data.txt)).
- Comparison computed by script (`code/compare.py`), interpolating each codec's rate curve at every PSNR the shape coder actually hits, with no extrapolation outside the measured range. **An earlier draft of this table was interpolated by eye and overstated the low-rate win by more than 3×** — see report 06.

**Container accounting.** WebP bytes are whole files; its RIFF floor is ~44 B and is left in. AVIF's container floor is **297 B** (measured: an 8×8 flat image encodes to 297 B), which is 10–14% of a file at this rate, so the AVIF column is payload with 280 B discounted. Without that discount the bottom rows would read as a shape win that is pure box overhead.

## Result

| PSNR | regions | shape coder | WebP | vs WebP | AVIF payload | vs AVIF |
|---|---|---|---|---|---|---|
| 24.03 | 153 | **2,986** | 3,069 | **−2.7%** | 2,755 | +8.4% |
| 24.52 | 202 | **3,465** | 3,620 | **−4.3%** | 3,140 | +10.4% |
| 25.10 | 288 | **4,295** | 4,429 | **−3.0%** | 3,682 | +16.6% |
| 26.09 | 546 | **6,156** | 6,221 | **−1.0%** | 4,710 | +30.7% |
| 26.44 | 615 | **6,488** | 6,919 | **−6.2%** | 5,145 | +26.1% |
| 27.10 | 840 | **7,870** | 8,379 | **−6.1%** | 6,009 | +31.0% |
| 27.60 | 1,080 | **9,124** | 9,585 | **−4.8%** | 6,778 | +34.6% |
| 28.12 | 1,317 | **10,543** | 10,874 | **−3.0%** | 7,636 | +38.1% |
| 28.66 | 1,685 | **12,202** | 12,438 | **−1.9%** | 8,547 | +42.8% |
| 29.17 | 2,131 | 13,990 | 13,901 | +0.6% | 9,519 | +47.0% |
| 29.70 | 2,660 | 16,091 | 15,450 | +4.2% | 10,622 | +51.5% |

Negative means the shape coder is smaller.

## What it says

**The crossover is at ~29.2 dB (≈0.75 bpp).** Below it the shape coder beats WebP by **1–6%**, best around 26.4–27.1 dB. Above it WebP pulls ahead and the gap widens with rate.

**The mechanism is the isoperimetric argument of report 04 read in the other direction.** Boundary cost grows as √(region count), while WebP keeps paying per-block mode and coefficient bits that do not shrink as fast under deep quantization. Draw fewer, bigger regions and the perimeter tax stays small; chase fidelity and the region count explodes and the tax takes over. Report 04's third mechanism ("shape fraction tends to 1 as R tends to 0") is correct about the numerator and silent about the denominator — the raster's own floor also fails to shrink, which is why a crossover exists at all.

**AVIF is unaffected.** It leads by 8–52% across the whole band and never loses. There is no rate on this image where the shape coder beats AVIF.

## Caveats, load-bearing

- The shape number is an **adaptive-arithmetic cross-entropy estimate** — no container, no header, no real bitstream — so it is optimistic by whatever a real format would cost. A 1–6% win is well inside the margin a real container could erase.
- **PSNR only.** At 24–26 dB the failure modes are not comparable: the shape render posterizes into flat regions, WebP goes blurry and blocky. Report 04 already measured flat interiors losing badly on smooth content by MS-SSIM (0.902 vs AVIF's 0.983 on sky), so a perceptual sweep could plausibly reverse this table.
- **One image.** No Kodak, no CLIC, no BD-rate.

None of that is fixed by more effort on the encoder. It is fixed by building the container and running a multi-image perceptual sweep.

## What this changes, and what it does not

It changes "parity is the ceiling" to **"parity is the ceiling above ~29.2 dB; below it there is a real 1–6% win over WebP."**

It does not change the verdict. A 1–6% win over the *second*-best codec, in a band where the best codec leads by 8–52%, is not a reason for anyone to adopt a format — particularly when the win sits inside the uncertainty introduced by not having a real bitstream.

What it does do is locate the byte win — 0.16–0.66 bpp, 24–28.7 dB — in the same place as every non-byte argument: thumbnails, previews, LQIP placeholders, e-ink and low-power displays, bandwidth-starved links. At that rate the geometry is also cheap enough to keep its other properties, which is the actual reason to care.
