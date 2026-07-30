# 34 — Why WebP wins, decomposed, and what is actually left

**Question.** Report 33 showed WebP beating this coder by +24.8% to +100.9% on six photographs. "WebP is better" is not an answer. Where do the bytes go, what specifically does WebP do that we do not, and which levers remain?

**Answer.** **79.3% of our file is geometry, and our wall chunk alone is 29.6% larger than WebP's entire file.** Colour is not the problem and never was. Two candidate levers were revived and tested in writing this report; one is now closed negative and the other was already at its best setting. Closing the gap needs a **48.9% cut in the wall bill**, and nothing measured or parked is that size.

Data: [`34-why-webp-wins-data.txt`](34-why-webp-wins-data.txt).

## Where our bytes go

The dog photograph, 738×414, at 1,229 regions / 29.22 dB, as a real SHPC v2 file:

| component | bytes | share |
|---|---|---|
| **wall chunk** | **9,302** | **79.3%** |
| colour chunk | 2,395 | 20.4% |
| header | 24 | 0.2% |
| **total** | **11,725** | |

WebP at matched fidelity (`-m 6 -q 8`, 29.25 dB): **7,176 B**.

Three consequences, and they are arithmetic rather than opinion:

1. **Our wall chunk alone is +29.6% against WebP's whole file.**
2. **With colour entirely free we would still lose by 30.0%.** Every colour lever in the study — RCT, residual context models, the entropy floor work — is competing for 20% of a file we lose by 63%.
3. **Reaching parity requires cutting 48.9% of the wall bill.**

## What WebP does that we do not

Report 02 dissected this and its crux still holds: **WebP stores residuals of an index map; we store the geometry of regions, and geometry is redundancy WebP never pays for.**

The sharper version, also from report 02 and worth restating because it is the whole explanation:

> The shape idea is already inside all three winners, in the only form that pays: **implicit, block-local, and derivable without an explicit global boundary map.**

- **VP8** (what we lose to here) carries a segmentation of up to 4 segments with per-segment quantizers, coded as a cheap tree. The segment assignment is **per macroblock**, so the "boundary" is the block grid and costs nothing.
- Edges inside a block are paid for in **DCT coefficients it is already sending** for texture. An edge and a gradient are the same kind of object to a transform codec; to us they are a boundary plus two flat fills.
- **AV1** goes further with wedge and geometric partition modes: a straight-line boundary inside a block, from a small codebook, for a handful of bits.

We transmit an exact, global, pixel-lattice boundary map. That is the product — it is what makes region #4,211 mean the same thing everywhere — and it is also the entire deficit.

Report 04's mechanism is the same fact stated physically: boundary cost grows as √(region count) while a block codec's per-block overhead shrinks faster under quantization, and our walls are measured at **2.22× longer** than compact cells of the same areas — that excess *is* the image information, which is why cheap geometry is also uninformative geometry.

## Two levers revived and tested for this report

**P-08, per-region affine (first-order) colour — CLOSED, NEGATIVE.** `PARKED.md` had this as "killed on numbers that are now stale", with a revive trigger of "after B10 (RCT) lands". B10 landed in report 15 and nobody re-ran it. Re-run now, on the dog:

| fidelity | constant regions | constant bytes | affine regions | affine bytes |
|---|---|---|---|---|
| ~29.2–29.4 dB | 1,229 | **14,356** | 2,048 | 27,300 |
| ~28.1–28.4 dB | 812 | **11,057** | 1,024 | 15,500 |
| ~27.0–27.4 dB | 413 | **6,802** | 512 | 8,900 |
| ~26.3–26.6 dB | 237 | **4,619** | 256 | 5,500 |

**Affine loses at every operating point, by up to 1.9×.** The mechanism is visible in the split: affine *does* buy what it promised on geometry — 1,024 affine regions reach 28.06 dB with 29,145 crack edges where constant needs 41,580 — but the plane coefficients cost **9.2 KB against the boundary's 6.3 KB**. Nine numbers per region instead of three is not worth the walls it saves.

**And it survives the fairness objection.** The affine plane coder predates RCT, so it is being measured with the weaker colour coder while the constant arm has the stronger one — exactly the trap `PARKED.md` warns about. Granting affine a *generous* RCT-scale 28% discount on its plane coefficients: 9.2 KB → 6.6 KB, total 12.9 KB at 28.06 dB, still worse than constant's 11,057 B at the **higher** 28.35 dB. The conclusion does not depend on the unfairness.

**The wall coder is already at its best measured setting.** All ten variants priced on this exact partition:

| variant | bytes | vs `caeBytes` |
|---|---|---|
| **`interAsym` (shipped)** | **9,302** | **−13.93%** |
| `interVH` | 9,540 | −11.72% |
| `inter12` | 9,628 | −10.91% |
| `base` | 10,140 | −6.18% |
| `noCross` | 19,726 | +82.52% |

The shipped coder wins. And `caeBytes` at 10,808 B equals the `hd` table's wall figure at this mark, which means **CAE was chosen over contour here** — so P-01 (contour turn coding) and P-02 (loop channel) are still unchosen at this operating point and **stay parked**. Their trigger did not fire.

## What is actually left, against the size of the hole

We need **4,549 B**. Ranked by what they could plausibly deliver:

| lever | status | plausible size | closes the gap? |
|---|---|---|---|
| P-03 wider CAE context | live, one run, never stacked with the interleave | −3% to −6% of walls ≈ 280–560 B | **no**, ~1/10th of it |
| P-01 / P-02 contour | parked, trigger checked here and did **not** fire | 0 B at this operating point | no |
| P-08 affine colour | **closed negative, this report** | negative | no |
| Colour levers (P-04, P-05) | blocked/parked | ≤ 2,395 B even if colour went to zero | **no**, arithmetically |
| **P-07 approximate / curve-fitted boundaries** | parked on cost, trigger fired | **the only lever of the right magnitude** | maybe — see below |

**P-07 is the only candidate, and it costs the thing we are selling.** Approximating boundaries — a coarse polyline, a wedge codebook, anything that stops being exact on the pixel lattice — attacks 79.3% of the file directly and is what AV1 does. But exactness is the property that makes the mask worth transmitting at all: report 28's whole finding is that a *stable, exact* partition is what a re-segmenting client cannot reproduce. A lossy boundary is a different product, and it should be entered deliberately rather than as a byte optimisation.

## The reframe, on these exact images

Principle 5 says price this against what a consumer would assemble, not against a raster codec alone. On the dog photograph, using our own wall coder as the sidecar — which report 19 measured as the *strongest* available, beating raw labels + `xz -9e` by 2.46×, so this is conservative:

| | bytes | carries a segmentation? |
|---|---|---|
| WebP alone | 7,176 | no |
| **WebP + a region map** | **16,478** | yes |
| **ours** | **11,725** | yes — the structure *is* the image |

**28.8% smaller than the baseline that matches the product**, on a fresh image from a corpus this study had never seen, reproducing report 24's +30.5% on Kodak-24.

Both statements are true and the record should carry both: **a poor image codec, a good structured-image format.**

## Caveats

- One image decomposed in full. The six-image byte table is in report 33; the component split, the affine re-run and the wall-variant sweep are all the dog photograph only.
- The affine arm's plane coefficients are coded by a pre-RCT coder. The generous-discount arithmetic above is what makes the conclusion safe, not a re-measurement — a proper re-measurement would be the honest way to *close* P-08 rather than downgrade it, and it is not done here.
- The sidecar figure prices the region map with our own wall coder. That is favourable to the comparison in the sense that it is the best sidecar available, and unfavourable in that a real consumer would likely use something worse and the margin would be larger.
- Nothing here re-opens the killed list. "Let the decoder regrow the regions" remains provably a context model bounded by `H(X | causal past)` (report 04) and is not a lever.
