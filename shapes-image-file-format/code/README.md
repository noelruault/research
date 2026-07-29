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
| `dashboard/` | builds the two visual pages — `build.sh` encodes every operating point with each codec and keeps a decoded copy, `gen.py` embeds them as data URIs and draws the rate-distortion chart, `genhd.py` builds the 1:1 mirror |
| `hd/` | the native-resolution round — `sweep.sh` and `ladder-sweep.sh` sweep the codecs at each size, `mirror-build.sh` cuts the 1:1 windows, `hdcompare.py` does the matched-fidelity interpolation |
| `lab/strongsweep.sh` | the report 05 sweep re-run at `cwebp -m 6` and `avifenc -s 0`, which is what killed the low-rate win |

## Running

```sh
cd lab && go build -o lab .

./lab frontier sierra.png     # the region-merge scale-space (report 04 data file)
./lab psnr a.png b.png        # the RGB PSNR definition used throughout
bash lowrate2.sh              # codec sweep as first published (report 05 data file)
bash strongsweep.sh           # the same sweep with both codecs configured properly (report 06 #9)

./lab hd src4k.png renders/   # native-resolution round: lossless price + scale-space (report 08)
./lab hdcheck sierra.png      # asserts the lean colour coder matches the original
./lab hdnd sierra.png         # the determinism probe that exposed report 06 #7
./lab crop in.png out.png x y w h
./lab diff a.png b.png out.png 32
./lab stat a.png b.png        # PSNR plus the error distribution one PSNR figure hides

cd ../mix  && go run . ../lab/sierra.png     # context mixing
cd ../geom && go run . ../lab/sierra.png     # geometry cost

python3 compare.py            # the matched-fidelity table
```

`compare.py` reads the two `*-data.txt` files in the parent directory, so regenerate those first if you change anything upstream.

External tools required: `cwebp`, `dwebp`, `avifenc`, `avifdec`, `brotli`, and for report 08 also `cjxl`/`djxl` and `sips`.

The report 08 round needs a 3840×2160 source; the study used the macOS Sierra wallpaper at native size, which is likewise not redistributed here.

## A note on determinism

**Three** functions here sort by a total order rather than by score alone, and all three are load-bearing: `potts.go`'s merge-candidate heap, `potts.go`/`lossless.go`'s colour-predictor selection, and `potts2.go`'s Ising relaxation candidate loop. Each one previously picked between equally-scoring options in Go map-iteration order.

`lab/potts.go` sorts merge candidates by the energy key, then the pair index. The comparator originally ordered by key alone, and because Go randomizes `range` over a map, the initial heap was built in a different order each run and equal-key candidates popped differently. That produced up to 7% spread in bytes at the coarse end and a headline taken from a single lucky run. Report 06 covers what it invalidated. If you modify any of them, verify determinism across at least three runs before believing a number — `./lab hdnd <image>` exists precisely to make that check one command. Report 06 #6 and #7 cover what each one invalidated.
