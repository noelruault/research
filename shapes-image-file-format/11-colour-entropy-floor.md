# 11 — The colour coder is not badly implemented, it is badly specified

**Question.** Every colour lever this programme queued — better predictors, residual contexts, coding order — competes for the same slack. How much slack is there? Measure `H(region colours | partition)` before spending another agent on a mechanism.

**Answer.** The floor does not block the target, and the thing standing between the coder and the floor is not on the queue. **The current coder sits 0.02% above the static ideal of its own model and 36% above the floor of its own data.** Its entire deficit is one missing transform.

Data: [`11-colour-entropy-floor-data.txt`](11-colour-entropy-floor-data.txt). Code: `code/lab/floor.go` (`floor`, `floordec`, `floordump`).

## The ladder, at lossless, in bytes on the real 4K partition

6,356,392 regions, 1.305 px/region. Colour is 92.9% of this bill.

| conditioning | bytes | kind | vs coder |
|---|---|---|---|
| `H(C)` order-0, per channel | 17,788,187 | static | +64.2% |
| `H(C)` order-0, joint RGB + codebook | 14,723,693 | static | +35.9% |
| `H(C │ best single decoded neighbour)` | 11,337,015 | achievable | +4.7% |
| **`H(C │ boundary-weighted mean)` — the current coder** | **10,832,609** | achievable | — |
| the same model's static ideal | 10,830,226 | oracle | −0.02% |
| `H(C │ MED on decoded pixels)` | 10,766,697 | achievable | −0.61% |
| **+ reversible cross-channel transform (G, R−G, B−G)** | **7,800,674** | achievable | **−28.0%** |
| + residual context (spread × decoded-neighbour count) | 7,295,931 | achievable | −32.7% |
| token alphabet + MED + RCT + gradient + cross-channel | 7,046,309 | best bespoke | −35.0% |
| **brotli −q11 on the RCT residuals, planar** | **6,904,345** | **decode-verified** | **−36.26%** |
| static oracle, MED+RCT+grad+prev, 12,798 contexts | 6,819,837 | oracle | −37.0% |

**The two rows that matter are adjacent.** The coder is within 0.02% of the best its own model can do — it is a competent implementation of the wrong model. Add a cross-channel transform and 28% falls out. RCT is the thing report 02 catalogued as one of WebP's four transforms in 2026, and the region coder simply never had one.

> **Caveat added by report 12.** The ladder above is *modelled* bytes. Report 12 found a case where that ordering inverts under a real compressor: a predictor refinement that improves the order-0 model by 0.04% makes brotli 0.60% worse, because at 1.305 px/region the residual stream is 37% zeros and brotli lives on the exact-hit rate rather than on residual variance. Treat the modelled rows as ranking *models*, not ranking *streams*. Only the brotli row is a measured stream.

## This reorders the queue

| lever | measured alone | on top of RCT |
|---|---|---|
| **cross-channel transform (RCT)** | **−28.0%** | — |
| residual context (queued as B9, the "largest win") | −10.33% | **+4.7 pp** |
| RD choice among 5 predictors | −5.24% | subsumed |
| coding order | −2.06% | ~0 |

B9 was queued as the biggest colour lever on the strength of −10.33%. It is a third of RCT, and stacked on RCT it is worth less than half its solo figure. **Every lever on the colour queue was competing for slack that one missing transform already accounted for.**

## Is −36% of the colour bill reachable? Yes — it has been reached

A WebP tie on the colour bill alone needs 6,896,137 B. The decode-verified stream is **6,904,345 B**. The gap is **8,208 B — 0.076% of the colour bill** — and it was produced with **no bespoke modelling at all**: the residuals handed to off-the-shelf `brotli -q11`.

Independently reproduced: re-running `brotli -q 11` on the dumped residual stream outside the agent's harness returns 6,904,345 B to the byte.

**This is a real stream, not a cross-entropy estimate.** A decoder given only the partition and these bytes rebuilds the image: **0 wrong samples of 24,883,200, max |Δ| = 0**, on all four stream variants.

## What is deliberately *not* claimed

- **Not a WebP tie.** 822,369 + 6,904,345 = 7,726,714 against WebP's 7,718,506 is 1.001×, but that wall figure comes from the coster report 06 #12 showed is **not decodable**, and its legality cost at the lossless rung was never tabulated. The arithmetic above covers the colour bill only. Anyone quoting a tie is quoting an illegal wall number.
- **brotli is an upper bound on achievable, never a lower bound on entropy.** No CM/PAQ-class coder was installed, so brotli is the strongest modeller reached and **the true floor is below 6,904,345**.
- **The oracle rows are not floors.** `H(C │ best neighbour)` with 525,851 contexts over 12 samples each reads 3,486,939 B; that is sample splitting, not information. At the lossy point a **junk control of 7,920 meaningless contexts reaches 9,051 B**, so no wide-context static number at that region count is quotable at all. At lossless the matched junk control puts overfit at 24,813 B, so the static rows there hold to ~0.2%.
- **One image.** B7 stands.

## The lossy regime answers differently, and it closes an argument

At 11,121 regions (745.8 px/region, colour = 21% of the bill):

| | bytes | vs coder |
|---|---|---|
| `colorBytes2` anchor | 32,143 | — |
| MED alone | 32,427 | **+0.9% — worse** |
| + RCT | 28,580 | −11.1% |
| brotli, RCT planar | **27,201** | **−15.4%** |

The raster predictor goes *positive* and the context ladder starves; essentially the whole win is again the cross-channel transform. But **−15.4% of colour is −3.2% of the total**, against the 8.3% still needed at 28.7 dB after report 09.

**So colour cannot close the mid-axis arm. Walls must.** The two ends of the ladder have now swapped owners: colour owns the lossless end and can nearly close it, walls own the middle and colour cannot help there.

## What this changes

The colour programme continues, but against a different target than the one queued. The next step is not another lever — it is adopting the cross-channel transform into `colorBytes2` and re-pricing the record, because every colour figure in reports 04 through 09 was measured without it.
