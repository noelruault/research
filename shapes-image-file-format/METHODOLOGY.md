# Measurement methodology

Every comparison protocol and metric this study uses, what each is valid for, and **what confounds it**. `PRINCIPLES.md` says how to behave; this says how to measure. Reach for it before designing a run, and again before publishing a number.

The traps listed here are not hypothetical. Each one was walked into, in this repo, and cost a published claim.

> **Incomplete on purpose.** Two consultant briefs are out on (a) comparison against real background-removal tools and matte corpora, and (b) format-comparison rigour — BD-rate, perceptual metrics, a modern corpus. Their protocols get merged here when they land. What follows is what has actually been run and validated.

---

## 1. Comparison protocols

### 1.1 Matched-fidelity byte comparison

The default for "is our file smaller". Reports 16, 17, 23, 33.

**Protocol.** Encode ours at a chosen operating point. Measure its PSNR. Sweep the rival's quality parameter upward and take the **smallest setting whose PSNR reaches ours**. Compare bytes at that point.

**Knobs, named in advance.** The rival gets max effort (`cwebp -m 6`), the same encode resolution, and the same delivery pipeline. Any knob given to one side is given to the other. PSNR uses *this project's* definition (`lab psnr`) on both sides, never the rival's own metric.

**Traps.**
- **Floor overshoot.** If the rival bottoms out (`q1`) *above* our fidelity, it is over-delivering quality and would be smaller still if it could match exactly. **The stated gap then understates the rival's advantage** — say so. Happened on two of six images in report 33.
- **Steelmanning one side only.** Falsification #11, committed twice. A comparison read +87.9% and was a dead heat once both sides got the same resampling knob.

### 1.2 Lossless identical-content comparison

Cleanest comparison available, when it applies. Report 32.

**Protocol.** When the finest scale-space mark exceeds the exact partition's region count, `runRD` never merges and **the render is the source**. Every arm then stores pixel-identical content and no fidelity matching is needed at all.

**Verify, do not assume.** Point `p4dec` at the **original** as its reference, not at the render. It reported EXACT on all three sprites, which is what turned an assumption into a fact.

**Trap.** PNG byte streams differ between source and render even when decoded pixels are identical, because encoders pick different filters. Compare decoded pixels, never file bytes.

### 1.3 Two-arm substrate comparison

For "does the region graph help?". Reports 33, 35.

**Protocol.** Run the **identical** algorithm on both substrates — our region colours, and the rival's decoded pixels at matched fidelity. The only variable is what gets traversed or classified.

**Trap, and it cost a headline (falsification #14).** Our region colours are **already spatially averaged by the partition**; raw pixels are not. Comparing them directly measures *averaged vs not averaged* as much as it measures the substrate. **Give the pixel arm the regularisation a practitioner would apply** — a majority/median filter — and apply it to both arms. Doing so took a 3.5–5.9× claim down to 1.0–1.7×.

**The deeper version, still not run.** Give the pixel arm its *own* partition (SLIC, Felzenszwalb, watershed) and classify that. It isolates "our segmentation" from "having any segmentation at all", and it is the honest apples-to-apples.

### 1.4 Cross-source segmentation

Partition from one image, colours from another. Report 37, verb `lab crosseg`.

**Protocol.** Merge on source A, price and paint from source B, and run the baseline — same target segmented on B directly — so both are compared at matched region count against the same reference.

**Established result:** catastrophic for a *file* (−10.82 dB to save 10.8% bytes), free for a *mask*. **A partition must match the colours it will carry.**

### 1.5 Baseline choice

**Price against what a consumer would actually assemble.** For a format carrying pixels *and* a segmentation, the baseline is **raster + a region-map sidecar**, never a raster codec alone. Both numbers get published (`PRINCIPLES.md` #5): a poor image codec, a good structured-image format.

Price the sidecar with the **strongest** available coder. Ours beat raw labels + `xz -9e` by 2.46× (report 19), so using it is conservative rather than self-serving.

---

## 2. Metrics, and what confounds each

| metric | measures | valid for | **confounded by** |
|---|---|---|---|
| **RGB PSNR** (`lab psnr`) | fidelity | matched-fidelity comparison | Not perceptual. At 24–26 dB our posterisation and WebP's blur are not comparable failures (report 04) |
| **boundary Jaccard** | mask identity | stability across deliveries (report 28) | Unforgiving by construction: a one-pixel shift scores as a full miss. Measures identity, not similarity |
| **pair agreement** | mask overlap | nothing, as a headline | Dominated by interior pixel pairs; reads 85–87% for masks sharing only 22–41% of boundaries. **Never the headline** |
| **dissolved crossings** (`lab silhouette`) | silhouette survival | A1/A1b | Needs the `invisible` companion column, or it blames the merge for an upstream loss |
| **blob count** (connected components) | mask fragmentation | cleanup burden | **Requires a regularised rival arm** (§1.3). Structurally favours a piecewise-constant mask |
| **edge fidelity dE** (`lab bgclass`) | does the mask edge sit on real image structure | **between arms on one image** | **Scales with the image's own contrast. NOT comparable across images** — report 36's cross-image row was withdrawn for exactly this |
| **frayed edge** (`lab bgcut`) | mask cutting through region interiors | the rival arm only | **Our arm's zero is definitional**, and our partition is a biased referee |

### The referee rule

**Never score a comparison on our own partition.** Report 33 measured "frayed edge" against our partition and flagged it; report 35's blob metric repeated the mistake in spirit and had to be retracted.

Score against the **source image** instead — that is what edge-fidelity dE does, and neither arm owns it.

### The metric-choice rule

**Choose the metric before the run, and publish the unflattering one alongside.** Where a metric is chosen because it suits the format — e.g. boundary *recall* over F-measure for selection, since over-segmentation is free and under-segmentation is fatal — the reasoning is registered in advance **and the conventional metric is reported anyway**. Reporting only the flattering metric makes the flattering metric worthless.

---

## 3. Pre-run checklist

1. **Name every knob** each side may vary. Write it down before running.
2. **Choose the metric and the threshold** before the data exists. A threshold picked afterwards is not a threshold.
3. **Ask who the referee is.** If it is our own data structure, find a neutral one.
4. **Ask what the rival would do in practice.** Not the naive version — the version a practitioner ships. Falsifications #11 and #14 are both this failure.
5. **Reproduce the baseline** before changing anything.
6. **Run it twice.** Three separate nondeterminism bugs here came from Go's randomised map iteration.
7. **Check the exit status you think you are checking.** `cmd 2>&1 | head && echo OK` reports `head`'s status.
8. **Bash, not zsh.** The one zsh loop here mis-split `set -- $pair` and printed a table of zeros.

## 4. Post-run checklist

1. **A perfect score is a broken test.** 100.0000% agreement was a file compared with itself.
2. **A flat number beside moving ones is a bug**, not a finding. An unchanged 15/15 caught a render that had silently dropped its alpha.
3. **A plateau is not a trend.** Dissolution stuck at exactly 259 across three marks was information loss upstream, not merge behaviour.
4. **Attack the claim that flatters us hardest**, before publishing rather than after.
5. **Sanity-check sampled inputs.** Two "keep" points landed on grass and were caught only by printing their RGB values. Probe first.
6. **State what the number does not show**, in the report, not in a footnote.

## 5. Reproducibility

Every number in a report has a `*-data.txt` companion holding raw output **and the command that produced it**. Runnable scripts live in [`code/runs/`](code/runs/). Tools that a comparison depends on are checked for up front and the script **exits** rather than silently skipping an arm — a missing `cwebp` once meant an empty column rather than an error.

Corpora are not committed: Kodak-24 from `r0k.us/graphics/kodak/`, sprites from `noelruault/sprites`, the background-removal photographs were session-supplied and are not in the repo.
