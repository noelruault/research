# 15 — The colour column re-priced, and a modelled number with the wrong sign

**Question (B10).** Reports 11 and 12 established that the region colour coder never had a cross-channel transform, and that modelled bytes do not rank like real bytes. Apply both to every published operating point and re-price the colour column.

**Answer.** The transform helps **everywhere**, from −9.2% at 227 regions to −36.3% at lossless, monotonically. And the modelled column — the kind of number nine reports were built on — is **not merely mis-ranked at the low end, it has the wrong sign**.

Data: [`15-recolour-data.txt`](15-recolour-data.txt). Code: `code/lab/recolour.go` (`recolour`, `rcdec`).

## The re-priced column

Every mark at 3840×2160, same labels, `brotli -q11` on the actual dumped residual stream.

| regions | PSNR | `colorBytes2` | RCT modelled | **RCT brotli** | +8 B coeff | **real Δ** |
|---|---|---|---|---|---|---|
| 227 | 21.99 | 740 | **795 — worse** | **672** | 674 | **−9.24%** |
| 344 | 22.72 | 1,108 | **1,174 — worse** | 996 | 1,008 | −10.14% |
| 536 | 23.38 | 1,708 | **1,780 — worse** | 1,504 | 1,488 | −11.96% |
| 849 | 24.25 | 2,654 | **2,700 — worse** | 2,317 | 2,292 | −12.69% |
| 1,383 | 24.99 | 4,219 | 4,189 | 3,652 | 3,571 | −13.44% |
| 6,417 | 27.50 | 18,694 | 17,160 | 16,003 | 15,607 | −14.39% |
| 11,121 | 28.51 | 32,143 | 28,580 | 27,201 | 26,528 | −15.37% |
| 96,359 | 32.74 | 256,936 | 211,288 | 208,732 | 206,393 | −18.76% |
| 710,144 | 40.42 | 1,721,406 | 1,348,719 | 1,317,392 | 1,312,707 | −23.47% |
| 3,380,956 | 53.37 | 6,863,309 | 5,076,591 | 4,635,675 | 4,625,993 | −32.46% |
| **6,356,392** | exact | **10,832,609** | 7,800,674 | **6,904,345** | **6,898,336** | **−36.26%** |

Full 21-mark table in the data file. **Both arms are decode-verified at both ends**: partition + stream (+ the 8 transmitted bytes) rebuild the image with **0 wrong samples of 24,883,200, max |Δ| = 0**.

Every anchor reproduced independently before anything was believed: `colorBytes2` = 10,832,608.78 and 32,142.63; `rct_modelled` = 7,800,674 (report 11 exactly); brotli = 6,904,345 and 6,898,336 (reports 11 and 12 exactly); coefficients −0.01432/+0.13591 and stream statistics 36.693% zeros / 1.5161 mean run (report 12 exactly). Recovery drift is +0.0000% at both anchors.

## The sign flip

At 227 regions the modelled cost of RCT is **795 B against `colorBytes2`'s 740 — 7.4% worse.** The compressed stream is **672 B — 9.2% better.**

The modelled number does not rank wrong. It points the **wrong way**. The same inversion holds at 344, 536 and 849 regions, and only closes at 1,383.

Report 12 found modelled and real bytes ranking two refinements differently. This is the same effect at a severity that report did not reach: a lever a modelled evaluation would have **rejected outright at four of the twenty-one operating points** is in fact the largest colour win in the study. The mechanism is report 12's: at low region counts the residual stream is dominated by exact hits and short runs, and brotli lives on those rather than on residual variance — so a transform that raises modelled entropy while raising the exact-hit rate wins in bytes and loses on paper.

**This is a methodological finding about the whole record, not about RCT.** Every colour figure in reports 04–09, and the modelled columns of 11 and 12, are cross-entropies. None was checked against a real compressor. At least four operating points here would have been read backwards.

## Combined effect, and what may not be claimed

At 11,121 regions, taking report 09's interleaved wall coder (which is causal, unlike the published one) together with this colour column:

| | walls | colour | total |
|---|---|---|---|
| published | 121,047 | 32,143 | **153,190 B** |
| interleave + RCT + 8 B | 105,752 | 26,528 | **132,280 B** |

**−13.6%**, and every component of the new total is decodable — unlike the published figure, whose wall half report 06 #12 showed no decoder can produce.

**`cwebp -m 6` is 137,033 B at 28.7 dB. This mark is at 28.51 dB. Those are not the same fidelity, and I am not going to compare them.** Reading 132,280 against 137,033 would be falsification #1's error — mismatched fidelity — which this study has already made once. **The matched-fidelity comparison has not been run and must be before anyone claims parity.** That is the next task, and it is a single WebP quality search.

Also unclaimed: at lossless, 822,369 + 6,898,336 = 7,720,705 against WebP's 7,718,506 is **1.000×** — but the 822,369 wall figure is the illegal coster. A legal lossless wall number does not exist yet.

## What changes in the record

- **Every colour figure in reports 04–09 is understated** by 9–36% depending on operating point, and the modelled columns of 11 and 12 are directionally wrong below 1,383 regions.
- Report 11's "−28.0% modelled" and report 12's ladder are confirmed at their measured points and **superseded as a ranking tool** — the brotli column is the one to quote.
- Report 14's finding that the capability sweet spot is 227–1,383 regions gains a corollary: at exactly those region counts, a modelled evaluation of the colour coder is **actively misleading**.
