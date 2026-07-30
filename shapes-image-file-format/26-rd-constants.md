# 26 — The "stale" RD constants are better than the correct ones

**Question (P-06).** `potts2.go` hardcodes `bitsPerEdge = 1.73` and `bitsPerReg = 25.0`, both measured at 512×288, and both drive the rate-distortion merge key and the Ising relaxation's λ at every resolution and on every image. The parked register called this **plausibly the largest single win left** and the most dangerous item in the queue, since re-tuning changes the partitions themselves. Re-measure and re-tune.

**Answer.** Re-tuning makes it **worse** — slightly but consistently, on every image tested. The constants are not stale cost estimates that drifted; they are tuning parameters whose best value is not their measured value.

## What the constants should be, if they were cost estimates

Measured across all 24 Kodak images at their ~4,500-region marks:

| | constant | measured (median) | error |
|---|---|---|---|
| `bitsPerEdge` | 1.73 | **1.529** | +13% high |
| `bitsPerReg` | 25.0 | **22.4** | +12% high |
| **ratio** `bitsPerReg / bitsPerEdge` | **14.45** | **14.63** | **+1.2%** |

**The ratio — the thing the merge key actually depends on — was already right to 1.2%.** The merge decides between candidates on `dSSE / (l·bitsPerEdge + bitsPerReg)`, so scaling both constants together only rescales λ, which slides along the ladder rather than changing which merges happen. Two numbers that were each ~12% wrong were wrong by nearly the same factor.

## The measurement

Each arm builds its **own** partitions and is compared at matched **fidelity** — the mark in each ladder nearest the old arm's PSNR — never by pricing one arm's partition with the other's coder, which would reproduce falsification #3.

| image | arm | regions | PSNR | SHPC | sidecar margin |
|---|---|---|---|---|---|
| kodim01 | old | 4,604 | 27.59 | **28,013** | **+37.2%** |
| kodim01 | re-tuned | 4,771 | 27.64 | 28,810 | +36.2% |
| kodim05 | old | 4,457 | 26.65 | **30,950** | **+34.6%** |
| kodim05 | re-tuned | 4,456 | 26.66 | 31,026 | +34.5% |
| kodim17 | old | 4,340 | 32.93 | **26,915** | **+26.3%** |
| kodim17 | re-tuned | 4,344 | 32.95 | 26,982 | +26.2% |
| kodim23 | old | 4,387 | 34.76 | **25,702** | **+18.7%** |
| kodim23 | re-tuned | 4,392 | 34.77 | 25,785 | +18.6% |

Worse on all four, on **both** target metrics — raw bytes and the sidecar margin that report 24 made the one that matters. And the comparison is **generous to the re-tuned arm**: its PSNR is fractionally *higher* in every case, so it is being credited with slightly more fidelity for its extra bytes.

## Why the wrong value is the better one

The ratio governs the merge, and it barely moved. What moved is the **absolute** `bitsPerEdge`, which enters the Ising relaxation as `relax(im, lab, nreg, lambda*bitsPerEdge, 6)` — a straightening pressure, not a cost estimate.

Report 04 measured that relaxation alone, straightening walls without changing the region count, is worth **15%**. Straighter walls code better than a linear per-edge model predicts, because the context coder pays for unpredictable *turns*, not for edges. So over-weighting the wall term relative to its true linear cost buys real bytes, and setting the constant to its "correct" measured value **weakens the straightening and loses them**.

**1.73 is not a stale measurement of 1.529. It is a tuning parameter that happens to look like a cost.** The comment calling it "measured cost of one crack edge under the CAE coder" is what made it look stale, and that comment is the actual defect.

## Outcome

**P-06 is closed negative.** The constants are unchanged in the repo; both binaries were built and the working tree was restored to the original values before anything was committed.

The parked entry that called this "plausibly the largest single win left" was wrong, and its reasoning is worth keeping as an example: it argued from the *gap between a constant and a measurement* without asking what the constant was actually doing. The gap was real — 13% — and irrelevant, because the constant is not that measurement.

## Caveats

- Four images, one operating point each. A wider sweep could find content where the re-tune helps; the effect sizes here (0.2–2.8%) are small enough that four images is thin evidence for a general claim, though the *sign* is consistent.
- Only the two constants were varied, together, to their measured values. **The tuning question is unexplored in the other direction**: if over-weighting walls helps, values *above* 1.73 may help more. That is a genuine open lever and the opposite of what this item proposed. Queued as P-06b.
- The relaxation explanation is inferred from report 04's 15% figure and the code path, not isolated by an ablation. **Unverified** as a mechanism, though the outcome it explains is measured.
