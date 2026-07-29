# 09 — The wall coder had two bugs and one free 12%

**Question.** Reports 01–08 treated the crack-edge coder as a solved component: MPEG-4-style context arithmetic coding, the approach that beat vertex/polygon coding by 20.5% on shapes' own turf, so presumably near its floor. At 3840×2160 walls are **79% of the bill** at the operating point where the WebP comparison is closest, so if that assumption is wrong it is wrong about most of the file. Two independent attacks were run against it, plus an outside review of the whole study.

**Answer.** The assumption was wrong twice over. The published coder is **not decodable**, and a change that costs nothing — no extra contexts, no extra models, no side information — takes **12.6%** off the wall bill at the operating point. Neither finding rescues the shape idea; both change numbers the record had been quoting.

Data: [`09-crossplane-wall-data.txt`](09-crossplane-wall-data.txt), [`09-context-width-data.txt`](09-context-width-data.txt). Code: `code/lab/crossplane.go`, `code/lab/wallctx.go`, `code/lab/selected.go`.

## Result 1 — the win is the coding schedule, not the context size

The crack map is two binary planes: `V` (vertical cracks) and `Hz` (horizontal). The published coder writes **all of `V`, then all of `Hz`**. That ordering is the whole problem.

Around each vertex of the crack lattice the four incident bits are the four "labels differ" indicators of a 4-cycle over a 2×2 pixel neighbourhood, so a single one among three zeros is impossible — knowing three constrains the fourth hard. Count how much of that structure each plane is allowed to use:

- **`Hz` already has all six** other members of its two vertices, and it is enormously valuable: strip `Hz`'s four `V` taps and the wall bill **doubles**, 121,047 → 246,460 B.
- **`V` has one** — `V(x,y-1)`. Its vertices' `Hz` members are all invisible to it, because `V` is a finished plane before `Hz` starts.

So the first plane pays the entire "where does a contour start" cost blind, and it is the expensive plane: at 11,121 regions the split is **106,875 B of `V` against 14,172 B of `Hz`**.

Interleaving the two planes in a single raster scan — `V(x,y)` then `Hz(x,y)` at each pixel — means neither plane is ever coded blind. At 3840×2160, 11,121 regions, on identical partitions:

| variant | context bits V/Hz | models | walls | vs base |
|---|---|---|---|---|
| base (published `caeBytes`) | 10/10 | 2,048 | 121,047 B | — |
| `noCross` (Hz stripped of V taps) | 10/10 | 2,048 | 246,460 B | +103.6% |
| `base12` (more capacity, same schedule) | 12/12 | 8,192 | 119,733 B | −1.09% |
| `hzFirst` (code Hz plane first) | 10/10 | 2,048 | 133,594 B | +10.37% |
| **`interVH` (interleave)** | **10/10** | **2,048** | **107,577 B** | **−11.13%** |
| **`interAsym` (interleave, 8/12 budget)** | 8/12 | 4,352 | **105,752 B** | **−12.64%** |

**`interVH` matches the baseline's context width and model count exactly** and still takes 11.1%. Quadrupling the models on the old schedule buys 1.1%. Capacity is not the lever; the schedule is. Both schedules decompose the same `H(V, Hz)`, so this is a pure model-efficiency result — the same 10-bit budget buys more when neither factor is conditioned on nothing.

Simply reordering does not work: `hzFirst` is **10.4% worse**, because it only transposes which plane pays blind, and `Hz`-alone (246,460 B) is dearer than `V`-alone (106,875 B) on this content.

**Where it pays.** The win band is roughly 25–31.5 dB at 4K, peaking at −12.9% at 6,417 regions. It **reverses at fine partitions** — up to +12.6% worse at 3.4M regions, and +1.7% on the exact partition — because interleaving costs `Hz` its lower-vertex taps, a trade that is only good when walls are sparse and long. Below ~1,400 regions the contour coder is already cheaper than either, so the lever is worth exactly zero there.

At the study's read-off fidelities, against `cwebp -m 6`:

| fidelity | published total | with `interAsym` | deficit vs WebP, before → after |
|---|---|---|---|
| **28.7 dB** | 163,471 B | **148,450 B** | **+19.3% → +8.3%** |
| 30.0 dB | 245,180 B | 232,807 B | +32.7% → +26.0% |
| 31.5 dB | 381,339 B | 375,419 B | +48.6% → +46.3% |

A bit over half the 26,438 B target, closed by a scan-order change. **It does not clear WebP and it changes no verdict.**

## Result 2 — the context *is* under-conditioned at 4K, and that was invisible at 512×288

The second attack widened the context template instead of reordering it. Greedy tap selection grew the `V` plane from 10 bits to 16 (the new taps are all long-range horizontal reach — the coder was blind to runs of vertical cracks longer than 2 px), then **froze that template** and applied it unchanged at every resolution:

| resolution | samples/plane | Δ at the coarsest mark | Δ at the finest |
|---|---|---|---|
| 512×288 | 147,456 | **+2.07%** | +2.43% — worse at every one of 8 marks |
| 960×540 | 518,400 | −3.16% | +2.00% — crosses zero at ~1,200 regions |
| 1920×1080 | 2,073,600 | −7.53% | −0.59% |
| **3840×2160** | 8,294,400 | **−10.32%** | +0.30% |

**The same template is 2–4% worse at 512×288 and 10.3% better at 4K.** The 10-bit context was chosen at the small eval and was correct there; at 4K it is under-conditioned. This is falsification #2's lesson pointing the other way, and it would have been invisible at the old eval size — a prototype at 512 would have rejected the idea outright.

Two controls matter. Static `H(X|ctx)` keeps falling out to 20 bits while the adaptive cost turns over at 16, so the ceiling is **model-learning cost, not information exhaustion**. And a zero-fitting control — extend the template with whichever causal tap is merely *nearest*, having looked at no data — still delivers **−4.45%** of the −6.22%, so the result does not depend on the selection.

**But most of it does not become bytes.** Below ~6,400 regions the contour coder already wins, so a 10% cheaper CAE is worth nothing; the payoff band is 6,417 to ~2M regions, peaking at −6.22%. At lossless it is −0.07% of a bill that is 93% colour.

**Head to head, the schedule beats the capacity by two to one** — and does it at identical model count. Fable, reviewing the study, predicted "under-conditioned, not starved, so grow the template first". The diagnosis was right and the remedy ordering was wrong.

## Result 3 — the published wall coder cannot be decoded

`potts.go:311` puts `get(Hz, x+1, y)` in the `Hz` context: a crack edge to the **right** in the same row, which has not been coded yet. The same context also reads `Hz(x-1,y)` and `Hz(x-2,y)` to the left, so **no scan order can supply all three** — left-to-right lacks the right tap, right-to-left lacks the left ones. A decoder-side replay that rebuilds each context from planes filled in only as the schedule reaches each bit reports **21,554 mismatching contexts at 512×288 and 51,995 at 960×540**; every variant declared causal reports zero.

So every CAE number this study has published was produced by a coder that no decoder can run. Repairing the tap in place costs, at 4K:

| regions | 6,417 | 11,121 | 19,338 | 96,359 | 710,144 |
|---|---|---|---|---|---|
| cost of legality | +3.4% | **+4.6%** | +6.3% | +11.9% | +12.7% |

**Wherever CAE is the chosen wall coder, the published wall numbers are optimistic by that much** — the eleventh and twelfth entries in this study's ledger of errors that flattered the hypothesis. It does not overturn result 1: against a *legal* baseline of 126,583 B the interleave is **−16.5%**, better than the −12.6% quoted against the illegal one.

**Report 08's tables have not yet been re-priced against a legal coder.** That is a known outstanding correction, not a completed one.

## What is not claimed

- **One image.** Everything here is the same 4K Sierra photograph. The frozen 16-tap template transfers across 20 operating points and 4 resolutions, but **whether it transfers across content is untested** — the lab holds one native-resolution photograph. Do not ship the template on this evidence.
- **The two levers overlap** and their gains cannot be added; both attack the same wall bill and were measured separately.
- **The third arm — logistic mixing — was still running when this was written** and is not included.
- Cross-entropy of adaptive binary models, no container, as everywhere else in this study. Comparable to the other numbers here, optimistic in absolute terms.
- `bitsPerEdge = 1.73` in `potts2.go:15` is the measured CAE cost per crack edge, and it drives the RD merge key and the relaxation λ at **every** resolution although it was measured at 512×288. A cheaper wall coder makes it staler still. Re-tuning it would change the partitions themselves, so it was deliberately not touched — a like-for-like number on identical labels was worth more. Anyone pulling that thread must re-run the baseline on the re-tuned partitions too, or they will reproduce falsification #3.

## What this changes

The compression verdict does not move: at 28.7 dB the shape coder goes from 19.3% behind a properly configured WebP to **8.3% behind**, and AVIF remains 30–50% ahead everywhere. What changes is the standing of the coder itself — the component this study had treated as mature turned out to be illegal in one tap and mis-scheduled in a way worth 12.6%, both found by reading it rather than by having ideas about it.

That is the honest summary of the whole reopening so far: **the shape idea is not more competitive than report 08 said, but the shape *coder* was worse than report 08 knew**, and the two roughly cancel.
