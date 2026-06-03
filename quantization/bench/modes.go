package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

// modes.go — the `emit` and `score` subcommands that make a fair, uniform
// shootout possible (report 10). `emit` writes OUR quantized PNG using the
// same nearest-color assignment every tool uses; `score` measures any two
// PNGs (original vs quantized) with the same CIEDE2000 scorer, so pngquant,
// ImageMagick, and our quantizer are all judged identically.

// quantizerByName maps the shootout's CLI labels to the pieces we ship-test.
func quantizerByName(name string) Quantizer {
	rgb := RGBSpace{}
	switch name {
	case "pca": // the deterministic default
		return Divisive{Space: rgb, PCA: true}
	case "refine": // the quality mode: PCA-divisive init + k-means refine
		return KMeans{Space: rgb, Init: Divisive{Space: rgb, PCA: true}, Iters: 10}
	case "kmeans++":
		return KMeans{Space: rgb, Seed: "kmeans++", Iters: 10}
	case "median":
		return MedianCut{}
	case "pca-oklab": // divisive in OKLab
		return Divisive{Space: OKLabSpace{}, PCA: true}
	case "refine-oklab": // OKLab-divisive init + k-means refine, all in OKLab (the champion)
		return KMeans{Space: OKLabSpace{}, Init: Divisive{Space: OKLabSpace{}, PCA: true}, Iters: 10}
	case "refine-oklab-ew": // + error-weighted refinement (libimagequant trick)
		return KMeans{Space: OKLabSpace{}, Init: Divisive{Space: OKLabSpace{}, PCA: true}, Iters: 10, ErrWeight: true}
	case "multirestart-oklab": // SA substitute: keep-best over restarts
		return MultiRestart{Space: OKLabSpace{}, Restarts: 8, Iters: 10}
	case "hyab-oklab": // HyAB metric refinement
		return HyABKMeans{Init: Divisive{Space: OKLabSpace{}, PCA: true}, Iters: 10}
	case "pnn-oklab": // PNN agglomerative init + k-means refine
		return KMeans{Space: OKLabSpace{}, Init: PNN{Space: OKLabSpace{}}, Iters: 10}
	case "pnn-oklab-raw": // PNN init only (no refine), to isolate init quality
		return PNN{Space: OKLabSpace{}}
	// --- cross-domain (report 09) ---
	case "spacecurve": // crypto/db: Morton Z-order, equal-weight cut
		return SpaceCurve{}
	case "spacecurve-refine":
		return KMeans{Space: OKLabSpace{}, Init: SpaceCurve{}, Iters: 10}
	case "mst": // astrophysics: MST/FoF single-linkage
		return MSTCluster{}
	case "mst-refine":
		return KMeans{Space: OKLabSpace{}, Init: MSTCluster{}, Iters: 10}
	case "detanneal": // stat-mech: deterministic annealing
		return DetAnneal{Init: Divisive{Space: OKLabSpace{}, PCA: true}, Steps: 30}
	default:
		return nil
	}
}

// nearestInSpace finds the palette index minimizing Euclidean distance in sp
// — a kd-tree-compatible perceptual assignment when sp is OKLab. This is the
// "matched assignment" lever: cluster AND assign in the same perceptual space.
func nearestInSpace(palVecs []Vec3, v Vec3) int {
	best, bestD := 0, v.dist2(palVecs[0])
	for i := 1; i < len(palVecs); i++ {
		if d := v.dist2(palVecs[i]); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func runEmit(args []string) {
	fs := flag.NewFlagSet("emit", flag.ExitOnError)
	in := fs.String("in", "", "source image")
	out := fs.String("out", "", "output PNG")
	n := fs.Int("n", 16, "palette size")
	qname := fs.String("q", "pca", "quantizer: pca|refine|kmeans++|median|pca-oklab|refine-oklab")
	space := fs.String("space", "rgb", "assignment space: rgb | oklab (perceptual, matched)")
	_ = fs.Parse(args)

	q := quantizerByName(*qname)
	if q == nil {
		fmt.Fprintln(os.Stderr, "unknown quantizer:", *qname)
		os.Exit(2)
	}
	img := mustLoad(*in)
	pal := q.Quantize(img, *n)

	// Assignment: Euclidean RGB by default (what pngquant/ImageMagick do when
	// they write an indexed PNG), or Euclidean OKLab when -space oklab — the
	// matched perceptual assignment we want to test against the RGB default.
	var sp Space
	var palVecs []Vec3
	if *space == "oklab" {
		sp = OKLabSpace{}
		palVecs = make([]Vec3, len(pal))
		for i, c := range pal {
			r, g, b, _ := c.RGBA()
			palVecs[i] = sp.FromRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}
	}

	b := img.Bounds()
	out2 := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bb, _ := img.At(x, y).RGBA()
			sr, sg, sb := uint8(r>>8), uint8(g>>8), uint8(bb>>8)
			var idx int
			if sp != nil {
				idx = nearestInSpace(palVecs, sp.FromRGB(sr, sg, sb))
			} else {
				idx = pal.Index(color.RGBA{R: sr, G: sg, B: sb, A: 255})
			}
			pr, pg, pbv, _ := pal[idx].RGBA()
			out2.SetRGBA(x, y, color.RGBA{R: uint8(pr >> 8), G: uint8(pg >> 8), B: uint8(pbv >> 8), A: 255})
		}
	}
	writePNG(*out, out2)
}

func runScore(args []string) {
	fs := flag.NewFlagSet("score", flag.ExitOnError)
	a := fs.String("a", "", "original image")
	b := fs.String("b", "", "quantized image")
	label := fs.String("label", "", "row label to print")
	_ = fs.Parse(args)

	s := scorePair(mustLoad(*a), mustLoad(*b))
	fmt.Printf("%s MSE=%.4f PSNR=%.4f meanDE=%.4f p95DE=%.4f\n",
		*label, s.MSE, s.PSNR, s.MeanDE, s.P95DE)
}

// scorePair measures quantized against original pixel-for-pixel — used for any
// tool's output, so all tools are scored identically.
func scorePair(orig, quant image.Image) Score {
	b := orig.Bounds()
	n := b.Dx() * b.Dy()
	if n == 0 {
		return Score{}
	}
	var sumSq float64
	des := make([]float64, 0, n)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r0, g0, b0, _ := orig.At(x, y).RGBA()
			r1, g1, b1, _ := quant.At(x, y).RGBA()
			sr, sg, sb := uint8(r0>>8), uint8(g0>>8), uint8(b0>>8)
			qr, qg, qb := uint8(r1>>8), uint8(g1>>8), uint8(b1>>8)
			dr, dg, db := float64(sr)-float64(qr), float64(sg)-float64(qg), float64(sb)-float64(qb)
			sumSq += dr*dr + dg*dg + db*db
			des = append(des, CIEDE2000(RGBToLab(sr, sg, sb), RGBToLab(qr, qg, qb)))
		}
	}
	mse := sumSq / float64(n*3)
	psnr := math.Inf(1)
	if mse > 0 {
		psnr = 10 * math.Log10(255*255/mse)
	}
	sort.Float64s(des)
	var sum float64
	for _, d := range des {
		sum += d
	}
	return Score{MSE: mse, PSNR: psnr, MeanDE: sum / float64(len(des)),
		P95DE: des[int(math.Min(float64(len(des)-1), float64(len(des))*0.95))]}
}

func mustLoad(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, path, err)
		os.Exit(1)
	}
	return img
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
