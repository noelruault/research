# 11 — Validation at scale (Kodak-18)

The claim — "our quantizer beats the incumbents" — re-tested on a recognized
benchmark at 3× the original corpus, scored identically (same CIEDE2000, no dither,
identical source pixels). Raw: [11-kodak-validation-data.txt](11-kodak-validation-data.txt).

**On the dataset:** CQ100 (the intended set) is **unreachable in this environment** —
Mendeley is off the egress allowlist (`Host not in allowlist`). The substitute is the
**Kodak Lossless True Color Image Suite**, the canonical pre-CQ100 color-quantization
benchmark (768×512 photographs). 18 of 24 images were retrievable from a GitHub raw
mirror (`lemire/kodakimagecollection`; 6 were 404). So this is **Kodak-18**, which
together with the six paintings (reports 02/05/09) is a **24-image** validation —
4× the original, anchored by the standard set, but not the 100-image CQ100.

## Result — mean ΔE2000 over 18 Kodak images (lower = better)

| N | pngquant | ImageMagick | refine-RGB | refine-OKLab | ours (crossover) | win vs pngquant | win vs IM |
|---|---|---|---|---|---|---|---|
| 4 | 8.396 | 8.887 | 8.124 | **8.093** | 8.124 | 14/18 | 14/18 |
| 16 | 4.266 | 4.976 | **4.205** | 4.367 | 4.205 | 10/18 | 18/18 |
| 64 | 2.507 | 2.907 | 2.498 | **2.478** | 2.478 | 11/18 | 18/18 |
| 256 | 1.595 | 1.813 | 1.628 | **1.561** | 1.561 | 13/18 | 18/18 |

("ours crossover" = refine-RGB for N≤32, refine-OKLab for N≥64 — the ship config.)

## Verdict — honest

- **We beat both incumbents on the mean at every N.** The headline claim survives the
  3× scale-up on the standard benchmark.
- **vs ImageMagick (octree): decisive.** 18/18 images at N=16/64/256, 14/18 at N=4.
  Not close.
- **vs pngquant (libimagequant): competitive-to-better, not a sweep.** We win the mean
  at all four N (by 1.4–3.7%) and the majority of images (10–14 of 18), but at **N=16
  it's effectively a tie** (4.205 vs 4.266, 10/18). So the fair statement is *"matches
  or slightly beats libimagequant, and clearly beats ImageMagick"* — not "dominates
  pngquant." Overclaiming would be unscientific; pngquant is a strong, well-tuned bar
  and we are at or just past it.

## Crossover — confirmed in shape, fuzzy at the very low end

The RGB↔OKLab crossover (report 02) broadly holds on Kodak: **OKLab wins decisively at
N≥64** (and N=256), RGB and OKLab are **close at low N**. One wrinkle: at N=4 OKLab
edged RGB on Kodak (8.093 vs 8.124), whereas on the paintings RGB won at N≤16. So the
low-N winner is **dataset-dependent and within noise**; the consequential, robust
split is **N≥64 → OKLab**. For shipping, the exact low-N threshold (RGB vs OKLab below
~32) doesn't matter — both beat the incumbents and differ sub-perceptibly.

## What this settles, and the remaining honest gap

- **Settled:** the quality result is not a six-image fluke. On 18 standard photographs
  the ship config beats ImageMagick comprehensively and edges/matches libimagequant at
  every palette size, with the crossover confirmed.
- **Still open:** true **CQ100** (100 images, per-image reference MSE) would be the
  fully formal claim and is the right thing to run wherever Mendeley is reachable; the
  harness (`compare-quant.sh`, `emit`/`score`) takes any directory, so it's a corpus
  swap, not new code. GIMP and Aseprite-CLI remain unadded (heavy / no CLI here).
- The N=16 near-tie with pngquant points back to the only real headroom (report 06):
  **perceptual error weighting** — a *different objective* needing its own metric, not
  more clustering.

## Reproduce

```
# fetch Kodak-18 into corpus-kodak/ (lemire mirror), then:
cd bench && go build -o /tmp/qb .
# per N: pngquant --nofs, convert -dither None -colors, qb emit (rgb & oklab), qb score
```
Exact loop in `11-kodak-validation-data.txt`'s companion script (`/tmp/kodak.sh` in the
session); the engine modes are `bench/modes.go`.
