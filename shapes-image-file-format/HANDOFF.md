# Handoff — start here

You are picking up a shape-based image format. Everything below is measured and committed; nothing is from memory or conversation. Read this file, then `PRINCIPLES.md` (**what this is for and how it is measured — seven rules, each naming the failure it prevents**), then `PREREGISTRATION.md` (**the active plan — three measurements with their decision thresholds committed before they run**), then `AUTORESEARCH.md` (the running ledger), then `PARKED.md` (set-aside ideas with revive triggers). `DESIGN-ALPHA.md` holds the alpha design study — approach A built and shipping as SHPC v2, approach B documented and reserved. Report 13 holds the capability argument and the older roadmap.

**The single most useful thing to know: the byte-optimisation phase is finished, and it is mostly negative. Do not restart it.** The value now is in making the format *usable* and *demonstrating* the capability that justifies it.

## Where it actually stands

| claim | evidence |
|---|---|
| Poor image codec | Loses to WebP on **all 24 Kodak images**, mean **+27.8%**, range +3.6% to +71.5%, zero wins (report 23) |
| **Good structured-image format** | Beats WebP **+ a region-map sidecar** on all 24, mean **+30.5%**, range +18.7% to +37.8% (report 24). **The only claim here that survived a corpus** |
| It is a real format | **SHPC v1** emits files that round-trip bit-exactly; container overhead ~20 B (report 21) |
| The capability is real | A free client-side mask keeps only **24–40%** of its boundaries across two deliveries of the same image; 10 images, none above half (report 28) |
| Encode cost | **2 m 7 s and 2.89 GB at 4K**, ~10 s at 768×512 (reports 18, 25) |
| **Alpha works** | **SHPC v2** carries per-region alpha end to end; silhouette dissolution 16–62% → **0.00%** at every usable mark. Opaque output byte-identical to v1's (`DESIGN-ALPHA.md`, A1b) |

**The positioning that follows:** if you want pixels, use WebP. If you want pixels **and** a segmentation with stable region identity, this is the cheapest measured way to get both, by roughly 30%. Price it against **raster + sidecar**, never against a raster codec alone.

## Settled — do not re-derive

Thirteen claims were produced by this study and then falsified by it. The register is `06-corrections-and-falsifications.md`. The ones that will waste your time if you re-open them:

- **Bytes vs WebP alone.** Settled, no. Twenty-four images.
- **The RD constants** (`bitsPerEdge`, `bitsPerReg`). Closed in both directions (reports 26, 27). Scaling both is *provably* inert — the merge key is `dSSE/(bitsPerEdge·(l+ratio))`, so uniform scaling cannot change candidate ordering. The ratio only slides along the RD curve; no setting dominates.
- **Parallelising the encoder.** Wrong lever (report 25). The embarrassingly parallel part was the per-mark pricing, and `HDONLY` already deleted it. What remains is a sequential agglomerative merge.
- **"Let the decoder regrow the regions."** Provably a context model bounded by `H(X | causal past)` (report 04). This one does not expire.
- **Illuminant × albedo decomposition.** Raises joint entropy 5.9%; every shuffled control beats the real field (report 12).

## Method rules — these were learned the hard way

1. **Name which knobs each side of a comparison may vary, before the run.** Encoder effort, encode resolution, delivery pipeline. This error was made **twice**, the second time in the session that documented it: a comparison read +87.9% and was a dead heat once both sides got the same knob.
2. **Reproduce the baseline before changing anything.** Every report here matches the published anchor first, then measures.
3. **Check compression levers against a real compressor.** Modelled cross-entropy had the **wrong sign** at four operating points (report 15).
4. **A perfect result is a broken test.** A 100.0000% agreement was a file compared with itself (report 28).
5. **Do not publish a mechanism fitted to three points as a predictor.** One did, and collapsed to correlation +0.005 at n=24 (falsification #13).
6. **Changing the partition invalidates every baseline.** Each arm must build its own and be compared at matched *fidelity*, never by pricing one arm's partition with the other's coder.
7. **Stage explicit paths.** `git add -A` after subagents have run once committed an 11 KB file nobody had read.

## The queue — ranked by what makes this a real option

The old queue optimised bytes. These are ranked by what stands between "a measured claim" and "something a person would choose".

### H1 — Build the capability demo *(highest value, low cost)*

The entire pitch is addressable regions with stable identity, and **nothing exercises them**. There is no tool that opens a `.shpc`, lets you select a region, recolour it, and write it back. Report 13 lists selection, non-destructive editing and cutout animation as the ranked applications; all of them operate on an already-encoded partition, so none is blocked by encode cost.

Everything needed exists: the container decodes (`lab p4dec`), the region adjacency graph is already built by `colorBytes2` as `share[]`, and edits are O(regions). This converts a measured claim into a demonstrable one, and it will find gaps in the format faster than any further measurement.

### H2 — ~~Alpha~~, and a colour-space tag · **alpha DONE, tag still open**

**Alpha shipped** as SHPC v2 mode 1 — per-region flat, round-trips bit-exactly, v1 files still decode, and v2 costs exactly one byte more on an opaque image. Mode 2 (per-pixel plane) is reserved in the header and rejected by the decoder until A3 shows real game art needs it. See `DESIGN-ALPHA.md`.

**Still missing: colour-space tag and metadata.** Neither blocks a game asset the way alpha did, so both are smaller than they look.

### H3 — A decoder outside Go

There is one decoder and it is Go. No WASM, no browser path, no C. Adoption is impossible without one, and the decoder is the cheap half — decode is a fill operation with no entropy-decode-then-inverse-transform pipeline.

### H4 — Truncation and progressive decode

Report 13 claims a truncatable progressive stream as a **strength**, and it is not implemented. The nested hierarchy already exists in `hdMarks` — the merge produces valid coarser partitions at every mark. This is the one claimed capability that is currently vapour, so either build it or strike it from report 13.

### H5 — Memory: 2.89 GB at 4K

Never profiled, only measured. Driven by the merge's per-pixel structures. This is what excludes mobile and any multi-tenant encoder, and `HDONLY` did not touch it. At 768×512 it is not a problem, so it only blocks 4K work.

### H6 — Finish report 28's Test 2 on the corpus · **now M1, pre-registered**

Test 1 (mask stability across deliveries) is 10 images. Test 2 (agreement with the transmitted partition) is **2**, and is labelled indicative. It supports the same claim and is cheap.

### H7 — A different segmenter for the free-mask client · **now M2, pre-registered**

Report 28 gave the client *our own* merge, which bounds free-mask agreement from **above**. SLIC or Felzenszwalb would agree less and would strengthen a finding already measured in its generous case.

### H9 — Boundary recall against human ground truth · **now M3, pre-registered — the one that decides the niche**

Report 14 judged the regions meaningful **by eye, on three windows of one photograph**, and every photo-editing application in report 13 rests on that. H6 and H7 harden a claim already held; neither measures whether the regions are *right*. This one does, against the strongest freely available automatic segmenter, and it is the only queued item that can go against us.

**Metric and thresholds are committed in `PREREGISTRATION.md` before the run** — including why the primary metric is boundary recall plus under-segmentation error rather than the conventional F-measure, and why the F-measure gets published anyway.

### H8 — P8b: coarser initial partition *(low value, real hazard)*

The merge starts at one region per pixel — ~8.28M sequential steps at 4K. A superpixel pre-pass would cut that by orders of magnitude. **It changes the partitions**, so every baseline needs re-running or it reproduces falsification #3. Speed only; touches no surviving claim.

## Where things are

- **Reports** `01`–`28` in this directory, each with a `*-data.txt` companion holding raw output and the command that produced it.
- **Ledger** `AUTORESEARCH.md` — newest entry first, includes what each iteration got wrong.
- **Parked** `PARKED.md` here and `../PARKED.md` for the convention. Every entry carries what it depends on and a concrete revive trigger. **Re-read it whenever a baseline moves** — that is when a parked idea silently becomes good again.
- **Code** `code/lab` — builds and vets clean, `go test` passes. Verbs: `hd` (encode a scale-space; `HDONLY=<n>` for one mark), `p4enc`/`p4dec` (container), `rcdec`, `wallcheck` (decoder-side causality), `wallxexact`, `recolour`, `floor`.
- **Corpus** Kodak-24 is not committed; fetch from `r0k.us/graphics/kodak/`. The original eval image and its renders live in the session scratchpad and are reproducible with `lab hd`.

## What to do first — decided 2026-07-30, see `PREREGISTRATION.md`

**Evidence first, niche after.** The format has two candidate niches — authored assets (gaming, animation) and photo editing — and the record cannot currently choose between them. Rather than pick one and build, run the three measurements that decide it, with thresholds committed before they run.

1. **M1 and M2** (H6, H7) — cheap, code exists, same corpus. They harden report 24's baseline.
2. **M3** (H9) — the niche decision. New corpus, a comparison arm to verify rather than assume, a day of work.
3. ~~**Phase 1 — SHPC v2: alpha**~~ — **ALPHA IS DONE.** Approach A (per-region flat) built and shipping as SHPC v2 mode 1; A1 and A1b both run and both retracted the fix the study had proposed. **The colour-space tag is still open** and is now the smallest remaining Phase 1 item.

**H1 stays queued, not cancelled.** The earlier recommendation was to build the region editor first, on the reasoning that a demo finds format gaps faster than more measurement. That still holds — it is *which* demo that is unresolved, and M3 is what resolves it. A photo-editing demo and an asset-pipeline demo are different products.
