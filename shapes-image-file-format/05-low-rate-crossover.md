# 05 — The low-rate crossover: the one place shapes appeared to beat WebP

> **Corrected twice, and the second correction removes the result.** This report originally claimed a 3–9% win below 28.4 dB, was corrected to 1–6% below 29.2 dB after a nondeterminism was found (report 06 #6), and is corrected again below: the comparison ran `cwebp` at its **default** method, not the setting anyone shipping a file would use. At `-m 6` the win collapses to a wash that alternates sign between adjacent samples, and the crossover falls to ~26.1 dB. Report 08 then shows even that disappears as resolution rises. The original table is kept in place, because how a result dies is the useful part.

**Question.** Report 04 swept the frontier down to 256 regions and stopped, because the fixed eval sat at 28.6 dB. That left the bottom of the rate curve unmeasured — and the bottom is exactly where WebP is weakest. Does the picture change there?

**It looked like it did.** For a while this was the only measured case in the whole investigation of a shape representation beating a shipping codec on bytes at matched fidelity. It no longer survives.

## Method

- Potts+Ising frontier extended from 8,000 down to 153 regions ([`04-region-merge-frontier-data.txt`](04-region-merge-frontier-data.txt)). The merge floors out around 150 regions on this image; coarser marks produce nothing new.
- `cwebp` swept q0–q50 and `avifenc` q0–q38, in one run, on the same 512×288 image, measured with the project's own RGB PSNR ([`05-codec-rd-sweep-data.txt`](05-codec-rd-sweep-data.txt)).
- Comparison computed by script (`code/compare.py`), interpolating each codec's rate curve at every PSNR the shape coder actually hits, with no extrapolation outside the measured range. **An earlier draft of this table was interpolated by eye and overstated the low-rate win by more than 3×** — see report 06.

**Container accounting.** WebP bytes are whole files; its RIFF floor is ~44 B and is left in. AVIF's container floor is **297 B** (measured: an 8×8 flat image encodes to 297 B), which is 10–14% of a file at this rate, so the AVIF column is payload with 280 B discounted. Without that discount the bottom rows would read as a shape win that is pure box overhead.

## Result as first published — `cwebp` at its default method

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

Negative means the shape coder is smaller. **This table is wrong in the same way report 06's other entries are wrong: the baseline was not the strongest form of the thing being beaten.**

## The same table with the codecs configured properly

`cwebp`'s default is method 4. `-m 6` is slower to encode, identical to decode, and what any build pipeline emitting WebP would use. On this image it is **5.6–8.9% smaller** — larger than the entire effect being claimed. `avifenc -s 4` is likewise not AVIF's best; `-s 0` is.

| PSNR | regions | shape coder | WebP `-m 6` | vs WebP | AVIF `-s 0` payload | vs AVIF |
|---|---|---|---|---|---|---|
| 24.03 | 153 | **2,986** | 3,049 | **−2.1%** | — | — |
| 24.52 | 202 | **3,465** | 3,598 | **−3.7%** | 3,023 | +14.6% |
| 25.10 | 288 | **4,295** | 4,334 | **−0.9%** | 3,548 | +21.1% |
| 26.09 | 546 | 6,156 | 6,058 | +1.6% | 4,556 | +35.1% |
| 26.44 | 615 | **6,488** | 6,703 | **−3.2%** | 4,972 | +30.5% |
| 27.10 | 840 | **7,870** | 8,079 | **−2.6%** | 5,843 | +34.7% |
| 27.60 | 1,080 | **9,124** | 9,218 | **−1.0%** | 6,567 | +38.9% |
| 28.12 | 1,317 | 10,543 | 10,394 | +1.4% | 7,382 | +42.8% |
| 28.66 | 1,685 | 12,202 | 11,885 | +2.7% | 8,319 | +46.7% |
| 29.17 | 2,131 | 13,990 | 13,215 | +5.9% | 9,232 | +51.5% |

Data: [`05-codec-rd-sweep-strong-data.txt`](05-codec-rd-sweep-strong-data.txt). Both tables are printed by `code/compare.py`, which reads both sweeps rather than quietly replacing one with the other.

## What it says now

**There is no crossover worth the name.** Below ~28 dB the two curves sit within ±4% of each other and **the sign alternates between adjacent sample points** — −2.1, −3.7, −0.9, +1.6, −3.2, −2.6, −1.0, +1.4. That is two curves lying on top of one another sampled at slightly different places, not a coder that wins. Above ~28 dB WebP is ahead and stays ahead.

**And the shape coder is still the flattered side.** Its number is an adaptive-arithmetic cross-entropy with no container, no header, and no actual bitstream. WebP's is a file you could serve. Closing that gap moves the shape column up, never down.

**AVIF is untouched by any of this.** It leads by 15–52% across the whole band and never loses. There is no rate on this image where a shape representation beats AVIF.

**The mechanism that predicted a crossover was not wrong — it was just too small.** Boundary cost grows as √(region count) while WebP keeps paying per-block mode and coefficient bits that do not shrink as fast under deep quantization, so the two curves *do* converge as rate falls. They converge to a tie, and the tie is worth a few percent in either direction depending on where you sample. Report 04's third mechanism ("shape fraction tends to 1 as R tends to 0") is correct about the numerator and silent about the denominator; the raster's floor also fails to shrink, which is why the curves meet at all.

## Caveats, load-bearing

- The shape number is an **adaptive-arithmetic cross-entropy estimate** — no container, no header, no real bitstream — so it is optimistic by whatever a real format would cost. Any win of a few percent is well inside the margin a real container would erase.
- **PSNR only.** At 24–26 dB the failure modes are not comparable: the shape render posterizes into flat regions, WebP goes blurry and blocky. Report 04 already measured flat interiors losing badly on smooth content by MS-SSIM (0.902 vs AVIF's 0.983 on sky), and report 08 shows the same split at 4K — the shape coder is a near-tie on texture and 5.4 dB behind on a gradient. A perceptual sweep could move this table in either direction.
- **One image, one size.** No Kodak, no CLIC, no BD-rate. Report 08 varies the size and the remaining wash gets worse.

None of that is fixed by more effort on the encoder.

## What this changes

It changed "parity is the ceiling" to "parity is the ceiling above ~29.2 dB; below it there is a real 1–6% win" — and then back again. **Parity, at the very bottom of the rate range, on one image at one size, against an idealised bitstream, is the ceiling.**

It never changed the verdict. What it does do is locate the *only* band where the geometry is even competitive — 0.16–0.66 bpp, 24–28 dB — in the same place as every non-byte argument: thumbnails, previews, LQIP placeholders, e-ink and low-power displays, bandwidth-starved links. At that rate the geometry is cheap enough to keep its other properties, which was always the actual reason to care.
