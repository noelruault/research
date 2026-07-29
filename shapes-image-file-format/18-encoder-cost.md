# 18 — Encode costs 3m44s and 2.9 GB, and that decides the applications

**Question (P6).** Report 13 listed encoder cost as **UNMEASURED** and "appears large" — agents had been recovering partitions from rendered PNGs specifically to avoid re-running the merge. If a 4K encode takes minutes, whole application classes are excluded. Measure it.

**Answer.** It is worse than "appears large", and it is a *lever* rather than a wall.

## The numbers

Apple M5 Pro, 15 cores, single process, nothing else loaded. `lab hd` walks the full agglomerative scale-space — the merge starts at the exact partition and coarsens through every mark, so reaching any one operating point requires most of the work regardless.

| | shape encoder | `cwebp -m 6` | ratio |
|---|---|---|---|
| 960×540, full ladder (14 marks) | **10.2 s** | 0.15 s | **68×** |
| 3840×2160, full ladder (20 marks) | **3 m 44 s** (223 s user) | ~1.5 s | **~150×** |
| peak resident memory at 4K | **2.89 GB** | — | — |

**CPU utilisation was 102–103%** — the encoder is effectively single-threaded on a 15-core machine.

Scaling 960→4K is 22× wall-clock for 16× the pixels: slightly superlinear, consistent with an agglomerative merge whose priority queue grows with region count.

## What this excludes, and what it does not

**Decode is unaffected and remains the format's strength.** Decoding is filling regions with colours — no entropy-decode-then-inverse-transform-then-upload pipeline. Nothing measured here touches viewing cost. The asymmetry is extreme in the same *direction* as modern video codecs, just further along it.

| application | verdict |
|---|---|
| Archival, scientific, forensic | **fine** — encode once, offline, cost irrelevant |
| Author-time assets (games, animation, print) | **fine** — encode in the build, ship the result |
| Batch CDN re-encoding of a library | **expensive but survivable** — 3.7 min/image at 4K is a real bill, not a blocker |
| Upload-time encoding in a web service | **excluded** — nobody waits 3.7 minutes, and 2.9 GB per worker is untenable |
| Interactive re-encode while editing | **excluded** — though editing *within* an existing partition is O(regions) and unaffected |
| Mobile or embedded encode | **excluded** — 2.9 GB settles it on its own |

The applications report 13 ranked highest — selection primitives, non-destructive editing, cutout animation — all operate on an *already-encoded* partition. **None of them is blocked by this.** What is blocked is anything that needs to encode on demand.

## Why this is a lever and not a wall

Three things about the measurement say the number is soft:

1. **Single-threaded.** 103% CPU on 15 cores. The Ising relaxation sweeps and the per-mark pricing are embarrassingly parallel; only the agglomerative merge itself is inherently sequential.
2. **It computes the whole ladder.** All 20 marks are priced when a production encoder targeting one operating point needs one. The merge must still pass through the region counts, but pricing every mark is pure overhead.
3. **It is research code.** Nothing here has been profiled once. There is no evidence the constant factor is near its floor — and this study has now twice found large wins in components assumed to be mature.

**None of that is measured**, so the honest claim is: 3m44s is the cost of the current implementation, not of the idea.

## What this changes in the roadmap

Report 13's **W8 is now measured** rather than flagged, and it moves up: it does not block the stage-4 applications, but it eliminates a whole class of deployment before any of them ships, and that should be known now rather than after the container is built.

**New roadmap item — P8: profile and parallelise the encoder.** Not research, engineering, and it has an obvious first step (parallelise the relaxation sweeps, stop pricing marks nobody asked for). Placed in stage 2 alongside the container, because "3.7 minutes" is the kind of number that kills adoption arguments before the byte numbers get heard.
