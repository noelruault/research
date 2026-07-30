# 28 — The free mask is a different mask, and it moves when the delivery moves

**Question.** Report 24 is the study's one surviving claim: pay ~30% more than WebP and get pixels *plus* a segmentation, cheaper than assembling the same thing from a raster codec and a sidecar. The obvious objection was never tested — **a client can segment the decoded pixels itself, for free**, so the mask costs nothing and report 24's baseline is wrong. Report 13 answers that a *transmitted* partition is deterministic and identical on every client, but that is an argument, not a measurement.

**Answer.** The free mask is genuinely free and genuinely different. Across two deliveries of the *same image*, only **24–40% of its boundaries survive**, and it agrees with the transmitted partition on **22–41%**. Report 24's baseline holds for any use needing stable region identity, and does not apply to one-shot local selection.

Knobs, and they are deliberately generous to the free-mask side: the client is given **this study's own merge** — the strongest segmenter available here, and one whose nondeterminism was fixed in falsification #6 — run on the WebP-decoded pixels at matched fidelity. A real client using a different library, version or threshold would agree *less*, not more.

## Test 1 — is the client's mask stable across delivery?

The same source image, delivered as two different WebP files (a matched-fidelity encode and a re-encode at another quality — what any CDN transcode or `srcset` pipeline produces), each segmented client-side:

| | kodim01 | kodim23 |
|---|---|---|
| regions from delivery A | 4,604 | 4,374 |
| regions from delivery B | 5,135 | 4,313 |
| pair agreement | 86.90% | 84.92% |
| **boundary Jaccard** | **0.4015** | **0.2383** |

**Between a quarter and two-fifths of the boundaries are shared.** Region counts differ by up to 11.5%. There is no correspondence between region #4,211 in one delivery and any region in the other.

## Test 2 — does the client's mask match the transmitted one?

| | kodim01 | kodim23 |
|---|---|---|
| pair agreement | 87.38% | 84.15% |
| **boundary Jaccard** | **0.4069** | **0.2213** |

The client-derived mask and the shipped partition are **different segmentations of the same picture**. Both are defensible; they are not interchangeable.

**Read the pair-agreement column carefully.** 85–87% looks like near-agreement, but it is dominated by interior pixel pairs, which are non-boundary in every plausible segmentation. The boundary Jaccard is the number that matters, and it is 0.22–0.41.

## What this settles, and what it does not

**Report 24's comparison holds** wherever region *identity* must be stable — the same region referring to the same thing across clients, across re-encodes, and across time. That covers per-region animation keyframes, editing round-trips, and anything where a mask is authored once and consumed elsewhere. A client-side mask cannot provide it: it is a function of the delivered bytes, and the delivered bytes change.

**Report 24's comparison does not apply** to a consumer doing one-shot local selection where any plausible segmentation is acceptable — "roughly select the sky in this image, now, on this machine." For that use the mask really is free and paying ~30% for a transmitted one buys nothing.

That is a narrower claim than report 13 made, and it is the first time the capability argument has a number attached instead of an assertion.

## A methodology note, because the first run was wrong

The initial test reported **100.0000% agreement** for kodim23 across two qualities — implausible, and checked rather than published. The cause: report 23's matched-fidelity search had selected `q45` for that image, and the "different quality" arm was also `q45`, so the script compared a file with itself (`md5` identical, both 14,986 B). Re-run against a genuinely different delivery (`q70`, 20,702 B), agreement falls to 84.92% and Jaccard to 0.2383.

A perfect score is evidence of a broken test far more often than of a perfect result.

## Caveats

- **Two images.** The effect is large and consistent in direction, but this is not the corpus report 24 rests on.
- **One segmenter on both sides.** Using our own merge for the client is the generous case; it bounds free-mask agreement from *above*. The realistic case — SLIC, Felzenszwalb, SAM, or a different build of any of them — is untested and would be worse.
- **Boundary Jaccard is unforgiving by construction**: a one-pixel shift in an otherwise identical contour scores as a full miss on both sides. It measures identity, which is the thing under test, but it is not a perceptual similarity measure and should not be read as one.
- The regions were matched by nearest region count, not at matched fidelity, since the client has no fidelity target — it segments what it receives.
