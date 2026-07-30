# 25 — 43% of the encode was work nobody asked for, and parallelism is the wrong lever

**Question (P8).** Report 18 measured the encoder at **3 m 44 s and 2.89 GB** for a 4K image, at 102% CPU on a 15-core machine, and flagged it as a lever rather than a wall: single-threaded, pricing all twenty scale-space marks when a production encoder needs one, never profiled. Profile it and parallelise.

**Answer.** The prescription was wrong. **Parallelism is not the lever — not doing the work is.** Pricing one mark instead of twenty takes the encode from **3 m 44 s to 2 m 7 s (−43%)** with byte-identical output, and what remains is a sequential merge that threads cannot help.

## Where the time went

The per-mark timeline from a full 4K run, cumulative:

| mark (regions) | done at | interval |
|---|---|---|
| 3,380,956 | 32 s | 32 s |
| 2,005,132 | 1 m 7 s | 35 s |
| 1,189,921 | 1 m 37 s | 30 s |
| 710,144 | 2 m 0 s | 23 s |
| 423,771 | 2 m 17 s | 17 s |
| … | … | shrinking |
| 11,121 | 3 m 18 s | 5 s |
| 227 | 3 m 44 s | 3 s |

**The four finest marks consume 2 m 0 s of the 3 m 44 s — 54%** — and they are partitions nobody ships. Per-mark cost scales with region count: relaxing 3.38M regions, contour-coding 10.8M crack edges and writing a 4K PNG is expensive, and it is done twenty times.

Reading `hd.go` rather than a profiler is what settled it: **the per-mark work runs on a *copy*** — the snap callback copies the parent array, relabels, relaxes, prices and writes a render, while the merge carries on from the *unrelaxed* partition. So every mark is independent of every other, and of the merge.

## The measurement

`HDONLY=<n>` keeps only the mark nearest *n*, which is what an encoder shipping a single file does.

| | time | output |
|---|---|---|
| full ladder, 20 marks | **3 m 44 s** | — |
| **single mark @ 11,121 regions** | **2 m 7 s** | **byte-identical render**, identical pricing |
| of which the lossless stage | 1 s | — |

The render `md5`s match the published one exactly and the priced row is identical to the ladder's, so **the 43% is free**: no approximation, no quality change, just not computing nineteen things nobody wanted.

## Why parallelism is the wrong prescription

The remaining 2 m 7 s is almost entirely the **agglomerative merge from one region per pixel** — 8,294,400 regions down to 9,680, one merge step at a time, each step's choice depending on the previous state. That is sequential by construction. The lossless stage is 1 s and the surviving mark's pricing is a few seconds.

So the part that *is* embarrassingly parallel — per-mark pricing on independent copies — is exactly the part just deleted. Threading what remains would buy close to nothing.

**The real remaining lever is algorithmic, not parallel:** the merge begins at one region per pixel, so a 4K image costs ~8.28M merge steps to reach ~11k regions. Starting from a coarser initial partition — the exact partition is already 6.36M, and a superpixel pre-pass would be far coarser — would cut the step count by orders of magnitude. **That changes the partitions**, so every baseline in the study would need re-running on the new ones or it reproduces falsification #3. It is a real option and a dangerous one, and it is not what "parallelise the encoder" meant.

## What this changes

| | before | after |
|---|---|---|
| 4K encode | 3 m 44 s | **2 m 7 s** |
| 960×540 encode (report 18) | 10.2 s | ~6 s (same ratio, unmeasured) |

Still ~85× `cwebp` at 4K. **The application verdict from report 18 does not move**: fine for archival, author-time assets and batch pipelines; still excluded from upload-time encoding, interactive re-encode and mobile, where 2.89 GB settles it regardless of the clock.

Report 13's **P8 is therefore half done and half reclassified.** The free half is taken. The other half is not a parallelism task, and the roadmap entry saying so was wrong.

## Caveats

- Peak memory was **not** re-measured; the 2.89 GB figure stands and is driven by the merge's per-pixel structures, which `HDONLY` does not touch.
- One image at one target mark. The 43% is the ratio for a 20-mark ladder reduced to 1; an encoder wanting three operating points saves proportionally less.
- The 960×540 figure above is scaled, not measured.
