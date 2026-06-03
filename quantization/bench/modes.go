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
	default:
		return nil
	}
}

func runEmit(args []string) {
	fs := flag.NewFlagSet("emit", flag.ExitOnError)
	in := fs.String("in", "", "source image")
	out := fs.String("out", "", "output PNG")
	n := fs.Int("n", 16, "palette size")
	qname := fs.String("q", "pca", "quantizer: pca | refine | kmeans++ | median")
	_ = fs.Parse(args)

	q := quantizerByName(*qname)
	if q == nil {
		fmt.Fprintln(os.Stderr, "unknown quantizer:", *qname)
		os.Exit(2)
	}
	img := mustLoad(*in)
	pal := q.Quantize(img, *n)

	// Assign every pixel to its nearest palette color (Euclidean RGB), the
	// same mapping pngquant/ImageMagick apply when they write an indexed PNG.
	b := img.Bounds()
	out2 := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bb, _ := img.At(x, y).RGBA()
			idx := pal.Index(color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bb >> 8), A: 255})
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
