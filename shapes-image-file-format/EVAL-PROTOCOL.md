# Evaluation protocol — comparing against real tools and shipping formats

**Status: PROPOSAL, not yet registered.** Written 2026-07-30, before any of these runs. To activate any measurement below, append an amendment to `PREREGISTRATION.md` adopting it verbatim or with named changes — never run first and register after. M1–M3 in `PREREGISTRATION.md` are untouched and still run first; reports 29–31 stay reserved for them. Results from this file land as reports 38+.

This document closes four gaps the record names but has not run: no comparison against real background-removal tools (reports 33, 35 caveats), no comparison against other segmentation methods on our own substrate (report 28 caveat, method rule 6), format comparison on PSNR only (report 23 caveat), and no mapping to what incumbent communities measure, so outsiders cannot read our numbers.

Everything here inherits the standing invariants: name the knobs before the run, reproduce the baseline before believing, a perfect result is a broken test, publish the unflattering metric next to the flattering one, steelman both sides symmetrically (falsifications #11 and #14 are what happens otherwise), run it twice, stage explicit paths.

## Verification ledger — checked on this machine, 2026-07-30

Per the study's rule, nothing below is asserted from memory. Each fact is either verified today with the method stated, or marked TODO.

| fact | how verified |
|---|---|
| `ssimulacra2`, `butteraugli_main`, `cjxl`, `djxl` already installed (jpeg-xl 0.11.1_3, brew) | `ls /opt/homebrew/bin`, ran both binaries, usage text captured |
| `vmaf` 3.0.0 installed | `vmaf --version` |
| `dssim` 3.4.0, `oxipng` 10.1.1, `zopfli` 1.0.3 available in brew, not installed | `brew info` each |
| `cwebp` 1.6.0, `avifenc` 1.3.0 (aom 3.13.1), `ffmpeg` 8.1, `pngquant`, `brotli` installed | `which` + version flags |
| Python 3.14.2 system, **no numpy** — but `uv` 0.11.16 present; `uv run --with scikit-image` gives scikit-image 0.26.0 + numpy 2.5.1 in an isolated env; `slic`, `felzenszwalb`, `watershed` import and run | executed a 64×64 SLIC/Felzenszwalb round trip |
| `rembg` 2.0.77 installs and imports under `uv run --python 3.12 --with "rembg[cpu]" --with "numba>=0.59" --with "llvmlite>=0.42"` (onnxruntime 1.28.0). Without the numba pin the resolver picks numba 0.53 / llvmlite 0.36 and the build **fails** | both runs executed; failure and fix captured |
| Apple Lift Subject reachable: `VNGenerateForegroundInstanceMaskRequest` compiles with `swiftc` and instantiates (revision 1) on this machine, macOS 26.5.2 (25F84) | wrote and ran a 10-line Swift probe |
| Kodak fetch URL live | `curl -I https://r0k.us/graphics/kodak/kodak/kodim01.png` → HTTP 200 |
| Sprites corpus exists at `/Users/noelruault/go/src/github.com/noelruault/sprites` (report 32's path) | `ls assets/props/extra` → ak74/bow/pickaxe present |
| `lab bgclass` derives regions via `exactPartition(ours)` — connected components of identical colour in a flat render — so **any external partition rendered as a flat-mean-colour PNG feeds the identical binary unchanged** | read `code/lab/bgclass.go` |
| rembg code MIT; models offered include u2net, isnet-general-use, isnet-anime, birefnet-* , BRIA RMBG 2.0, SAM | fetched github.com/danielgatis/rembg |
| U-2-Net (u2net weights' home repo) Apache-2.0 | fetched github.com/xuebinqin/U-2-Net |
| SAM 2: code, checkpoints, training code Apache-2.0; has an unprompted automatic mask generator; 4 checkpoints (Tiny 38.9M → Large 224.4M) | fetched github.com/facebookresearch/sam2 |
| BiRefNet: MIT; targets DIS/HRSOD/COD/matting; reports S-measure, weighted F, MAE, HCE | fetched github.com/ZhengPeng7/BiRefNet |
| AIM-500: 500 natural images with manually labelled alpha mattes — 100 portrait, 200 animal, 34 transparent, 75 plant, 45 furniture, 36 toy, 10 fruit; Google Drive; MIT-badged release agreement | fetched github.com/JizhiziLi/AIM |
| AM-2k: 2,000 animal images with alpha mattes, 20 categories; Google Drive; MIT-badged release agreement | fetched github.com/JizhiziLi/GFM |
| P3M-10k: 10,421 portraits with alpha mattes; test sets P3M-500-P (**faces blurred**) and P3M-500-NP (no blur); Google Drive; MIT-badged agreement | fetched github.com/JizhiziLi/P3M |
| BSDS500: official Berkeley page live; terms quoted: “free to download … for non-commercial research and educational purposes”, test-set quarantine rule stated | fetched the Berkeley BSDS page |
| DIS5K: DIS-TR/VD/TE1–4; binary GT masks; Apache-2.0 code + a separate `DIS5K-Dataset-Terms-of-Use.pdf`; benchmark metrics maxF, weighted F, MAE, S-measure, E-measure, HCE; isnet.pth “academic”, isnet-general-use.pth “broader applications” | fetched github.com/xuebinqin/DIS |
| DUTS: official site says all rights reserved by the authors; images derive from ImageNet; binary GT only | web search; **rejected below on those grounds** |
| CLIC: images released under the Unsplash license (2020 lossy track); pro-valid 41 images, 2021 test 60; hosting moved to Google Cloud storage Aug 2025 | web search across clic.compression.cc archives + TFDS catalog |
| Matting convention: SAD/MSE/Grad/Conn (Rhemann et al. 2009); trimap methods score the unknown region only, **trimap-free methods score the whole image** | literature search, multiple independent papers agree |
| AOM CTC (AV2, v8): PSNR/PSNR-HVS, SSIM/MS-SSIM, CIEDE2000, VMAF, CAMBI, per-plane and weighted-YUV **BD-rate**, plus complexity | located CTC PDFs on aomedia.org |

**TODO — could not verify today, must be resolved before the dependent run, never assumed:**

- **T1.** The actual text of the AIM-500 / AM-2k / P3M release agreements (the PDFs behind the MIT badge). Read at download time; if any restricts benchmarking or publication of scores, record it and adjust.
- **T2.** Per-tool training-data contamination table: which corpora each rembg model was trained on (u2net → DUTS-TR is reported in its paper; isnet-general-use → DIS5K-TR + undisclosed; birefnet-general → multi-dataset mix including DIS5K). Must be pinned from each model's paper/README before scoring that model on any corpus, and any overlap disclosed in the report. A tool scored on its own training distribution is a steelman; scored on its training *set* is a broken test.
- **T3.** Exact CLIC download URLs post-2025 migration, and the per-image Unsplash provenance.
- **T4.** Connectivity (Conn) metric implementation source. SAD/MSE/Grad are a page of Go each from the Rhemann definitions; Conn is not, and the reference implementation is MATLAB. Either port it with a cross-check against published numbers, or drop Conn and say so — do not ship an unverified Conn.
- **T5.** SAM 2 install on macOS/CPU via uv (torch dependency, ~GBs). Unverified. Only needed if M3's comparison-arm check selects it.
- **T6.** Google Drive fetch for the matting corpora needs `gdown` or a browser; not scriptable-verified today.
- **T7.** Whether Apple's mask API returns a soft matte via `generateScaledMaskForImage` as expected (the request class is verified; the full CLI is not written).

---

## The three claim families, and what would falsify each

| family | claim as currently held | falsifier this protocol can produce |
|---|---|---|
| F1 — substrate | The region graph is a better *execution* substrate for selection than pixels: 140–249× fewer decisions, edges on 28–54% larger CIELAB steps (report 35) | An equal-count off-the-shelf partition (SLIC/Felzenszwalb) matches those numbers → the advantage is “having any partition”, not “having ours” (M5). A real tool's mask snapped to our partition loses accuracy vs snapped to SLIC (M4b) |
| F2 — product | “Cheap supervised colour-over-regions” is a usable selection primitive next to real background-removal tools | Real tools beat the click arm so broadly that no image class survives for it (M4c thresholds below) |
| F3 — format | Poor image codec / good structured-image format: loses to WebP alone, beats WebP+sidecar by ~30% (reports 23, 24) | The win does not survive perceptual metrics, BD-rate aggregation, a modern corpus, steelmanned lossless arms, or a cheaper honest sidecar (M6 rules below) |

---

## M4 — Background removal against real tools

The gap reports 33 and 35 both name in their caveats. Three experiments, deliberately separated, because they test different claims.

### Arms

| arm | what it is | pinned how |
|---|---|---|
| rembg/u2net | `rembg` 2.0.77, session `u2net` | version + model hash in data companion |
| rembg/isnet | session `isnet-general-use` | same |
| rembg/birefnet | session `birefnet-general` | same |
| Apple Lift Subject | Swift CLI around `VNGenerateForegroundInstanceMaskRequest`, macOS build recorded (the model is OS-embedded and not separately versioned — disclose exactly that) | macOS 26.5.2 (25F84) |
| ours | SHPC partition at the capability mark (~1,200 regions, the operating point of reports 33–35) + the M4c click rule | commit hash |

**SAM 2 is excluded from M4 by design, and the exclusion is pre-registered.** Its automatic mode emits *many* masks with no foreground/background decision; any rule we write to pick “the subject” would be us authoring their tool, which is the mirror image of falsification #14. SAM 2 belongs in M3's comparison-arm check (segmentation quality), where unprompted mask *sets* are the right shape. If M3's “strongest freely available segmenter” check selects it, its Apache-2.0 licence is already verified.

### Corpora

| corpus | role | why | licence state |
|---|---|---|---|
| **AIM-500** | primary | natural images, real alpha GT, category spread including 200 animals — the content class reports 33–36 already use | MIT-badged agreement, **T1** |
| AM-2k (test split) | secondary | animals only; density where our record already has qualitative results | MIT-badged agreement, **T1** |
| P3M-500-NP | optional third | portraits, no face blur (the -P set's blurring alters exactly the pixels a matting eval scores — use NP) | MIT-badged agreement, **T1** |
| DIS5K DIS-VD | not for matting; optional for hard-mask IoU only | binary GT; also BiRefNet/ISNet train on DIS-TR, so it flatters them — usable only with T2 disclosure | terms PDF, **T1** |
| DUTS | **rejected** | all-rights-reserved licence, ImageNet-derived images, binary GT only | — |

Sample size: full AIM-500 if runtime permits; else the **first 100 by filename sort** — committed here so the subset cannot be chosen after seeing results.

### Metrics — primary vs alongside, per the M3 discipline

- **Primary: IoU and boundary F-measure at 2 px tolerance**, on masks binarised at α=0.5. These are the numbers a tools-community reader expects, and they are the ones our hard region edges could *lose* on soft-hair images — that is precisely why they are primary and not the CIELAB edge-step proxy this study invented for itself.
- **Alongside, unconditionally: SAD, MSE, Grad** on the full soft alpha, whole-image (the trimap-free convention, verified above). Our per-region flat alpha cannot represent a soft matte; these metrics will punish that and they get published anyway. Conn only if T4 resolves.
- **Alongside: decisions count and mask representation bytes** (region-id set vs PNG/RLE/WebP-lossless of the pixel mask) — the numbers *our* pitch rests on.
- Paired per-image win/loss counts, Wilcoxon signed-rank, and bootstrap 95% CI on mean deltas (via `uv run --with numpy --with scipy`). n=100–500 makes these real.

### M4a — harness calibration, not a result

Run each tool on the corpus, score against GT, compare to that tool's published corpus numbers where they exist. Any tool landing far from its published band means the harness is broken, not the tool. Nothing from M4a is publishable as a finding; it exists so M4b/M4c stand on a verified harness (the same role reproducing the anchor plays in every report here).

### M4b — the substrate test with a real tool. **This is the highest-information experiment in this file.**

Report 35's composition claim — “a model that outputs region labels inherits the cheapness and the exact edges” — is testable without our classifier at all:

1. Take each tool's raw mask on the **original** image (tools get their best input; knob named).
2. Project it onto three partitions of the *same decoded render* our arm would ship: **ours** at the capability mark; **SLIC** at matched region count (±20%, M2's rule); **Felzenszwalb** at matched count. Projection rule: per-region mean of the tool's soft alpha (report both α=0.5-binarised and soft where the metric allows).
3. Score raw vs each projection against GT: ΔIoU, ΔBF, ΔSAD. Plus bytes: pixel-mask encodings vs region-id set against each partition.

| outcome | reading |
|---|---|
| Snapping to ours loses less accuracy than snapping to SLIC/Felzenszwalb at matched count, and Δ vs raw is small (pre-registered: median ΔIoU ≥ −0.02) | F1 survives against a real tool: the partition carries real mask-relevant structure. Strongest available version of the report-35 claim |
| Snapping to ours ≈ snapping to SLIC | The substrate advantage is “any partition”, not ours. F1 dies in its current form; what remains is transmitted+stable (M1/M2), which snapping cannot test |
| Snapping to ours loses materially more than to SLIC | Worse than dead — our regions actively misalign with subject boundaries. Feeds M3's worst-case row and must be said plainly |
| Snapping *improves* on the raw mask (IoU up) | Check for a broken test first (principle: a too-good result). Then it is the composition pitch measured, and the headline |

Knobs, named now: tools run on originals; projections computed on the decoded render each partition describes; region counts matched ±20%; no morphological cleanup on any arm (snapping *is* the cleanup — adding another would repeat #14).

### M4c — the click arm against automatic tools

The asymmetry is structural and must be the first sentence of the report: **our arm receives k user clicks; the tools receive zero.** That is the product's honest shape (an interactive selector vs one-shot automatic tools), not a fair fight, and it is being measured anyway because the user-visible question is “why not just run rembg?”.

Click protocol, so no author hand-picks examples while looking at results (report 35's caveat): k keep points sampled uniformly from the GT foreground eroded by 5 px, k remove points from the eroded background, fixed RNG seed recorded, k ∈ {1, 3, 5, 10} all reported. Classifier: `lab bgclass` unchanged.

Pre-registered floors, committed before any run and stated as judgment calls anchored on the record: the claim “usable cheap draft mask” **survives** if at k=5 the median IoU ≥ 0.85 on AIM-500's SO (salient-opaque) subset, and **dies** if median IoU < 0.70 there. Between the two: report the distribution and the per-category split, claim nothing. On STM/NS categories (transparent, non-salient) we expect to lose outright — flat per-region alpha cannot represent them — and those rows are published as the unflattering companion, not filtered out.

---

## M5 — Other segmentation methods on our own substrate

Report 28 handed the client our own merge and called it the generous bound; M2 already registers the *stability* comparison. M5 is the missing *capability* comparison: rerun report 35's supervised protocol with the pixel arm upgraded from raw pixels to **its own partition**.

Mechanics, verified today: scikit-image's `slic`, `felzenszwalb`, `watershed` run under uv; a ~30-line bridge script fills each superpixel with its mean colour and writes a flat PNG; `lab bgclass` consumes that PNG through `exactPartition` **unchanged** — same binary, same classifier, same seeds, no new Go code. Neither side gets a knob the other is denied: superpixel count tuned to ±20% of our region count (M2's rule), parameters and versions recorded.

Measure exactly report 35's panel: decisions, blobs (with and without majority filter — keep #14's steelman), edge CIELAB step against the source referee. Where M4's corpora are in play, add accuracy vs GT.

Pre-registered decision rule:

| outcome | reading |
|---|---|
| Ours beats equal-count SLIC/Felzenszwalb on edge step and blobs | First evidence the *specific* segmentation matters. Check the test, then it is a new claim — register before promoting it |
| Parity | The report-35 advantage was “any partition vs no partition”. Rewrite the claim to say that; the byte pitch (report 24) is untouched because it already prices *any* sidecar, but the capability language must stop implying our partition is special |
| SLIC beats ours | Publish it. Combined with a bad M3 this ends the photo-editing branch on quality grounds as well as semantic ones |

Cost: hours. This is the cheapest falsification opportunity in the file.

---

## M6 — Format comparison rigour

Runs **after** the M-phase (PREREGISTRATION's out-of-scope note stands; this schedules the gap, it does not jump the queue).

### M6a — BD-rate over the existing sweeps

We already produce quality ladders (reports 05, 23); the aggregation is what is missing. Compute Bjøntegaard-delta rate per image with PCHIP interpolation over the **overlapping** quality interval only (AOM CTC practice, doc located), our PSNR definition on both sides as ever. Report per-image BD-rate, mean, and bootstrap CI, for: ours vs WebP (expected large positive — we lose; published), ours vs WebP+sidecar (expected negative — we win; published). **Reporting rule, adopted now: the two BD-rate numbers only ever appear together.** That is principle 5 turned into a mechanical rule an outside reader can check. Implementation: extend `code/compare.py` under `uv run --with scipy` rather than a new tool.

### M6b — perceptual metrics

`ssimulacra2` and `butteraugli_main` are already on the machine; `dssim` is one `brew install`. Run all three over the report-23 matched-fidelity pairs — no new encodes needed, an afternoon.

Expectation committed before the run: our posterised flat regions should score **worse** under perceptual metrics than PSNR suggests (banding is exactly what they punish and PSNR forgives), so this is likely an unflattering result and is published regardless — that is the point. Sensitivity check: re-run the matched-fidelity *matching itself* using SSIMULACRA2 as the matcher on a 6-image subset; if the report-24 verdict flips under a different fidelity definition, then “matched at PSNR” is load-bearing and every claim that rests on it must say so.

Skip VMAF for stills (video-fusion harness, Y4M plumbing, no still-image calibration story) — note the omission rather than half-run it.

### M6c — corpus

- **Kodak-24 stays primary.** Continuity with reports 23/24/28 outweighs its age; every existing baseline lives there. Fetch URL verified live.
- **Add CLIC pro-valid (41 images)** as the modern high-res arm, at native resolution, both sides (falsification #8: never freeze the resolution axis silently; falsification #11: both sides get the same delivery pipeline). Encode cost is the constraint — 2 m 7 s and 2.89 GB at 4K (report 18) — so pre-commit the subset rule: all 41 if feasible, else the first 15 by filename. Unsplash licence, T3 for exact URLs.
- **Sprites: replace n=3 with an enumerated corpus.** Inclusion rule committed now: every `.png` under the sprites repo's `assets/` tree, no hand-picking; report the count. Fixes report 32's “not a corpus” caveat with the cheapest possible discipline.

### M6d — steelman upgrades a sceptical reviewer would demand

1. **PNG arm gets a real encoder.** Report 32 used Go's `image/png` — a weak arm. Re-run lossless with `oxipng -o max` (brew, verified available) and name the flag. The PNG wins may shrink; that is the correction working.
2. **JPEG XL arm.** `cjxl` is installed and absent from every table. Lossless: `cjxl -d 0 -e 9`. Lossy: `-e 9` sweep. JXL is the strongest current lossless competitor and its absence is the first thing an outside reader will notice.
3. **Lossy sprites with alpha.** Report 32 was lossless-only by accident of the finest mark. The pitch lives at coarse marks; run matched-fidelity lossy vs `cwebp -m 6` with alpha on the enumerated sprite corpus. Fidelity metric must include the alpha channel — define it before running (RGBA PSNR with alpha as fourth channel matches how the merge already prices it).
4. **The sloppy-sidecar arm.** Report 24's caveat: a consumer might accept an *approximate* mask. Price one honest version: our label map downsampled 2× and 4×, upsampled nearest-neighbour, boundary displacement measured (mean px), sidecar bytes re-priced. Pre-registered rule: if a sloppy sidecar at ≤1 px mean boundary displacement lands cheaper than our margin, the +30.5% headline must be re-qualified to “vs an *exact* sidecar”, and the identity argument (M1/M2) carries the rest of the weight alone.
5. **Encode/decode cost table, in-process.** Report 32's timing was fork/exec noise and said so. Build the in-process harness (Go benchmark over decode; encoder wall-clock + peak RSS) once, reuse everywhere. Incumbents publish complexity; we currently cannot.

---

## What the incumbents measure — the legibility map

For each community, what they report, so our tables can carry at least one column an outsider already trusts:

| community | their numbers | our mapping |
|---|---|---|
| Codec (AOM CTC v8, located) | PSNR/PSNR-HVS, SSIM/MS-SSIM, CIEDE2000, VMAF, CAMBI; **BD-rate** over quality ladders; encode/decode complexity | M6a BD-rate, M6b metrics, M6d-5 cost table |
| JPEG XL circle | butteraugli distance, SSIMULACRA2 (both binaries already installed here) | M6b |
| Matting (Rhemann 2009 convention, verified) | SAD, MSE, Grad, Conn; whole-image for trimap-free | M4 alongside-metrics |
| DIS / salient-object (DIS repo, verified) | maxF, weighted F, MAE, S-measure, E-measure, HCE | report as context in M4 only if a DIS corpus is used |
| Superpixels (Stutz benchmark; metric list **partially verified — TODO** fetch its metrics page) | boundary recall, under-segmentation error, ASA | **M3's registered BR + UE is already this community's convention** — say so in report 31, it is the answer to “why not F-measure” an outsider will accept |

---

## Tooling, ordered by cost

1. **Free — already installed:** ssimulacra2, butteraugli_main, cjxl/djxl, vmaf, cwebp/dwebp, avifenc/avifdec, pngquant, brotli, ffmpeg, sips, Go 1.25, swiftc.
2. **One command:** `brew install dssim oxipng`. `uv` envs resolve on demand (incantations verified above; the rembg numba pin is load-bearing).
3. **Small scripts:** SLIC→flat-PNG bridge (~30 lines, uv); GT-seeded click sampler (~40 lines); BD-rate in `compare.py` (+scipy).
4. **Small Go:** `lab maskscore` (IoU, BF@2px, SAD/MSE/Grad vs a GT PNG — one file, mirrors existing verb style); `lab snap` (project a mask PNG onto a partition render). Conn is T4.
5. **Medium:** Swift CLI for Lift Subject (~80 lines; API instantiation verified, full pipeline is T7). In-process timing harness.
6. **Heavy, deferred:** SAM 2 under torch/uv (T5) — only if M3's arm-check selects it. Corpus downloads via Google Drive (T6) + agreement reading (T1, T2).

Unavoidable dependencies, named: Python via `uv` for scikit-image (superpixels), rembg (onnxruntime), scipy (BD-rate/stats) — no system-Python installs, everything in disposable uv envs with versions recorded in data companions. Swift/Vision for the Apple arm — macOS-only by nature, disclosed as such.

## Run order — information per unit effort

1. **M1, M2 as registered** — unchanged, first, cheap, code exists.
2. **M5** (SLIC substrate rerun of report 35) — hours, needs only the bridge script, can kill F1's current wording on its own.
3. **M4b** on an AIM-500 subset with rembg only — one day including corpus fetch + `maskscore`; the single most informative new run.
4. **M3 as registered** — BSDS500 fetchability and terms now verified above, which discharges its first TODO; the segmenter-arm TODO stands (SAM 2 licence pre-verified if selected).
5. **M6b** perceptual metrics over existing report-23 pairs — an afternoon, zero new encodes.
6. **M4c** click arm vs tools, full corpus + Apple arm (T7 CLI).
7. **M6a/M6c/M6d** BD-rate, CLIC, sprite-corpus expansion, sloppy sidecar, JXL/oxipng arms.
8. **M4a-adjacent extras** (tools run on our decoded renders — does posterisation break them?) — cheap curiosity, last.

## Amendment rule

Same as `PREREGISTRATION.md`: once any part of this file is adopted by amendment there, changes to that part are append-only with a dated note stating whether data had been seen. Until adoption, this file may be edited freely — it is a proposal, and saying otherwise would be pretending a registration that has not happened.
