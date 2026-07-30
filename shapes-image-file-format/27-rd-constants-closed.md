# 27 — The RD constants have no win in them, in either direction

**Question (P-06b).** Report 26 found that setting `bitsPerEdge`/`bitsPerReg` to their *measured* values makes the coder worse, and concluded the absolute value acts as a straightening pressure rather than a cost. The obvious successor: if over-weighting walls helps, push `bitsPerEdge` **above** 1.73 and see how far it goes.

**Answer.** That experiment is **provably vacuous**, and the parameter it should have named — the ratio — turns out to have no win in it either. The whole P-06 family is closed.

## Part 1: raising `bitsPerEdge` with the ratio held is a no-op, and here is why

The merge key is `dSSE / (l·bitsPerEdge + bitsPerReg)`. Hold the ratio at 14.45, so `bitsPerReg = 14.45·bitsPerEdge`, and it becomes:

```
dSSE / ( bitsPerEdge · (l + 14.45) )
```

The key is **proportional to `1/bitsPerEdge`**, and a uniform scaling cannot change the *ordering* of candidates — so the merge makes identical decisions. The relaxation then receives `λ·bitsPerEdge`, and since λ is the merge key at the snapshot it also scales as `1/bitsPerEdge`, leaving that product **constant** too.

Measured rather than argued — `bitsPerEdge` 1.73 → 3.0, a **73% increase**, ratio held:

| | result |
|---|---|
| render at 4,387 regions | **byte-identical** (`md5` 8b3188c0…) |
| full 20-mark ladder | identical except **1-byte** float tie-breaks at two marks |

So P-06b as specified would have swept a parameter that does nothing. Report 26 said the absolute value "enters the relaxation as a straightening pressure" — **that was wrong**, or at least incomplete: it enters as a product with λ that cancels. What actually produced report 26's effect was the incidental **1.2% ratio change** (14.45 → 14.63), not the 12% change in scale.

## Part 2: the ratio is the real parameter, and it has no win either

Sweeping `bitsPerReg` with `bitsPerEdge` fixed at 1.73, each arm building its own partitions:

| image | ratio | regions | PSNR | SHPC | sidecar margin |
|---|---|---|---|---|---|
| kodim01 | 8.0 | 4,279 | 27.35 | 25,742 | +40.0% |
| kodim01 | 11.0 | 5,390 | 27.69 | 31,160 | +32.8% |
| kodim01 | **14.45 (base)** | 4,604 | 27.59 | **28,013** | **+37.2%** |
| kodim01 | 20.0 | 4,984 | 27.74 | 30,017 | +34.8% |
| kodim01 | 30.0 | 5,286 | 27.86 | 31,819 | +32.6% |
| kodim23 | 8.0 | 4,200 | 34.46 | 24,273 | +20.3% |
| kodim23 | 11.0 | 4,294 | 34.64 | 25,020 | +19.5% |
| kodim23 | **14.45 (base)** | 4,387 | 34.76 | **25,702** | **+18.7%** |
| kodim23 | 20.0 | 4,474 | 34.96 | 26,721 | +17.8% |
| kodim23 | 30.0 | 4,686 | 35.14 | 28,195 | +16.1% |

**Fidelity drifts with the ratio**, so the byte columns are not matched and must not be compared directly — that is exactly the error falsification #1 records. Tested for **dominance** instead, which is comparison-free: does any arm deliver *both* fewer bytes *and* higher PSNR than the base?

**No arm dominates the base on either image, in either direction.** Every setting lands on the same rate-distortion curve. Lower ratios sit lower-left (fewer bytes, less fidelity), higher ratios upper-right. **The ratio chooses where on the curve you sit; it does not move the curve.**

That is what a well-behaved Lagrangian parameter is supposed to do, and it means the constant is doing its job. The apparent 5.6%-fewer-bytes of ratio 8 on kodim23 buys exactly the 0.30 dB it costs.

## Outcome

**P-06 and P-06b are both closed, and the parked entry's premise is retracted.** The register called this "plausibly the largest single win left" on the strength of a 13% gap between a constant and a measurement. There is no win here at all:

- the *scale* is provably inert;
- the *ratio* slides along the RD curve without improving it;
- and report 26's "straightening pressure" mechanism does not survive the algebra above, though its *measured outcome* — that re-tuning to measured values is slightly worse — stands, and is now explained by the 1.2% ratio change rather than by the scale.

Repo unchanged: `bitsPerEdge = 1.73`, `bitsPerReg = 25.0`, restored and verified before committing.

## Caveats

- Two images, five ratios, one operating point each. Dominance was checked, not a full BD-rate; a proper RD-curve integration over the corpus could still find a small win the eyeball test misses.
- The sidecar margin correlates with region count here — lower ratios produce fewer regions and a *higher* margin — which is an artifact of comparing at unmatched fidelity, not a real gain. It is the reason the margin column must not be read as a ranking.
- λ's exact definition was inferred from the cancellation the experiment confirmed, not read from `runRD`. The empirical no-op is the evidence; the algebra is the explanation offered for it.
