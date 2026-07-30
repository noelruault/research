# 17 — At the operating point the applications want, it is a dead heat

**Question (P0b).** Report 16 measured near-parity at 11,121 regions / 28.51 dB. Report 14 found that is *not* where the segmentation is good — at 11,121 the median region is 34 px of texture speckle, while at 227–1,383 regions the sky is 2 regions and a hundred regions describe 90–99% of the image. Nine reports optimised a rate the applications do not want. **What do the bytes look like at the rate they do?**

**Answer.** A dead heat, once both coders are given the same knob. And the first attempt at this measurement made the study's own falsification #11 error, in the same session it was documented.

## The wrong answer first, and why it was wrong

The capability point lands at **~48 KB**, which is *below* `cwebp`'s native floor of 85,102 B at 3840×2160. So WebP cannot reach it by turning quality down — it has to encode small and upscale. Running that:

| at 24.99 dB, 4K output | bytes |
|---|---|
| WebP, best of a resolution×quality search (960×540, `q28`) | 25,700 B |
| shape coder at **native 4K**, 1,383 regions | 48,297 B |
| | **+87.9%** |

**That comparison is invalid.** It gives WebP the resolution knob and denies it to the shape coder — precisely falsification #11, which this study recorded hours earlier. The +87.9% is an artifact of comparing a native-resolution shape file against a resampled WebP.

## The symmetric measurement

Both coders encode at 960×540 and upscale to 3840×2160 with the identical `sips` call, both scored against the same 4K original.

| at ~24.98 dB, 4K output | bytes | PSNR |
|---|---|---|
| `cwebp -m 6 -q 28 -resize 960 540` | **25,700 B** | 24.99 dB |
| **shape coder, 3,546 regions** | **25,399 B** | 24.97 dB |
| | **−1.2%** | −0.02 dB |

The shape coder is **1.2% smaller at 0.02 dB lower fidelity**. That is a dead heat, and it is the closest this format has ever come to a modern codec on bytes.

**Composition of the shape number**, both halves measured today and both decodable:

| | published | today | source |
|---|---|---|---|
| walls | 17,346 B | **16,700 B** | cross-plane interleave (report 09), `wallxexact` |
| colour | 10,312 B | **8,699 B** | RCT + brotli + 8 B coefficients (report 15) |
| **total** | **27,658 B** | **25,399 B** | **−8.2%** |

`colorBytes2` reproduced the published 10,312 B exactly on the recovered partition before anything was changed, and the interleave's base column reproduced the published 17,346 B wall figure.

## What this means

**The format is at parity with WebP at the operating point where its structure is actually useful**, and it delivers 3,546 addressable regions — a region adjacency graph, stable ids, per-region editability — that WebP cannot deliver at any file size.

Report 13 argued the requirement was never "smaller than WebP" but "not meaningfully larger, while carrying something WebP cannot carry". At 28.51 dB that is +0.91% (report 16). At the capability point it is **−1.2%**.

## Load-bearing caveats

- **~~The wall half is still an idealised cross-entropy~~ — SETTLED, report 21.** The container was built and costs **19 bytes**. The real file is **25,418 B** against WebP's 25,700 B: **−1.097%**, round-tripping bit-exactly. The estimate below was honest to within tens of bytes.
- **One photograph — and report 22 has now shown it matters.** On three Kodak images the same pipeline runs +9.4% to +71.5% over WebP. This −1.097% does not generalise.
- **Region count versus resolution.** 3,546 regions at 960×540 is 146 px/region. Report 14's segmentation-quality measurements were taken at 4K, where 1,383 regions is 6,000 px/region. The two regimes are not identical, and **whether the segmentation is as good at this specific mark has not been measured** — only that region counts in this range behave well at 4K.
- PSNR only, and upscaling makes both sides blurrier in ways PSNR treats generously.

## What this changes in the roadmap

P0b is answered, and it moves the format from "close at a rate nobody wants" to "at parity at the rate the applications want". The next items are unchanged in order but now carry more weight, because a headline rests on them: **P4, the container** — without it the parity claim cannot be closed — and **P5, a second image**, because every number above is one photograph.
