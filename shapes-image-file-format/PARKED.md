# Parked ideas — shapes-image-file-format

Convention and required fields: [`../PARKED.md`](../PARKED.md). **Re-read this file whenever any baseline in this study moves** — that is exactly when a parked entry can silently become valuable and exactly when nobody thinks to look.

Live state and the active queue are in [`AUTORESEARCH.md`](AUTORESEARCH.md). This file is only for things *not* being worked on.

---

## P-01 — Contour turn coding · `parked`

**What it is.** Three free context taps on the contour coder's turn stream: the junction map of the three candidate targets, hard exclusions from decoder-known structure, and absolute direction of travel. No side information, no mode flags, no partition change.

**Why parked — with the number.** It works: **−6.40% of the turn stream** at 11,121 regions, and a junk control of 864 equal-sized contexts *costs* +0.49%, so it is information rather than capacity. It is worth almost nothing **only because of where the CAE/contour crossover now sits.** Report 09's interleave moved that crossover from ~6,400 regions down to between 849 and 1,383, so the contour coder is chosen only below ~849 regions at 4K = 21.99–24.25 dB. Measured *before* report 09 the same lever would have been worth **−2.76% of the whole file** at 6,417 regions.

**Depends on.** (a) The CAE/contour crossover position — currently ~849–1,383 regions at 4K. (b) The CAE coder's strength; anything that makes CAE *worse* widens the contour band. (c) The operating point the product ships at — report 14 found the capability sweet spot is 227–1,383 regions, which is **inside or adjacent to the contour band**.

**Revive when.** Any of: the crossover moves back above ~3,000 regions; or the product settles on 227–1,383 regions as its operating point, where contour is the coder actually used.

**Trigger checked 2026-07-30 and it did NOT fire.** The legality repair landed (report 20) and it does widen the contour band — at 11,121 regions the legal *published-style* CAE goes above contour, so that coder switches. But the coder actually chosen there is the **interleaved** one at 105,752 B, which beats contour's 126,291 and is itself confirmed decodable. So contour remains unchosen above ~849 regions and this entry stays parked. The "has not been recomputed" caveat is now discharged twice.

**Cost to revive.** Low — the code exists and reproduces. Re-run and re-price.

**Where.** Report `10-contour-turn-data.txt`; code `code/lab/turnx.go`, `turnprice.go`, `turnload.go`; commit `dbfb6bd`.

> **Flagged as the strongest revive candidate in this file.** A working, verified, free improvement that is worth zero solely because of a threshold that has already moved once today and is due to move again when #12 is fixed.

---

## P-02 — Contour loop channel · `parked`

**What it is.** The contour coder spends a flat `log2(nv)+2` per closed loop with no context and no ordering — 1,658 B at 1,383 regions (3.6% of the contour bill), 6,927 B at 6,417 (7.4%). Never examined.

**Why parked — with the number.** Killed by the same crossover collapse as P-01 before any work was done: its 3.6–7.4% shares sat at 1,383–6,417 regions, which the interleave handed to CAE.

**Depends on.** Identical to P-01.

**Revive when.** Same triggers as P-01 — and revive it *together* with P-01, since both only pay inside the contour band.

**Cost to revive.** Low-moderate; nothing has been built.

**Also unfixed:** the loop channel transmits no loop count and no end-of-loops flag, so a decoder cannot tell when the loop list ends. Repair costs 10.8 B at 227 regions to 277 B at 6,417. That is a **correctness hole, not an optimisation**, and it should be fixed whenever the contour coder is next touched regardless of this entry's status.

**Where.** Report `10-contour-junction-map-data.txt`; commit `d7f836e`.

---

## P-03 — Wider CAE context (16-bit template) · `subsumed, partially`

**What it is.** Greedy-selected 16-bit context template for the CAE V-plane, frozen and transferred across operating points. Worth **−6.22%** of the wall bill at 11,121 regions.

**Why parked.** The cross-plane interleave from the same round is worth **−12.64%** at identical model count, so the interleave was adopted and this was not. **Whether they stack has never been measured** — both attack the same wall bill and their gains cannot simply be added.

**Depends on.** The assumption that the two overlap substantially. Unverified.

**Revive when.** Anyone re-opens the wall coder. The stacking measurement is a single run and would settle whether there is another ~3–6% available at the mid-axis, which is where the remaining 8.3% gap to WebP lives.

**Cost to revive.** Low — both implementations exist and co-compile.

**Where.** `09-context-width-data.txt`, `code/lab/wallctx.go`, `code/lab/selected.go`; commit `a6d1c73`.

---

## P-04 — Colour lenses: spectra, matter, vision, biology · `blocked`

**What it is.** Four multi-disciplinary research lenses on colour coding — low-dimensional reflectance manifolds, a coarse-to-fine cascade over the merge hierarchy, perceptual chroma acuity, and biological/structural colour.

**Why parked.** Blocked on sequencing, **not on merit** — an earlier note cancelled them on an unmeasured ~1% ceiling and that reasoning was withdrawn (commit `a284d9b`). Report 11's floor is an upper bound on *achievable*, never a lower bound on entropy, and it was measured inside one model family (predict-then-code-residual). A lens proposing a different *representation* is not bounded by it.

**Depends on.** B10 (adopt the cross-channel transform) landing first, so each lens is measured against the post-RCT baseline of **6,898,336 B** rather than the pre-RCT 10,832,609. Against the old baseline every one of them would book a −28% win RCT already took.

**Revive when.** B10 lands. Then run **spectra first** — it is the only one of the four proposing a different representation rather than a better model.

**Cost to revive.** One agent each, sequentially.

**Where.** Ordering and rationale in `AUTORESEARCH.md`; the original briefs are in the workflow script `shapes-close-the-gap-wf_72ddd376-8f6.js`.

---

## P-05 — Residual-context colour model (B9) · `blocked`

**What it is.** Conditioning the colour residual on prediction spread × decoded-neighbour count. Measured **−10.33%** of the colour bill at lossless, standalone.

**Why parked.** Superseded in priority: on top of RCT it is worth **+4.7 pp, not 10.3**. It was queued as the largest colour lever and is roughly a third of the transform it should be stacked on.

**Depends on.** RCT being adopted first, and on the modelled-vs-real caveat below.

**Revive when.** Immediately after B10 — and it must be re-measured **through a real compressor**, not modelled. Report 12 found a refinement that improved the order-0 model by 0.04% and made brotli **0.60% worse**. This entry's −10.33% is a modelled figure and has never been checked against brotli.

**Cost to revive.** Low; implemented and reproducible.

**Where.** Workflow journal, `wf_72ddd376-8f6`; report `11-colour-entropy-floor.md`.

---

## P-06 — Re-tuning `bitsPerEdge` / `bitsPerReg` (B3) · `parked`

**What it is.** `potts2.go:15` hardcodes `bitsPerEdge = 1.73` and `bitsPerReg = 25.0`, measured at 512×288, and they drive the RD merge key and the Ising relaxation λ at *every* resolution. Actual wall cost at 4K is 1.22–1.61 bits/edge at the operating rungs and **0.4534 at lossless**.

**Why parked.** Not because it is small — plausibly the **largest single win left**. Parked because re-tuning changes the *partitions themselves*, so every published number would need re-running on the new partitions or it reproduces falsification #3. It is the most dangerous item in the queue to do carelessly.

**Depends on.** The wall coder's real cost per edge — which **report 09 already changed** (interleave −12.6%) and #12 will change again. The constant is staler now than when it was flagged.

**Revive when.** After #12 is fixed and the wall coder is stable, so the constant is re-measured once against a settled coder rather than twice.

**Cost to revive.** High — a full scale-space re-run plus re-pricing every published mark, baseline included.

---

## P-07 — Curve-fitted region boundaries · `parked` — **trigger has fired**

**What it is.** Fitting curves to the crack-edge boundaries to get genuine resolution independence, instead of a staircase on the pixel lattice.

**Why parked.** Report 13 listed it as P7, conditional on the regions being semantically meaningful — "only worth it if P0 says the regions are meaningful".

**Depends on.** P0.

**Revive when.** **P0 has now been answered positively** (report 14: heavy-tailed partitions, sky is 2 regions, a hundred regions cover 90–99% of the image). **The condition on this entry is satisfied and it has not been re-evaluated.** It remains parked only on cost, not on doubt.

**Cost to revive.** High — new subsystem, and the boundary coder would need re-pricing from scratch.

---

## P-08 — Per-region affine (first-order) colour · `killed on numbers that are now stale`

**What it is.** Report 04 tested first-order (affine) Mumford-Shah regions instead of piecewise-constant ones: a colour ramp per region rather than a flat fill. Killed with "*worse* than constant regions: 3× the parameters, no fidelity gain — at this rate the residual is texture, not gradient".

**Why this entry exists.** That is a **numbers kill against a baseline that has since moved a long way.** It was measured when region colours were coded with a boundary-weighted mean predictor, an order-0 residual model, and **no cross-channel transform** — the configuration report 11 showed sits 36% above its own data's floor. Affine coefficients were judged too expensive relative to a colour coder we now know was badly specified.

**Depends on.** The cost of transmitting per-region colour parameters, which RCT (−28%) and the residual-context model have both changed.

**Revive when.** After B10. Cheap check: re-run report 04's affine comparison with RCT applied to *both* arms. If affine coefficients also compress ~28% better, the original comparison's conclusion may or may not survive — **and nobody knows which**, because the arms were never re-measured together.

**Cost to revive.** Low — `code/lab/affine.go` exists.

**This is exactly the class the convention file warns about:** killed on a comparison, where one side of the comparison has since improved by 28%.

---

## P-09 — Progressive / truncatable shape stream · `killed on bytes, alive as capability`

**What it is.** Shipping the scale-space hierarchy coarse-to-fine so the stream can be truncated anywhere and still decode a valid image.

**Why parked.** Report 04 killed it **as a bytes argument** — 6.37 dB behind JPEG-2000 at the wall's budget, needing 12× the bytes to match.

**Status changed by re-framing, not by measurement.** Report 13 lists it as strength 5: the nested hierarchy is *already built* (`hdMarks`) and gives level-of-detail that follows objects rather than a mip grid. As a capability for gaming LOD and constrained-link transmission it is intact and unimplemented.

**Revive when.** Any application work starts — it is already there and needs a container, not research.

**Where.** `hdMarks` in `code/lab/hd.go`; report 04.

---

## Killed and *not* parked — do not re-derive these

These fail on arguments that do not expire when a baseline improves. Full list and numbers in `04-adjudication.md` and `06-corrections-and-falsifications.md`.

- **"Don't serialise, let the decoder regrow the regions."** Any region reconstructible from decoded pixels is a function of the causal past, so it *is* a context model, bounded below by `H(X | causal past)`. Improving any coder does not touch this.
- **Seeds derived by the decoder from a shared coarse field.** Collapses to downsample-and-upsample.
- **L-system / grammar coding.** Reduces to fractal/IFS coding, which lost to DCT in the 1990s.
- **Foveation / CSF.** Real (−71%) but codec-agnostic preprocessing that lowers the wall for every codec equally, and needs gaze.
- **Illuminant × albedo decomposition** (report 12). Killed on measurement, not argument: dividing region colours by a smooth illumination field *raises* their joint entropy 5.9% and every shuffled control beats the real field. Fails in the 21-parameter form and fails monotonically worse as the field gets finer.
