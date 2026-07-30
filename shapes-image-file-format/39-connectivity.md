# 39 — Connectivity: the foreground collapses to one piece

**Question.** The owner's observation: *stuff not connected to the centred object is usually background and can be discarded.* Does it hold, and does it fix the failures reports 35–37 left standing?

**Answer. Yes, and dramatically — the foreground becomes exactly one connected piece on both test images and both substrates.** It is also **not a format advantage**: connectivity is a graph operation available on any substrate, and the pixel arm gains just as much.

Data: [`39-connectivity-data.txt`](39-connectivity-data.txt). Images: [`39-connectivity/`](39-connectivity/). Disable with `CONN=0`.

## Two operations, two different failures

The observation turns out to be two things, and they fix different problems:

1. **Hole filling.** A background component that never touches the image border is *enclosed* by the subject, so it is almost certainly a misclassified part of it. Flip it. → fixes **the bobcat's dark markings** and the dog's eyes.
2. **Disconnected rejection.** Of what remains kept, keep only the component containing the subject seed. Anything not attached to the subject is background however much it looks like it. → fixes **the dog's floating tree blobs**.

**Applied to both arms with identical code.** Falsification #14 was giving one arm a cleanup the other did not get; that is not repeated here.

## The result

| bobcat | removed px | bg blobs | fg blobs |
|---|---|---|---|
| region, raw | 99,404 | 133 | 11 |
| pixel, raw | 95,462 | 344 | 191 |
| **region + connectivity** | 96,955 | **10** | **1** |
| **pixel + connectivity** | 111,035 | **8** | **1** |

| dog | removed px | bg blobs | fg blobs |
|---|---|---|---|
| region, raw | 246,489 | 44 | 33 |
| pixel, raw | 248,211 | 187 | 264 |
| **region + connectivity** | 257,901 | **4** | **1** |
| **pixel + connectivity** | 266,781 | **5** | **1** |

**The foreground collapses to exactly one piece** in every case. Background blobs fall 133 → 10 and 44 → 4 on the region arm. The dog's removed-pixel count *rises* by 11,412 — that is the detached tree blobs being discarded, which is the operation doing its job.

## It works, and it is not ours

After the pass the two arms are level: **11 vs 9 blobs** on the bobcat, **5 vs 6** on the dog. Connectivity is a graph operation on a mask, and any substrate can run it.

**So this does not add to the format's case.** What survives from report 35 (as corrected) is unchanged: **cost** — 140–249× fewer decisions — and **edge fidelity**. Not fragmentation.

One genuine asymmetry, unmeasured here: on the region arm both operations run on the region adjacency graph at **O(regions)** rather than O(pixels). They were computed on the pixel mask so both arms run identical code and the mask comparison stays fair. The cost claim is separate and this run does not measure it.

## The limit, visible in both pictures

**Connectivity cannot reject background that touches the subject.**

- **Dog** — the detached tree blobs on the right are gone. The tree mass directly behind the ear **survives**: it is one connected component with the dog.
- **Bobcat** — the branch crossing in front of the cat survives for the same reason.

So: disconnected background is free to remove; connected background is untouched. On these two photographs that removes most of the error, and **what remains is exactly the achromatic collision at the point of contact** — which is what report 38's roadmap targets with a defocus feature and a graph cut.

## Caveats

- **Two images.** The seed is the centroid of the keep examples; if it lands on a removed pixel the code falls back to the largest kept component, which is a heuristic and untested.
- **"Background touches the border" fails** for a subject cropped at the frame edge, and for a background fully enclosed by the subject. Neither occurs here.
- **Keeping only the seed component discards legitimately detached subject parts** — a thrown ball, a second animal, an arm behind an occluder. A strong assumption, stated as a heuristic rather than a rule. `CONN=0` turns it off.
- **No ground-truth masks**, so these remain fragmentation counts, not accuracy. Report 38's item S0 is what fixes that.
