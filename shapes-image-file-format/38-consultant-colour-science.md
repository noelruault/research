# 38 — Consultant report: defocus is the lever, graph cut is the machinery

**External expert input, commissioned 2026-07-30**, on the question report 35 left open: colour cannot separate a black ear from dark foliage, so what does a professional actually do?

**This is not a measurement report.** It is a diagnosis plus a roadmap, recorded so the reasoning is available later. The one live measurement it contains **has been independently reproduced** — see the verification section, which is the first thing to read.

---

## Verification, before anything else

The report's load-bearing claim is that the exact pair defeating the colour rule differs enormously in local high-frequency energy. Re-measured independently on `images.png` (21×21 window, mean squared 4-neighbour Laplacian of luminance, `scratchpad/lapcheck.go`):

| probe | rgb | lap² |
|---|---|---|
| **black ear** (370,60) | `rgb(42,44,39)` | **777.0** |
| **dark foliage** (550,40) | `rgb(14,18,17)` | **12.0** |
| tree line (620,60) | `rgb(96,99,92)` | 55.6 |
| grass, near/in-focus (100,350) | `rgb(124,139,56)` | 1303.1 |
| grass, far/blurred (600,300) | `rgb(133,146,64)` | 132.8 |
| dog white body (250,150) | `rgb(222,216,202)` | 547.6 |

**Every Laplacian figure reproduces to the decimal. The 65× ratio on the collision pair is real.**

**One discrepancy, and it does not affect the finding.** The consultant reported the ear as `rgb(26,27,25)`; it is `rgb(42,44,39)` — the value this study has used since report 35. An incidental colour readout, probably probed off the composite panel rather than the source. The energies, which are the claim, are identical.

**A caveat the report states and the numbers confirm.** In-focus grass scores **1303.1** — *higher* than the dog's body at 547.6. **Focus alone does not separate foreground from background here.** It separates the dark pair that colour cannot; colour separates the chromatic pair that focus cannot. The two cues fail on disjoint pairs, and that complementarity is the whole design. Anyone reading "defocus is the lever" as "use defocus instead of colour" has misread it.

Still one window per class, on one photograph. The per-region version is item S1 below and is pre-registered before anything is believed.

---

## The diagnosis

**Achromatic collision is not a colour problem with a missing trick — it is information-theoretic.** Any pointwise colour transform (tone curve, colour space, chroma stretch) is a deterministic function of a single pixel value, so overlapping class-conditional colour distributions stay overlapping afterwards. Reports 36 and 37 measured the corollary twice: the collision *moved* rather than resolved.

Separating information can come from exactly three places:

1. **Neighbourhoods** — texture, focus. *Our corpus is unusually favourable: strong bokeh.*
2. **Structure** — adjacency, hierarchy. *Our format is unusually strong here.*
3. **Priors** — learned shape and semantics. *The honest residue.*

**Stop searching colour space. Add a defocus feature to the unary, and replace argmin-1-NN with an energy minimisation over the region adjacency graph already stored.**

## Lens-by-lens verdicts

| lens | verdict |
|---|---|
| Opponent spaces (OKLab, IPT, CAM16) | **Folklore here.** We already use CIELAB. Pointwise, so it cannot touch the collision. Not worth a run |
| Chroma/lightness decoupling | **Real for chromatic robustness, anti-useful for the collision** — down-weighting L makes two near-neutral dark classes *more* identical. Only as per-feature weighting inside a trained unary |
| Illumination-invariant / log-chromaticity | **Folklore, and actively harmful.** Designed to remove shading of the *same* surface. At `rgb≈(10..50)` it is division by small numbers — noise-dominated exactly where the collision lives |
| Shadow / intrinsic images | **Dead.** Report 12 already killed illuminant×albedo on measurement. Also conceptually wrong: ear and foliage are both genuinely low albedo |
| Texture banks (Gabor, LBP) | **Real but overkill on bokeh.** Local HF energy captures most of it at ~zero cost. Only if S1 dies |
| **Focus / defocus** | **The lever.** Defocus is a low-pass filter, so per-region HF energy is a blur estimate — established (depth-from-defocus). And the encoder already computes an ingredient: per-region residual SSE is integrated texture+focus energy |
| Scale-space / nested partitions | **Real but gated.** Coarse-parent id is a free feature — but the merge is colour-SSE-driven, so *precisely at achromatic boundaries* it may already have fused ear into foliage. The cue is contaminated exactly where it is needed. Test first (S0) |

## The machinery

**Boykov–Jolly / GrabCut-style s-t min-cut on the region graph — this is the one to build.** Unary from a scribble-fit model over region features; pairwise Potts weighted by shared boundary length × `exp(−ΔE²/2σ²)`. Binary labels make it submodular, so **min-cut is exact**. On ~1,200 nodes it is microseconds, and max-flow is ~150 lines of Go with no dependency.

Everything it needs already exists: `share[]` in `colorBytes2` **is** the boundary-length-weighted adjacency graph; `exactPartition` gives labels and colours; areas and residual SSE are one pass each.

**The 140–249× decisions ratio understates this.** A pixel-grid graph cut is 300k nodes and superlinear; the region-graph cut is 1.2k.

**A failure mode stated before it bites.** Contrast-sensitive smoothing prefers cutting at high-contrast edges. The ear/head edge is high-contrast (cheap to cut); the ear/foliage edge is the collision (expensive). **With an uninformative unary the cut hands the ear to the background** — the classic camouflage failure. Two fixes, both in the design: the defocus unary makes the ear informative, and the pairwise weight should include **focus contrast**, so a sharp/blurred boundary is object evidence even at ΔE≈0. That second term is the consultant's hypothesis, and S2 ablates it.

**Alternatives:** random walker (Grady) if the cut misbehaves — no shrinking bias, soft probabilities that double as UI ("here are the uncertain regions, touch one"). Binary Partition Tree for the hierarchy. **A learned-weight CRF adds nothing at this scale** — two swept hyperparameters beat a CRF with no data to train it.

## Roadmap, with pre-registered kill rules

| # | step | cost | kill rule |
|---|---|---|---|
| **S0** | **Hand-paint ground-truth masks**, then check the partition *contains* the answer: is the GT subject boundary present in the partition? | 1 day | <80% of collision-zone GT boundary at the working mark **and** <95% at the finest → region-level labelling cannot reach Lift-Subject quality; a boundary-band pixel refinement becomes required and the O(regions) claim gets an asterisk |
| **S1** | **Per-region defocus feature** (mean squared Laplacian; residual SSE as the free alternate) | ½ day | median ratio subject-dark : background-dark **<2× → defocus is dead**, jump to S5 and accept a learned selector. ≥5× → proceed |
| **S2** | **`lab bgcut2`: graph cut, both arms**, identical machinery on region graph and pixel grid | 2–3 days | must cut collision-zone error **≥50% vs 1-NN** at equal scribbles, else keep 1-NN. If the pixel arm matches on IoU *and* edge fidelity, the substrate claim collapses to cost-only — **publish that** |
| **S3** | Hierarchy feature (coarse-parent id, merge lifetime) | ½ day | <2 IoU points → drop it |
| **S4** | **Scribble economy** — IoU vs 1/2/4/8/16 scribbles | ½ day | if graph cut at 2 points does not beat 1-NN at 8, "one touch" is not real and the product needs multi-scribble UI |
| **S5** | **Compare against Apple Vision** `VNGenerateForegroundInstanceMaskRequest` — this machine is a Mac and that *is* Lift Subject. ~40-line Swift CLI, no new deps | 1 day + masks | Vision wins by >10 IoU → the format is the **execution layer** under a learned selector. Within ~5 points on bokeh → a no-model Lift Subject, the strongest result in this line. **If we beat it, check the test first** |
| **S6** | Compose: mask → snap to regions → region-id selection → existing container path. Converges with H1 | 1–2 days | — |

**S0 + S1 alone (1.5 days) tell you whether the rest is worth running.**

> **Verify the Vision API before building against it.** The name and availability are from the consultant's memory; this project's rule is to check the real source first. Marked TODO in S5.

## What not to do

- **No more colour spaces, tone curves or chroma stretches as collision fixes.** Pointwise; reports 36/37 paid for that lesson twice.
- **No illumination-invariant route** — noise-dominated at the dark end, rhymes with killed report 12.
- **No intrinsic-image decomposition** — killed on our own measurement.
- **No unsupervised anything** — report 33 settled it.
- **No texture banks before S1** — focus subsumes them on bokeh.
- **No self-trained CNN**, no learned-weight CRF.
- **Never again run a pixel arm without the region arm's machinery.** Falsification #14 is the freshest wound; S2 bakes the symmetry in.

## What genuinely needs a learned model

Stated explicitly: **deep depth-of-field scenes** (focus cue gone), **semantic disambiguation** ("the left person"), and **soft matting**. Those are priors, and no classical feature manufactures them.

**And a non-goal to name now:** everything above produces a *hard region-id cutout*. Lift Subject ships a **soft matte** — hair wisps are sub-region, and per-region flat alpha cannot represent them. That is `DESIGN-ALPHA.md` mode 2 / boundary-band territory. **Revive trigger: the demo needs hair.**

## Standing

The consultant's position — that the classical stack gets within touching distance of Lift Subject on the bokeh class — is **a hypothesis**, and S5 is written to kill it. The format's role survives either outcome: it is the cheapest and most stable *execution substrate* for whichever selector wins. S2(b) and S5 put numbers on that sentence instead of asserting it.
