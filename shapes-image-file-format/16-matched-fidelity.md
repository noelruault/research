# 16 — At matched fidelity the gap is 0.9%

**Question (P2b).** Report 15 left the shape coder at 132,280 B for the 11,121-region mark, every component decodable. `cwebp -m 6` is 137,033 B at 28.7 dB — but this mark is at 28.51 dB, and comparing those two is falsification #1's mismatched-fidelity error. So: what does WebP cost at *this* fidelity?

**Answer.** 131,082 B. **The shape coder is 0.91% larger.**

## The comparison

Both sides encode at native 3840×2160 — native versus native, matched on PSNR rather than on bytes.

| | bytes | PSNR | |
|---|---|---|---|
| `cwebp -m 6 -q 3` | **131,082 B** | **28.52 dB** | a real file |
| **shape coder** | **132,280 B** | **28.51 dB** | walls 105,752 + colour 26,528 |
| | **+1,198 B** | | **+0.91%** |

WebP is 0.01 dB better *and* 1,198 B smaller, so it still wins — by nine tenths of one percent.

Search: `q20` 233,530 B / 31.06 dB → `q10` 177,370 / 29.82 → `q5` 145,842 / 28.99 → `q2` 121,542 / 28.19 → **`q3` 131,082 / 28.52**. The target is bracketed between q2 (28.19 dB, below) and q3 (28.52 dB, above), and q3 is the first setting that meets the shape coder's fidelity.

## Where the 19.3% went

| | walls | colour | total | vs WebP at this fidelity |
|---|---|---|---|---|
| as published (report 08) | 121,047 | 32,143 | 153,190 B | **+16.9%** |
| + cross-plane interleave (report 09) | 105,752 | 32,143 | 137,895 B | +5.2% |
| + cross-channel transform (report 15) | 105,752 | **26,528** | **132,280 B** | **+0.91%** |

Two changes, neither of which touched the partition, the fidelity, or the region count. One reordered which of two crack planes is coded first. The other applied a colour transform WebP has had since 2010.

**And the published figure was never legal.** Report 06 #12 showed the published wall coster reads a context bit no decoder can supply. The 132,280 B above uses the interleaved coder, which report 09's decoder replay confirmed is causal, and a colour stream verified by rebuilding the image from the partition and the bytes alone — 0 wrong samples of 24,883,200.

## What is still flattered, and by roughly how much

**The wall half is still an idealised cross-entropy.** 105,752 B is what an adaptive binary arithmetic coder would emit with no container, no header, no framing. The colour half (26,528 B) is a real `brotli` stream. WebP's 131,082 B is a real file including its ~44 B container.

So the honest reading is: **the shape coder is within 1% of WebP at matched fidelity, and roughly that much of the remaining difference is container overhead it does not yet pay.** Parity is plausible and unproven. Building the container (report 13, P4) is what would settle it, and it can only move the shape number **up**.

Also unchanged: **one photograph** (B7), and **AVIF remains 30–50% ahead** at every fidelity measured — it is not a target and this does not make it one.

## What this changes

The README's headline — *"the deficit grows monotonically with resolution, reaching +19.3% at the eval's own fidelity"* — described a coder with an illegal wall coster and no cross-channel transform. Re-measured at the same fidelity with both fixed, it is **+0.91%**.

The compression verdict does not become "shapes win". It becomes something more useful: **a shape representation costs about the same as WebP at the same fidelity, while carrying a segmentation WebP cannot carry at any price.** Report 13 argued that was the claim worth pursuing. This is the number that makes it true — at one operating point, on one image, with the container still unbuilt.

The next honest step is not another byte lever. It is P0b — the same measurement at **1,383 regions**, where report 14 found the segmentation is actually good, because that is the operating point the applications want and nobody has ever benchmarked it.
