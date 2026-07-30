# 35 — Supervised colour separation: the region graph makes a cheap rule clean

**Question.** Report 33 concluded that the format "does not help you decide what the background is". That was tested with an *unsupervised* flood — seed from the border, absorb neighbours within a drifting RGB tolerance — and it failed on both substrates. The obvious objection: that is a bad algorithm, not a verdict on the representation. If a user supplies colour information — *this is grass, remove it; this is the dog, keep it* — does colour separate them, and does the region graph help?

**Answer. Yes on both counts, with one hard limit.** Chromatic separation works cleanly once examples are supplied. The region graph makes the same rule **140–249× cheaper**, and its mask edges sit on **28–54% larger colour steps** judged against the source. What colour cannot do is separate a black ear from dark foliage, and no substrate fixes that.

> **Corrected — falsification #14, and a reader caught it.** This report first headlined a **3.5–5.9× reduction in mask fragmentation**. That comparison was not fair: arm R classifies region colours the partition has *already spatially averaged*, while arm P classified raw pixels with **no cleanup at all**, which nobody ships. Given a majority filter on both arms the advantage falls to **1.0–1.7×** — on the bobcat at 5×5 it is a dead heat — and the fragmentation headline is **retracted**. What survives is the decisions ratio and the edge-fidelity result below, the latter measured on a neutral referee rather than on our own partition. The corrected tables are in the section "The steelman, and what it costs the claim".

Data: [`35-background-removal-supervised-data.txt`](35-background-removal-supervised-data.txt). Verb: `lab bgclass`. Images: [`35-bgclass/`](35-bgclass/).

## The change from report 33

Everything is held identical except the selection rule.

| | report 33 | this report |
|---|---|---|
| rule | flood from border, drifting RGB tolerance | **1-nearest-neighbour over supplied examples** |
| space | RGB | **CIELAB** |
| supervision | none | a few points the user touches |

CIELAB matters: a shadowed white hair and a sunlit blade of grass are close in RGB *magnitude* and far apart perceptually. That is exactly the confusion that sank the flood.

Both arms run the **identical** classifier on the **identical** examples. The only variable is what gets classified — our region colours, or the rival's decoded pixels.

## It works

![dog, 7 keep and 11 remove examples](35-bgclass/dog-7keep-11remove.png)

*Source | per-region cut | per-pixel cut | disagreement in red. The grass is gone and the dog is intact — the thing report 33 could not do at any tolerance.*

Mask agreement between the two arms rises from **61.55%** under the flood to **91.96–95.40%** under supervision. Both substrates now largely agree, because the rule is finally right.

**Report 33's pessimism was about the algorithm, not the representation.** That correction belongs on the record.

## The region graph wins twice

| run | arm | decisions | total blobs |
|---|---|---|---|
| dog, 8 examples | **region** | **1,229** | **75** |
| | pixel | 305,532 | 263 |
| dog, 18 examples | **region** | **1,229** | **77** |
| | pixel | 305,532 | 451 |
| bobcat, 15 examples | **region** | **1,433** | **144** |
| | pixel | 199,995 | 535 |

**Cost:** 140–249× fewer decisions. A region has one colour, so classifying the image means classifying ~1,200 things instead of ~300,000.

**Fragmentation, as first published and now retracted as a headline:** "blobs" counts connected components of each mask value. Sharpening the dog classifier from 8 to 18 examples removed 45,081 more background pixels and took per-**pixel** blobs **263 → 451** while per-**region** stayed **75 → 77**. That looked like a property of the representation. It is largely a property of *one arm having had a smoothing step and the other not* — see the correction below.

## The steelman, and what it costs the claim

A majority (median) filter on the mask is what any practitioner applies to a per-pixel classification. Applied to **both** arms, so neither gets a knob the other is denied:

| image | no filter | 3×3 | 5×5 | 7×7 |
|---|---|---|---|---|
| dog — region vs pixel blobs | 77 vs 451 (**5.9×**) | 64 vs 162 (2.5×) | 61 vs 111 (1.8×) | 52 vs 89 (**1.7×**) |
| bobcat — region vs pixel blobs | 144 vs 535 (**3.7×**) | 134 vs 180 (1.3×) | 107 vs 110 (**1.03×**) | 74 vs 88 (1.2×) |

**On the bobcat at 5×5 the advantage is gone.** The fragmentation claim does not survive a steelmanned pixel arm.

### What does survive: edge fidelity, on a neutral referee

Report 33 scored "frayed edge" against *our own partition*, which is a biased referee — flagged there, and repeated in spirit by the blob metric. So this scores against the **source image** instead: the mean CIELAB step across the mask edge. A mask sitting on genuine image edges scores high; one cutting through flat areas scores low. Neither arm owns that referee.

| image | arm | no filter | 3×3 | 5×5 | 7×7 |
|---|---|---|---|---|---|
| dog | **region** | **5.47** | **4.60** | **4.12** | **3.82** |
| dog | pixel | 4.04 | 3.27 | 3.05 | 2.99 |
| bobcat | **region** | **11.51** | **9.86** | **8.35** | **7.23** |
| bobcat | pixel | 7.47 | 6.67 | 5.79 | 5.37 |

**The region arm's mask edges sit on 28–54% larger colour steps at every setting, on both images**, using ~30% fewer edge pixels — a shorter and more decisive boundary. And the filter *degrades* edge fidelity on both arms: it buys smoothness by cutting through real edges, which is the trade a region mask does not have to make.

**The decisions ratio is untouched and the filter sharpens it.** The pixel arm *needs* the cleanup to be competitive on fragmentation (451 → 89); the region arm barely moves (77 → 52) because there was little to clean. So the pixel path costs 305,532 classifications **plus** a 7×7 majority pass; the region path costs 1,229 classifications and no pass.

## The hard limit, stated plainly

![bobcat](35-bgclass/bobcat-7keep-8remove.png)

Colour cannot separate things that are the same colour. The dog's black ear is `rgb(42,44,39)`; the foliage behind it runs `rgb(10,16,16)` to `rgb(54,60,60)`. The bobcat's dark markings sit inside the range of its own blurred background. So the dog cut keeps part of the tree line, and the bobcat cut has holes in the animal.

**No set of colour examples fixes this, on either substrate.** It is exactly the residue a learned model buys: iOS Lift Subject uses shape and semantics, not a colour lookup. This report does not claim to match it and did not test against it.

The honest division of labour that follows:

- **Deciding *what* to select needs semantics.** Colour gets you the chromatic majority; the achromatic collisions need a model.
- **Executing the selection is where this format is strong** — 140–249× cheaper, mask edges on 28–54% larger colour steps, and no cleanup pass needed.

And those compose: a model that outputs *region labels* rather than a pixel mask inherits the cheapness and the exact edges. **The 140–249× is what makes an expensive model affordable to run per-region.**

## Caveats

- **Two images**, one subject type: a centred animal on a natural background.
- **Examples were hand-picked by the author while looking at the picture.** A real gesture supplies one touch point, not seven. How this degrades with fewer or worse examples is untested, and it is the first thing to measure next.
- **Blobs measure fragmentation, not accuracy**, and the unfiltered blob comparison was unfair — see the correction above. There is no ground-truth mask for these images, so nothing here is an accuracy claim.
- **Edge fidelity is a proxy too.** A mask edge sitting on a large colour step is evidence it follows real structure, not proof it follows the *right* structure. A confidently wrong boundary also scores well.
- **No comparison against rembg, SAM or Lift Subject.** Still the honest next comparison, and still not run.
- Both arms sit downstream of a lossy encode: our partition at the capability mark, WebP at matched fidelity.
