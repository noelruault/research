# 33 — Does the format make background removal better?

**Question.** The structured-image pitch says regions are addressable, so "select the background" should be a graph query rather than image processing. Six background-removal photographs were supplied to test it. Does the `.shpc` representation actually make the job better?

**Answer, in one line: it makes the *mask mechanics* dramatically cheaper and edge-exact, and it does not help you decide what the background is — which is the part that was hard.** On these photographs no tolerance separates the subject from the background on either substrate.

> **Partly superseded by report 35.** The failure below is the *algorithm's*, not the representation's. Replace this unsupervised flood with a supervised classifier over a handful of example colours and the chromatic separation works cleanly — and the region graph is then **140–249× cheaper and 3.5–5.9× less fragmented** than the same rule on pixels. What survives from this report is the harder half: colour cannot separate a black ear from dark foliage, and deciding *what* to select still needs semantics. Read the two together.

Data: [`33-background-removal-data.txt`](33-background-removal-data.txt). Verbs: `lab bgcut`, `lab sbs`.

## The experiment

Both arms run the **same** flood algorithm — seed from the image border, absorb a neighbour whose colour is within a tolerance of the seed set's running mean. The only variable is what the flood traverses:

- **Arm R** walks the **region graph** of our partition (1,229 regions).
- **Arm P** walks the **pixel grid** of WebP's decoded output, at matched fidelity.

Same seeds, same tolerance, same acceptance test. Nothing is given to one side and denied the other.

## What the region graph genuinely wins

At tolerance 55, on the 738×414 dog photograph:

| | steps | selection is | frayed edge |
|---|---|---|---|
| **R — region graph** | **3,032** | **633 region ids** | **0** (by construction) |
| P — pixel grid | 731,252 | 232,656 pixels | 6,633 |

**241× fewer decisions**, and the selection is a list of 633 ids rather than a quarter-million pixels. That is the O(regions) claim, measured, and it is not arguable.

"Frayed edge" counts mask-edge pixels whose neighbour across the edge is in the *same* region — places the mask cut through a region's interior. **Arm R's zero is definitional, not evidence**: its mask edges *are* region boundaries. The meaningful figure is arm P's, and it should be read for what it is — the pixel wand disagreeing with the image's own tonal structure in 6,633 places, judged by our partition, which is a biased referee.

## What it does not win, and this is the headline

**Neither arm removes the background.** The failure is not in the substrate, it is in the selection rule, and the format does nothing about it.

| tolerance | region arm | outcome |
|---|---|---|
| 28 | 265 of 1,229 regions, 151,519 px | Background largely **survives** — grass is texture, so it is many regions of drifting colour |
| 55 | 633 of 1,229 regions, 232,656 px | Background mostly gone, but **the dog is being eaten** — belly and legs partly selected |
| 90 | 990 of 1,229 regions, 267,142 px | Almost everything selected |

**There is no setting in between.** A white dog in sunlight and bright sunlit grass are not separable by colour distance, and moving from pixels to regions does not change that — it changes the *unit* the same failing test is applied to. Look at `33-bgcut/dog-bgcut-tol55.png`: the cutout keeps the tree line, drops parts of the dog, and leaves grass fragments.

## What this means for the two niches

**This is a preview of M3, and it is not encouraging for photo editing.** `PREREGISTRATION.md` registered M3 to measure whether the regions are *semantically* meaningful against human ground truth. This report does not run M3 and does not substitute for it — but it is the first direct evidence, and it says the regions follow **tone**, exactly as report 14 found, while background removal needs **objects**. Where tone and object coincide the mask is excellent and free; where they do not, no traversal fixes it.

**The asset niche is untouched by this.** There the regions come from art that is already flat, and the subject is separated by authored alpha rather than discovered by a flood — which is what report 32 and A1b measured, and where the format holds up.

## Fidelity and bytes on these six, for completeness

Matched fidelity, `cwebp -m 6`, our own PSNR definition on both sides, at the capability band (~1,000–1,400 regions):

| image | regions | our PSNR | **ours `.shpc`** | WebP | delta |
|---|---|---|---|---|---|
| images | 1,229 | 29.22 | 11,725 | 7,176 | **+63.4%** |
| images-2 | 1,189 | 25.65 | 11,606 | 9,170 | **+26.6%** |
| images-3 | 1,005 | 28.31 | 6,172 | 3,946 | **+56.4%** |
| images-4 | 1,368 | 24.23 | 12,674 | 10,158 | **+24.8%** |
| images-5 | 1,165 | 24.06 | 13,484 | 9,116 | **+47.9%** |
| images-6 | 1,433 | 29.31 | 10,174 | 5,064 | **+100.9%** |

**WebP wins all six**, consistent with report 23's zero-wins-in-24. Two caveats, both against us: on `images-4` and `images-5` WebP hit its quality floor (`q1`) and **overshot** our fidelity by 1.3 dB and 0.26 dB, so it would be smaller still if it could match exactly — the stated gap understates WebP's advantage. And these are re-encodes of JPEG sources, so all arms inherit the same prior loss.

Visually the failure modes differ as report 04 predicted: we posterise into flat bands, WebP goes soft. See `33-bgcut/dog-fidelity-4panel.png`, whose fourth panel draws our region boundaries — the dog's outline *is* traced, which is why the mask mechanics work, and the grass is shattered into texture regions, which is why the selection does not.

## Caveats

- **Six images, one subject type.** All are consumer background-removal stock: a centred subject on a natural background.
- **The flood is a deliberately simple rule.** A better selector — learned, or seeded interactively, or using region adjacency and area rather than colour alone — might do far better *on the same region graph*. This report tests the substrate, not the ceiling. That the substrate is 241× cheaper is precisely what would make a smarter selector affordable.
- **No comparison against a real background-removal tool** (`rembg`, SAM). Those solve the semantic half this report shows is missing, and the honest comparison is against them, not against a flood fill.
- The source images are JPEG, decoded once to PNG and used identically by every arm.
