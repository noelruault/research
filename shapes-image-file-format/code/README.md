# code — the runnable experiments

Everything needed to re-derive the numbers in the reports. Nothing here is imported or shipped by any project; it is the evidence apparatus.

## The eval image is not included

The fixed eval is the macOS Sierra wallpaper at 512×288. That is an Apple asset and is not redistributed here. Supply your own `sierra.png` at 512×288 (any photograph with both smooth and textured content will reproduce the *shape* of the results, though not the exact byte counts).

## Layout

| path | what it is |
|---|---|
| `lab/` | the main experiment binary — Potts/Mumford-Shah region merge, Ising wall relaxation, crack-edge and contour coding, Voronoi and affine variants, PSNR |
| `lab/lowrate2.sh` | full-range WebP and AVIF rate-distortion sweep on one metric |
| `mix/` | context mixing over orders 0–4 (logistic mixing, online weight learning) — settles the starvation-vs-dilution question in report 06 |
| `geom/` | geometry-cost experiment: what does the *shape* of an image cost with colour free |
| `bakeoff/` | the original encoder bake-off harness — SVG, binary varint, quadtree, indexed PNG |
| `corpus/dict.sh` | shared-dictionary leave-one-out test over a best-case corpus |
| `compare.py` | matched-fidelity comparison; reads both data files and interpolates, so no number in report 05 is hand-derived |

## Running

```sh
cd lab && go build -o lab .

./lab frontier sierra.png     # the region-merge scale-space (report 04 data file)
./lab psnr a.png b.png        # the RGB PSNR definition used throughout
bash lowrate2.sh              # codec sweep (report 05 data file)

cd ../mix  && go run . ../lab/sierra.png     # context mixing
cd ../geom && go run . ../lab/sierra.png     # geometry cost

python3 compare.py            # the matched-fidelity table
```

`compare.py` reads the two `*-data.txt` files in the parent directory, so regenerate those first if you change anything upstream.

External tools required: `cwebp`, `dwebp`, `avifenc`, `avifdec`, `brotli`.

## A note on determinism

`lab/potts.go` sorts merge candidates by a **total order** — the energy key, then the pair index. This is load-bearing. The comparator originally ordered by key alone, and because Go randomizes `range` over a map, the initial heap was built in a different order each run and equal-key candidates popped differently. That produced up to 7% spread in bytes at the coarse end and a headline taken from a single lucky run. Report 06 covers what it invalidated. If you modify the merge, verify determinism across at least three runs before believing any number it produces.
