# Principles

Seven. Six are what this study enforced on itself and paid for learning; the seventh is what it is for.

They are here to be applied against specific decisions, so each one names the failure it prevents and where that failure actually happened. A principle with no failure attached is decoration.

---

## 1. Every claim carries a number and the command that reproduces it

Not memory, not conversation, not a plausible argument. Each report in this directory has a `*-data.txt` companion holding the raw output and the invocation that produced it, and the code lives in `code/lab` so any figure can be re-derived.

**Prevents:** the number that everyone believes and nobody can rebuild. `code/lab` once shipped broken because independent work was committed without compiling it.

## 2. Name the knobs, the metric, and the threshold before the run

Encoder effort, encode resolution, delivery pipeline — every knob given to one side must be offered to the other. This extends past knobs: **which metric decides, and what result would change our minds, are chosen before the data exists.** A threshold picked after seeing the numbers is not a threshold. `PREREGISTRATION.md` is this principle in practice.

**Prevents:** falsification #11, made twice. A comparison read +87.9% and was a dead heat once both sides got the same knob. The second time was in the session that documented the first.

## 3. A perfect result is a broken test

A score at the boundary of its range gets checked before it gets written down.

**Prevents:** report 28's first run, which reported 100.0000% agreement. It was a file compared with itself — the "different quality" arm had selected the same quality.

## 4. Falsifications are published, not deleted

Thirteen claims produced by this study were then killed by it, and all thirteen are in `06-corrections-and-falsifications.md` with the measurement that killed each. One had already been published as a README headline. Ideas killed on arguments that do not expire go on the killed list; ideas killed on numbers that could move go in `PARKED.md` with a revive trigger.

**Prevents:** the study's most common failure mode, which is re-deriving a dead idea. It also makes the surviving claims worth something — a record that only shows wins is not evidence.

## 5. Structure is the product; pixels are the by-product

Against WebP alone this format loses on all 24 Kodak images. Against WebP **plus a region-map sidecar** — what a consumer would actually assemble to get the same capabilities — it wins on all 24, by a mean of 30.5%. Both are true, and the pitch says both. If someone only wants pixels, tell them to use WebP.

**Prevents:** the twelve reports that optimised bytes against a baseline that did not match the product. Saying the loss out loud is what makes the win credible.

## 6. Determinism is a feature, not an implementation detail

Same input, same bytes, same region ids, forever. Region #4,211 means the same thing on every device and after every re-encode. That is the capability the format sells; a client re-segmenting decoded pixels keeps only 24–40% of its boundaries across two deliveries of the same image.

**Prevents:** three separate nondeterminism bugs from Go's randomized map iteration, two of which survived the first fix and were found months later. For a format whose output is committed art, a different part set per run is not a tolerable property.

## 7. Other formats inspire us; they do not set our bar

PNG, WebP, AVIF, JPEG XL are read for ideas and used as honest comparisons. They are not a specification to reimplement. **When a design question comes up, ask what a format made of shapes should do — not what a format made of pixels already does.**

PNG has an alpha channel because a grid of pixels has nowhere else to put coverage. That is a consequence of its representation, not a requirement we inherit; here a transparent area is a region you do not draw, and an anti-aliased rim may be something the decoder computes rather than something the file stores. That reasoning is in `DESIGN-ALPHA.md`, and it only exists because the question was asked from scratch.

**The initial target is game assets**, where the properties this format has — stable region identity, no decode step, primitive counts a shape runtime can draw, exact silhouettes — are the properties that matter, and where no existing format is trying to do this at all.

**The one honest exception, and it is not a loophole.** Where a *free alternative competes for the same user*, that is product competition, not bar-setting, and it has to be measured. A photo editor's alternative to our shipped mask is running a segmenter locally for nothing. That is why M3 in `PREREGISTRATION.md` compares against the strongest freely available segmenter and lets the answer decide whether the photo-editing niche is occupied. Refusing to check that would not be pioneering; it would be avoiding the result.
