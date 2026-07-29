# 14 — The regions are meaningful, and the capability point is not the byte point

**Question (P0 from report 13).** Twelve reports measured whether the regions are *cheap*. None measured whether they are *right*. Every capability claim in report 13 — background removal, subject selection, per-region editing — depends on regions landing on things a person would name. Report 04 found flat interiors banding badly on smooth sky, which suggested the merge might be following illumination rather than objects. So: do the regions follow objects, or light?

**Answer.** They follow **tonal boundaries**, which on this image means they follow objects wherever tone and object coincide, and follow cloud structure in the sky. The pessimistic reading was wrong. But the measurement surfaced something more useful: **the operating point where the segmentation is most useful is not the operating point where the bytes are most competitive.**

Method: a published render is the partition painted with each region's own mean colour, so 4-connected runs of one colour *are* the regions. Recovered them and counted how many cover three 448×448 windows whose content we can name. Code: [`14-segmentation/segq.go`](14-segmentation/segq.go), stdlib only. Data: [`14-segmentation-data.txt`](14-segmentation-data.txt).

## The measurement

| regions | PSNR | bytes | **sky** | ridge | snow | top-100 regions cover | median region | largest |
|---|---|---|---|---|---|---|---|---|
| 227 | 21.99 dB | 19,819 B | **2** | 21 | 19 | **99.4%** | 1,716 px | 12.5% of image |
| 1,383 | 24.99 dB | 50,016 B | **2** | 84 | 76 | **90.0%** | 223 px | 9.3% of image |
| 11,121 | 28.51 dB | 153,190 B | 11 | 633 | 452 | 65.2% | 34 px | 3.6% of image |

**The distribution is heavy-tailed at every level**, which is the signature of an object-like partition rather than uniform banding. At 227 regions a hundred regions describe **99.4%** of a 4K photograph. At 1,383 they still describe 90%. The largest region is 3,450× the median at 1,383 regions — a partition of similar-sized cells would be near 1×.

## What it looks like

**Sky at 1,383 regions** ([`1383-sky.png`](14-segmentation/1383-sky.png)) — two regions, split along a brightness contour that traces the cloud edge. Not an object boundary in the strict sense, but not arbitrary banding either, and **"select the sky" is a union of two ids**.

**Ridge at 1,383 regions** ([`1383-ridge.png`](14-segmentation/1383-ridge.png)) — the shadowed rock face is one large region, the lit snow another, the sunlit peak a third. Boundaries follow the ridgeline and the shadow terminator. This is the case the capability pitch needs and it works.

**Sky at 11,121 regions** ([`11121-sky.png`](14-segmentation/11121-sky.png)) — eleven regions now, and they trace **cloud form**: the bands follow tonal contours that are the actual structure of the sky, not horizontal slices. More fragmented, still not arbitrary.

## The finding that matters more than the answer

The two things this format is being optimised for want **different operating points**:

- **Capability is best at 227–1,383 regions** — sky is 2 regions, a hundred regions cover 90–99% of the image, median region is 223–1,716 px. This is **21.99–24.99 dB**, which is visibly posterised.
- **Bytes are most competitive at ~11,121 regions** — that is where the interleave put us at +8.3% over WebP, heading to ~+4.7% with RCT. But there the sky is 11 regions, one 448×448 window holds **633** of them, and the median region is **34 px** — texture speckle, not objects.

**Nine reports optimised the byte point. Report 13's applications all live at the capability point.** Nobody has ever measured the byte competitiveness *at 1,383 regions against WebP at matched fidelity*, because 24.99 dB was assumed to be below anything worth shipping. For a format whose selling point is structure rather than fidelity, that assumption deserves testing.

## Honest limits

- **Three windows on one photograph.** The window contents were named by me from a render, not from an annotated ground truth. There is no boundary-recall number against a human segmentation or against SAM, which is what a real answer requires. This is a strong indicator, not a proof. **B7 stands.**
- **"Meaningful" is doing work here.** Regions follow *tone*. On a landscape, tone and object largely coincide. On a scene with a patterned object, or a shadow falling across two different materials, they will not — and this measurement cannot tell you which.
- **The sky bands are still illumination, not objects.** Report 04's observation was not wrong, it was incomplete: the sky *is* split by brightness, but into 2 regions at the capability point rather than a dozen, which is a usable failure.
- Region counts are recovered from renders; above ~34k regions adjacent regions can share a rounded mean and fuse. All three marks here are far below that.

## What this changes

W1 in report 13 — "nobody has checked whether the regions are semantically meaningful" — is **answered positively enough to keep building on**, with the caveat that a proper boundary-recall measurement against annotated ground truth has not been done.

The new priority it creates: **measure byte competitiveness at the capability operating point (1,383 regions, 24.99 dB), not just at 28.5 dB.** If the format is to be sold on structure, the honest benchmark is at the rate where the structure is good — and that number does not exist in nine reports of measurement.
