# 04 — The adjudication: four disciplines, one honest number

**Question, posed at full strength.** *If serializing shapes destroys 2D adjacency, then don't serialize. How would nature solve this — bees, ants, grain growth, NASA?*

**Method.** Four independent adversarial investigations, each with a lens from outside compression (physics / energy minimization, collective systems / stigmergy, vision science, spacecraft downlink), each required to produce a mechanism, a runnable experiment, and a predicted number — plus two experiments of my own. **Every agent had to reproduce the eval before its findings were believed**; one agent's headline finding turned out to be an artifact of it silently substituting a different image, and the reproduction rule is what caught it.

All four returned **VERDICT: NONE**.

## The wall, re-measured precisely

`avifenc -q 30 -s 4`, `cwebp -q 34`, PSNR by the repo's own RGB definition:

| encoder @ ~28.6–28.7 dB | bytes | vs AVIF |
|---|---|---|
| **AVIF q30** | **8,911** | — |
| **Potts+Ising region coder** (1,685 regions; geometry 9.0 KB + colour 4.9 KB) | **12,202** | 1.37× behind |
| WebP q34 | 12,350 | 1.39× behind |
| best raster on the n=16 index grid (mixed context, orders 0–4) | 16.2 KB | 1.9× behind |
| SVG rect cover — what the project ships today | 110.9 KB | 12.8× behind |

## The one real win: the region count was an artifact

The premise under test was that 32,924 rects for a 147,456-pixel photo is absurd — 4.5 px per region is not "shapes", it is the quantizer manufacturing false contours on smooth gradients, and the greedy cover then splitting each region into many rectangles. **That premise is correct**, and three investigations confirmed it independently at increasing strength:

- true 4-connected regions of the same grid: **12,068**
- Felzenszwalb segmentation of the *original* at matched fidelity: **5,442**
- energy-minimizing Potts / piecewise-constant Mumford-Shah merge with zero-temperature Ising wall relaxation: **1,685 at 28.66 dB, 153 at 24.03 dB**

Straight walls also code far cheaper than jagged ones: the Ising relaxation alone cut bytes 15%.

So the shipped pipeline leaves an order of magnitude on the table *within the shape idea*: **~12 KB and 1,685 primitives instead of 110.9 KB and 32,924, at identical fidelity.** That is the actionable result of the whole investigation, and it is an engineering result rather than a research one.

## The wall it cannot clear, and why

At its own best operating point the region coder ties WebP and loses to AVIF by 1.37×. Optimistically improving both halves (better colour prediction, ~15% better contour coding) projects to ~9.3 KB against AVIF's 8.9 KB.

Four independent mechanisms explain the ceiling, and they agree:

1. **Geometry cannot be both cheap and informative.** Measured isoperimetric excess at the operating point: region walls are **2.22× longer** than compact cells of the same areas would be. That excess *is* the image information — it is exactly what makes cells follow objects rather than tile space. Plateau's laws, surface tension, grain growth and every other physical relaxation *minimize* wall length, i.e. they minimize precisely the quantity carrying the signal. Emergent geometry is cheap because it is uninformative.

2. **"Don't serialize" is a rename, not a loophole.** Any region the decoder can regrow from the already-decoded image is by construction a function of causal pixels; using it to predict the next pixel *is* a context model. The whole backward-adaptive / stigmergic family is therefore bounded below by `H(X | causal past)` — and the order-2 coder from report 02 is already a member of it. The idea cannot beat the coder it reduces to.

3. **Shape rate is affine in perimeter and independent of texture rate.** From MPEG-4's own rate model: as total rate falls, texture rate shrinks and shape rate does not, so the shape fraction tends to 1 as R tends to 0. Corroborated by the people who built these coders: Cagnazzo, Poggi & Verdoliva (EURASIP JIVP 2007) measure a segmentation map alone at **95% of the entire flat-coder budget** at 30 dB, with an object-based total 154% worse. *(Report 05 finds the sign change this argument misses: the raster's own per-block overhead also fails to shrink, which is why a crossover exists at all.)*

4. **Flat interiors band exactly where the eye is most sensitive to it.** Split by content at matched PSNR, flat regions *beat* AVIF on the textured mountain (MS-SSIM 0.968 vs 0.961) and *lose badly* on the smooth sky (0.902 vs 0.983), because a flat interior produces false contours where retinal lateral inhibition makes them maximally salient (Mach bands). The primitive is perceptually good at edges and texture, perceptually costly on gradients.

## Mechanisms tested and killed, with numbers

| mechanism | discipline | result |
|---|---|---|
| Backward-adaptive / stigmergic derived regions | collective systems | collapses to the context coder, ≥16.2 KB |
| PDE diffusion inpainting (Weickert R-EED, the literal morphogenesis codec) | physics / biology | seed mask **alone** 9.5 KB, over the wall before colour; total ~21–25 KB |
| Centroidal Voronoi / Lloyd relaxation, seeds transmitted | physics | 44.3 KB |
| Seeds *derived* by decoder from a shared coarse field — geometry cost exactly **zero** | physics | colours alone 17.6 KB; collapses to downsample + upsample |
| L-system / grammar coding | collective systems | reduces to fractal/IFS coding, lost to DCT in the 90s |
| Contour / turn coding instead of crack raster | space systems | junction map costs more than the turns save; topology tax is 60% of explicit geometry's cost |
| First-order (affine) Mumford-Shah regions | physics | *worse* than constant regions: 3× the parameters, no fidelity gain — at this rate the residual is texture, not gradient |
| Foveation / CSF | vision | −71% bytes, but codec-agnostic preprocessing that lowers the wall for everyone, and needs gaze |
| Portilla-Simoncelli texture synthesis | vision | ~5–6 KB, the only sub-wall number found — but PSNR collapses to ~18 dB; a generative texture codec, not a shape format |
| Hyperspectral amortization, 200 bands, geometry free | space systems | geometry falls to 2% of budget and shapes still lose **3.4–14.5×** |
| Shape-delta downlink against a shared reference | space systems | **+3.4%** on a two-motion scene designed to favour it; collapses to AV1 wedge/GPM |
| Progressive / truncatable shape stream | space systems | 6.37 dB behind JPEG-2000 at the wall's budget; needs 12× the bytes to match |
| Packet-loss resilience | space systems | raster segmentation buys it for +0.84% at 1080p; the rect encoding costs +483% |
| Shared dictionary / cached corpus prior | compression | 0% on dissimilar images; format-agnostic where it works (report 03) |
| Shared dictionary on a *best-case* corpus | compression | 1.02× shapes vs 1.01× WebP — no asymmetry (report 03) |
| Phase transition in palette size n | physics | swept n = 2…256, bpp monotonic, **no knee, no regime where shapes get cheap** |

## The industry already ran this contest

MPEG-4 settled the argument by measurement, not taste: context-based arithmetic coding of the shape bitmap beat vertex/polygon contour coding by **20.5% in inter mode** — on shapes' home turf. "Second-generation" segmentation coding (1985–95) lost to DCT and was abandoned. AV1 and VVC tested free-form partition boundaries and kept only heavily restricted parametric forms (wedge, GPM). **No operational spacecraft image codec is segmentation- or shape-based** — CCSDS 121/122/123 and ICER are Rice, wavelet and predictive coders, and in the most bandwidth-starved domain in existence region information appears only as ROI bit-budget weighting and loss containment, never as the representation.

No published study surfaced that benchmarks a geometric representation against WebP/AVIF/JXL at matched PSNR; the "geometry wins" results (Demaret & Iske, Shukla) are all against JPEG-2000 on smooth content, and on textured photos Peter & Weickert's diffusion coder loses to JPEG-2000 outright.

> **Treat that absence as unverified.** It rests on agent literature sweeps plus two targeted web searches, not a systematic review, and it is a claim about the *absence* of work — the hardest kind to establish. Anyone leaning on it as a novelty argument must run a proper search of IEEE Xplore, PCS, ICIP and arXiv first.

## Verdict

**The compression ambition is parked.** A shape representation can be made to tie WebP with a proper energy-minimizing segmenter, and that is a genuine 9× improvement on what the project ships. It cannot clear AVIF, and the reason is structural rather than a matter of engineering effort: geometry is either emergent-and-uninformative or informative-and-expensive, and its cost is largely independent of the rate being targeted.

**What survives, and is not a bytes claim:** ~1,700 named, editable, independently animatable regions instead of 32,924 anonymous rects; immediate-mode drawing with no decode step; per-region addressability; a truncatable progressive stream; and guaranteed per-pixel error bounds. No raster codec offers any of those. None of them is measured in bytes.
