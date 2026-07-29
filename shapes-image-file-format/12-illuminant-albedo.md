# 12 — Albedo is not flatter, and a ranking that inverts under a real compressor

**Question.** A natural image is approximately a smooth illumination field times a piecewise-constant albedo field — exactly the structure a region coder should want. Send the illumination cheaply as a global surface, and what remains per region is albedo, which ought to be flatter within a region and repetitive between regions: the same material recurring under different light. Does that decomposition cost fewer bytes than coding the region colours directly?

**Answer.** No, and not marginally. **Dividing region colours by a smooth illumination field raises their joint entropy by 5.9% and grows the alphabet by 50%**, and every shuffled control beats the real field. The premise is false on this image. What survives the round is an 8-byte refinement to the cross-channel transform worth **0.087%** — and a methodological result that matters more than either.

Data: [`12-illuminant-albedo-data.txt`](12-illuminant-albedo-data.txt).

## The premise, measured before any coder was written

Order-0 joint entropy over region colour triples on the exact 4K partition:

| field | joint entropy | distinct values | control: same field, spatial correspondence destroyed |
|---|---|---|---|
| **none — raw region colours** | **13,135,022 B** | **529,557** | — |
| poly2 (24 B field) | 13,914,233 B | 793,543 | 13,422,053 B |
| grid64 (16 KB field) | 13,786,991 B | 941,521 | 12,950,595 B |

**Every shuffled control beats the real field.** If illumination were being removed, destroying the spatial correspondence would hurt; it helps. There is no "same material under different light" to recover in this image's region colours — re-rounding `C/L` just mints new values. Cross-check: the raw figure plus a 529,557 × 3 B codebook is 14,723,693 B, which is report 11's joint-plus-codebook row exactly.

After brotli, every arm that transmits a field is a net loss: poly2 **+0.016%**, poly5 +0.033%, grid16 +0.030%, grid64 **+0.638%**, and textbook Retinex with full multiplicative chroma **+13.76%**.

**It is not report 04's "seeds derived from a shared coarse field" wearing new clothes, and that is demonstrated rather than asserted:** the mechanism fails in the 21-parameter polynomial form *and* fails monotonically worse as the field gets finer. Wrong direction in field resolution. There was nothing to rename because both forms are dead.

## What actually survived, and the controls that make it credible

Additive RCT and multiplicative chroma differ only in the coefficient on the already-coded G step — 1 versus R̂/Ĝ. Fitting `pred_c = ŵ_c + tG·(k_c + a_c·(ŵ_c/Ĝ − 1))` separates the physics from the knob: `a_c` is the only term carrying illuminant×albedo.

| arm | lossless | vs RCT | lossy, 11,121 regions |
|---|---|---|---|
| additive RCT (report 11's reference) | 6,904,345 B | — | 27,201 B |
| **fitted `a` only — the light term** | **6,898,336 B** | **−0.087%** | 26,528 B (−2.47%) |
| fitted `k` only — no light physics | 6,945,864 B | +0.601% | 26,720 B (−1.77%) |
| fitted `k` and `a` | 6,960,551 B | +0.814% | **26,400 B (−2.95%)** |
| **control: `a` negated** | 6,927,936 B | **+0.342%** | 27,313 B |
| **control: `a` on the other channel's ratio** | 6,941,438 B | **+0.537%** | 27,643 B |

Independently re-verified outside the agent's harness: `a` = 6,898,328 B + 8 B of coefficients, `aneg` = 6,927,928 B, `aswap` = 6,941,430 B. **Both controls cost more than doing nothing**, so direction and channel pairing both matter — this is information about the channel's own reflectance ratio, not extra capacity. No field is transmitted; the ratio is inferred from the already-decoded G. Total side information: **8 bytes**.

**Caveat that limits it hard.** The fitted coefficients are `a_R = −0.014`, `a_B = +0.136` — the entire win is the blue channel. On a sky-and-mountain photograph that is physically plausible and generically fragile. B7 stands.

## The result worth more than the lever: modelled bytes and real bytes rank differently

**`k`-only improves the order-0 model by 0.04% and makes brotli 0.60% worse.** The mechanism is visible in the stream:

| stream | zero % | differs from RCT | mean run |
|---|---|---|---|
| rct | 36.693 | 0.000 | 1.5161 |
| **a** | **36.845** | 4.731 | 1.5079 |
| k | 36.370 | 8.087 | 1.4912 |
| ka | 36.055 | 10.755 | 1.4784 |

At 1.305 px/region the residual stream is **37% zeros**, and brotli lives on that. A predictor refinement that lowers residual *variance* while lowering the *exact-hit rate* trades away more LZ matches than it wins in entropy. `a` leaves achromatic regions bit-identical to the baseline — where `ŵ_c/Ĝ − 1 = 0` — so it survives brotli; `k` perturbs everything, so it does not.

**Consequence for this study: any colour lever ranked on modelled bytes can invert once a real compressor is the reference.** Report 11's own ladder ranks `k` and `a` the other way round. Every bespoke colour figure in reports 04–09 is a modelled number, and none of them has been checked against a general compressor on the same stream.

## Regime split, unchanged

At 11,121 regions the best arm saves 801 B — **0.52% of that 153,190 B rung**, against the 8.3% still needed at 28.7 dB after report 09. Report 11's conclusion stands: colour owns the lossless end, and **walls must close the mid-axis arm**.

## What this closes

The light lens is **closed negative on its structural claim**. A smooth illumination field, transmitted at any resolution, is a net loss at both ends of the ladder. The 8-byte chroma coefficient is a real but marginal refinement to the cross-channel transform and belongs with B10 when that transform is adopted, not as a mechanism of its own.

The lossless colour bill is now **6,898,336 B against 6,896,137 B for a WebP colour tie — a gap of 2,199 B**, down from 8,208. What remains on colour is bookkeeping, not research.
