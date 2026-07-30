# 22 — The parity result does not generalise

**Question (P5).** Every number in twenty-one reports comes from one photograph: the macOS Sierra wallpaper. That single image now gates three headlines — parity with WebP, the 40–44% sidecar margin, and the segmentation-quality result. Run the whole pipeline on genuinely different content and see whether the numbers hold.

**Answer.** They do not. **On three standard Kodak photographs the shape coder is 9.4% to 71.5% larger than WebP at matched fidelity**, against **+0.93%** on the Sierra image. The parity result was a property of that photograph, not of the format.

Knobs, stated: both coders encode at **native 768×512 with no resampling**, WebP at `-m 6`, fidelity matched by bisecting `cwebp -q` to the shape render's PSNR. The shape files are **real SHPC v1 containers**, not estimates.

## The measurement

| image | regions | PSNR | **SHPC file** | **WebP** | delta |
|---|---|---|---|---|---|
| kodim01 (brick wall) | 7,819 | 28.80 | 39,891 B | 33,680 B | **+18.4%** |
| kodim01 | 4,604 | 27.59 | 28,013 B | 25,604 B | **+9.4%** |
| kodim05 (street scene) | 7,503 | 27.84 | 42,498 B | 33,436 B | **+27.1%** |
| kodim05 | 4,457 | 26.65 | 30,950 B | 26,652 B | **+16.1%** |
| kodim23 (parrots) | 7,240 | 36.51 | 35,805 B | 21,896 B | **+63.5%** |
| kodim23 | 4,387 | 34.76 | 25,702 B | 14,986 B | **+71.5%** |
| *Sierra 4K, report 21* | *11,121* | *28.51* | *132,301 B* | *131,082 B* | ***+0.93%*** |

Re-checked by hand at the worst point: at 34.76 dB the shape coder needs 25,702 B, while `cwebp -q 60` reaches **35.68 dB in 18,158 B** — **smaller and better**. The +71.5% is if anything conservative, since it credits the shape coder with matching a fidelity WebP overshoots for 29% fewer bytes.

**The format itself is fine.** SHPC round-trips bit-exactly on content it has never seen — 0 wrong of 1,179,648 samples on kodim23, container overhead +20.67 B. The pipeline works; it is simply not competitive on these images.

## Why — and it was measurable in advance

Report 14's region-area tool, run on the new content:

| | top 100 regions cover | largest region | median region |
|---|---|---|---|
| **Sierra** @ 1,383 regions | **90.0%** of the image | 9.3% | 223 px |
| kodim01 @ 4,603 | **39.9%** | 5.0% | 17 px |
| kodim23 @ 4,387 | **42.6%** | 2.6% | 10 px |

**On the Sierra wallpaper a hundred regions describe 90% of the picture. On the Kodak images a hundred regions describe about 40%.** That wallpaper has an enormous smooth sky — a few regions cover an enormous area for almost nothing, and the boundary bill stays tiny relative to the pixels those regions explain. The Kodak photographs have no such area. A brick wall, a busy street and a parrot's plumage are detail everywhere, so region count rises, the perimeter tax rises with it, and the isoperimetric argument from report 04 reasserts itself exactly as it predicted.

This is the report 04 mechanism, unchanged: **geometry is cheap only when regions are few and large.** The Sierra image is unusually generous in that respect, and twenty-one reports of tuning were conducted on it.

## What survives and what does not

**Does not survive:**

- **Parity with WebP as a general claim.** Reports 16, 17 and 21 measured it on one image. It is real for that image and does not transfer. Every headline stating it must now carry the content dependence.

**Survives, unchanged:**

- **The capability claims.** Addressable regions, stable ids, the adjacency graph, O(regions) editing, bounded per-pixel error, no generational loss — none of these depend on byte parity. On kodim23 you still get 4,387 addressable regions that WebP cannot deliver at any size.
- **The container.** SHPC v1 round-trips bit-exactly on unseen content at ~20 B of overhead.
- **Every coder improvement.** The interleave, RCT, the legality fix and the re-pricing are all content-independent mechanisms measured on identical partitions. They made the coder better; they did not make the format universally competitive.

**Not measured, and not to be assumed either way:**

- **The sidecar comparison (report 19).** Its margin is a *ratio* — WebP plus a region map versus shapes — and on busy content the region map itself gets much more expensive for both sides. It could hold, narrow or invert. **It has not been run on Kodak** and nobody should quote 40–44% as general until it has.
- Whether the capability operating point behaves differently on this content, or whether resampling changes the picture as it did on Sierra.

> **The predictor in this section is retracted — report 23, falsification #13.** Across the full Kodak-24 set, top-100 region coverage correlates **+0.005** with the byte delta. It was fitted to three points. The content-dependence conclusion stands; the explanation for it does not.

## What this changes

The honest headline is no longer "at parity with WebP". It is: **on content with large low-detail areas the shape coder reaches parity; on detailed content it is 9–72% behind, and the gap tracks how much of the image a few regions can explain.**

That is a narrower claim and a more useful one, because it is predictive: the top-100 coverage figure is measurable *before* encoding and tells you which regime an image is in.

**P5 was the item most likely to break something, and it did.** The remaining roadmap work — parallelising the encoder, re-tuning the RD constants — is now clearly secondary to establishing where the format's regime boundary actually lies, which needs the full Kodak-24 set rather than three images.
