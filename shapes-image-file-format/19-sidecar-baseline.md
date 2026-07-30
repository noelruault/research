# 19 — Against the baseline that matches the product, it wins by 40–44%

**Question (P1).** Report 13 argued the honest baseline is not WebP but **WebP plus a region-map sidecar**, because that is what someone would have to assemble to get the same capabilities. It put the shape coder 41% ahead — but priced the sidecar with *our own* wall coder, which report 06 #12 showed is not even decodable. Steelman the sidecar and measure.

**Answer.** The hypothesis holds. **40–44% smaller against the strongest sidecar available, 60–66% against what a consumer could actually assemble.** And the sidecar steelman turned out to be our own coder, by a factor of 2.3–2.5.

Knobs, stated because falsification #11 has been committed twice in this study: WebP gets `-m 6` at matched fidelity; the sidecar gets the best of five lossless encodings; the shape coder gets report 09's interleaved walls and report 15's RCT colour. Both sides deliver the same partition and the same pixel fidelity.

## The sidecar is the interesting part

The label map for the 11,121-region partition at 3840×2160, encoded five ways:

| encoding | bytes | vs ours |
|---|---|---|
| 16-bit grey PNG | 597,572 | 5.65× |
| boundary bitplane, one byte per pixel + `xz -9e` | 364,296 | 3.44× |
| raw 16-bit label plane + `brotli -q11` | 276,958 | 2.62× |
| raw 16-bit label plane + `xz -9e` | **259,624** | 2.46× |
| horizontal-delta label plane + `xz -9e` | 332,832 | 3.15× |
| **our interleaved crack coder** | **105,752** | — |

**Our boundary coder beats the best off-the-shelf option by 2.46×**, and by 2.25× at the capability point (16,700 B against 37,504 B). So pricing the sidecar with our own coder is not self-dealing — it is the steelman. Anyone assembling this from parts does substantially worse, which makes the comparison below *conservative*.

Note what fails: coding the *label ids* beats coding a *boundary bitplane* under a general compressor (259,624 vs 364,296), because a general compressor exploits the long runs of constant id along a row. But a purpose-built context coder on the crack edges beats both, because the boundary is where the information actually is.

## The comparison

| at 28.5 dB, 11,121 regions, 4K native | bytes | |
|---|---|---|
| **shape coder** | **132,280 B** | walls 105,752 + colour 26,528 |
| WebP + sidecar, our coder (steelman) | 236,834 B | **shapes 44.1% smaller** |
| WebP + sidecar, best off-the-shelf | 390,706 B | shapes 66.1% smaller |

| at ~24.98 dB, 3,546 regions, 960→4K | bytes | |
|---|---|---|
| **shape coder** | **25,399 B** | walls 16,700 + colour 8,699 |
| WebP + sidecar, our coder (steelman) | 42,400 B | **shapes 40.1% smaller** |
| WebP + sidecar, best off-the-shelf | 63,204 B | shapes 59.8% smaller |

The mechanism is the one report 13 named: raster-plus-sidecar pays for the boundaries **and** every pixel. The shape coder pays for the boundaries and derives the pixels from them. Geometry stops being overhead and becomes load-bearing.

## And the sidecar delivers *worse* structure at that price

This follows from construction rather than measurement, and it strengthens the comparison. In raster-plus-sidecar the mask and the pixels are produced by different coders: WebP's lossy pixels bleed across the boundaries the mask declares, so a region's interior is not constant and its edge does not align with the mask's edge. "Select this region and recolour it" leaves fringing.

In the shape coder the pixels **are** the regions — piecewise constant by construction, so mask and pixels cannot disagree. Same capability, but the sidecar version is an approximation of it at 1.7–2.9× the bytes.

## Caveats

- **Our 105,752 B and 16,700 B are idealised cross-entropies** with no container, while every generic number is a real file. The margin is 2.25–2.46×, far wider than any plausible container overhead, so the conclusion survives — but the exact ratio will shrink when report 13's P4 is built.
- **No dedicated modern region-map codec was benchmarked.** JBIG2 and CM-class coders are not installed here. Our coder is the MPEG-4 CAE lineage, which is the right family, but "nothing off-the-shelf beats it" rests on five encodings, not a survey. **Unverified.**
- **One photograph, and report 22 makes this urgent.** Parity with WebP alone collapses on Kodak content (+9.4% to +71.5%). This margin is a *ratio* and could hold, narrow or invert there — the region map gets dearer for both sides. **It has not been re-run on Kodak and 40–44% must not be quoted as general until it has.**
- **The comparison assumes the consumer wants this exact partition.** If a coarser mask would do, their sidecar gets cheaper — and so does ours, since both scale with region count.

## What this settles

Report 13's positioning is no longer a hypothesis. **For the product this format actually is — an image that carries its own segmentation — it is 40–44% cheaper than assembling the same thing from a raster codec and a mask, and the gap widens if the consumer uses tools they can actually download.**

Combined with reports 16 and 17: against WebP *alone* the format is at parity (+0.91% at 28.5 dB, −1.2% at the capability point) while delivering something WebP cannot. Against WebP *plus the missing piece*, it is 40–44% ahead.
