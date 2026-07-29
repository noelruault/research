package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: lab <sweep|potts|voronoi|lattice> <image.png> [args]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "sweep":
		sweep(os.Args[2])
	case "potts":
		potts(os.Args[2])
	case "voronoi":
		voronoi(os.Args[2])
	case "lattice":
		lattice(os.Args[2])
	case "psnr": // RGB PSNR, the same definition the repo harness uses, so external codecs are measured on the project's own metric
		a, b := load(os.Args[2]), load(os.Args[3])
		fmt.Printf("%.2f\n", psnrSSE(sseBetween(a, b), a.W*a.H))
	case "dump":
		n, _ := strconv.Atoi(os.Args[3])
		dumpQuant(os.Args[2], n, os.Args[4])
	case "lossless":
		lossless(os.Args[2])
	case "frontier":
		frontier(os.Args[2])
	case "crop":
		cropCmd(os.Args[2:])
	case "diff":
		diffCmd(os.Args[2:])
	case "stat":
		statCmd(os.Args[2:])
	case "hdnd":
		hdnd(os.Args[2])
	case "hdcheck":
		hdcheck(os.Args[2])
	case "hd":
		hd(os.Args[2], os.Args[3])
	case "affine":
		affine(os.Args[2])
	default:
		fmt.Fprintln(os.Stderr, "unknown mode")
		os.Exit(2)
	}
}

// sweep answers "is the region count an artifact of the palette size, and is there a phase transition in n?".
// For each n it reports the rect count, the true 4-connected region count, the boundary length, the fidelity, and the best raster bit cost.
func sweep(path string) {
	fmt.Printf("%-5s %8s %9s %9s %8s %10s %10s %10s\n",
		"n", "rects", "regions", "crackLen", "PSNR", "png", "order2", "bpp")
	for _, n := range []int{2, 4, 6, 8, 12, 16, 24, 32, 48, 64, 96, 128, 192, 256} {
		q := quantize(path, n)
		lab := make([]int32, len(q.Idx))
		for i, v := range q.Idx {
			lab[i] = int32(v)
		}
		orig := load(path)
		rec := q.paint()
		ps := psnrSSE(sseBetween(orig, rec), q.W*q.H)
		o2 := adaptiveBytes(q.Idx, q.W, q.H, 2, len(q.Pal))
		pngB := len(pngIndexed(q))
		fmt.Printf("%-5d %8d %9d %9d %8.2f %10s %10s %10.3f\n",
			n, q.Rects, components(lab, q.W, q.H), crackLen(lab, q.W, q.H), ps,
			kb(float64(pngB)), kb(o2), o2*8/float64(q.W*q.H))
	}
}

// dumpQuant writes the n-colour quantized grid as a PNG so the flat-art eval is a genuinely flat image, the regime the format claims.
func dumpQuant(path string, n int, out string) {
	q := quantize(path, n)
	q.paint().writePNG(out)
	lab := make([]int32, len(q.Idx))
	for i, v := range q.Idx {
		lab[i] = int32(v)
	}
	fmt.Printf("%s: %dx%d n=%d rects=%d regions=%d crack=%d indexedPNG=%d B\n",
		out, q.W, q.H, n, q.Rects, components(lab, q.W, q.H), crackLen(lab, q.W, q.H), len(pngIndexed(q)))
}
