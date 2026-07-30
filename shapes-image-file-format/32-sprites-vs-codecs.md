# 32 — SHPC v2 against shipping codecs, on game sprites

**Question.** Alpha is built and correct (`DESIGN-ALPHA.md`, A1b). Is it any *good*? Every byte number in this study is from photographs, where report 23 settled that the format loses to WebP on all 24 Kodak images. Game sprites are the niche this format is actually aimed at, and nobody had measured one.

**Answer. One win, two losses against WebP lossless; three wins of three against PNG and against AVIF lossless.** Not a general win, and the losses have a clear cause.

Data and commands: [`32-sprites-vs-codecs-data.txt`](32-sprites-vs-codecs-data.txt).

## This turned out to be a lossless comparison, which was not the plan

At the finest mark the scale-space has nothing to merge — `hdMarks`' finest stop (940 for ak74) is above the exact partition's region count (813), so `runRD` never runs and the render *is* the source.

Rather than assume that, `p4dec` was pointed at the **original sprite** as its reference instead of at the render. It reports EXACT on all three, alpha included. So every arm below stores pixel-identical content and **no fidelity matching was needed**.

## The numbers

| sprite | px | **OURS `.shpc`** | WebP `-z 9` | PNG (Go) | AVIF `--lossless` | as-authored PNG |
|---|---|---|---|---|---|---|
| ak74 | 2,352 | **1,831** | 1,938 | 2,617 | 3,605 | 5,217 |
| bow | 3,420 | **1,310** | 1,054 | 2,397 | 3,114 | 4,915 |
| pickaxe | 400 | **345** | 218 | 466 | 1,242 | 506 |

Relative to ours, negative meaning we are smaller:

| sprite | vs WebP-ll | vs PNG | vs AVIF-ll | vs as-authored |
|---|---|---|---|---|
| ak74 | **−5.52%** | −30.04% | −49.21% | −64.90% |
| bow | +24.29% | −45.35% | −57.93% | −73.35% |
| pickaxe | +58.26% | −25.97% | −72.22% | −31.82% |

**The as-authored column is not a codec comparison** and must not be quoted as one — those are the unoptimised PNGs sitting in the sprites repo. It is here because it is what that project ships today, which is a different and also useful number.

## Reading the split

- **ak74 wins.** It is the largest and most colour-varied — wood stock, metal body, several distinct parts. More regions, each genuinely flat, which is the content this representation is for.
- **pickaxe loses worst (+58%) because it is 400 pixels.** Our ~21 B header plus a 61 B wall chunk is most of the file; WebP's ~44 B floor is simply smaller. That is small-image overhead, not a coding failure, and it would shrink as a share of any real asset.
- **bow loses (+24%) on alpha specifically.** It is the softest-alpha sprite of the three (18.25% soft, A3 pilot) and its **alpha chunk is 540 B against a 519 B colour chunk** — alpha is the single largest component of that file. Approach A stores one flat value per region, so a soft gradient has to be paid for in extra regions and extra boundary.

That last point is the first measured argument for anything in `DESIGN-ALPHA.md`'s deferred list — a per-region alpha **ramp** (idea D) or the per-pixel plane (approach B) would both attack exactly the cost bow is paying. It is one sprite, so it is a pointer, not a mandate.

## What this does not say

- **n = 3**, from one project, 400–3,420 px. Not a corpus. Every one is small enough that container overhead is a visible share.
- **Lossless only.** The format's actual pitch — addressable regions at a few hundred primitives — lives at the **coarse** marks, and these numbers say nothing about them. A matched-fidelity lossy comparison against WebP-with-alpha has not been run.
- **Bytes were never the claim** (`PRINCIPLES.md` #5). Winning one sprite of three is consistent with flat art being friendlier content than photographs. It is not evidence of a general win, and report 23 stands unchanged.
- **Timing was measured and is not reportable.** At this size both sides are dominated by process startup, and our decoder shells out to an external `brotli`. That measures `fork`/`exec`, not decoding. A real benchmark needs in-process timing on a corpus large enough for the work to clear the noise.
- **AVIF lossless is a weak arm at these sizes.** Its losses here are not a general statement about AVIF, which beats this format by 30–50% on photographs (report 05).
