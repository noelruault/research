package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	// The two header-only modes take no image, so they are handled before the arity check.
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "csplithdr":
			csplitHeader()
			return
		case "cwidthhdr":
			cwidthHeader()
			return
		}
	}
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
	case "contourx": // contour turn-stream characterisation at every mark at or below the region cap
		mx, _ := strconv.Atoi(os.Args[3])
		contourx(os.Args[2], mx)
	case "turnprice": // contour turn-coder variants priced side by side; LABDUMP=<dir> keeps the partitions
		mx, _ := strconv.Atoi(os.Args[3])
		turnprice(os.Args[2], mx)
	case "turnload": // re-price partitions kept by turnprice, without re-walking the merge
		turnload(os.Args[2])
	case "affine":
		affine(os.Args[2])
	// Report 11, the colour entropy floor (floor.go).
	case "floor": // floor <image.png> [labels.bin ...]; FLOORDUMP=<dir> writes the residual byte streams
		floorCmd(os.Args[2:])
	case "floordec": // floordec <labels.bin> <stream.bin> <wmean|rct|medrct> <ref.png> [planar] — the decodability check
		floorDec(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "floordump": // floordump <image.png> <dir> — write the exact lossless partition for floordec
		floorDump(os.Args[2], os.Args[3])
	// Report 13, B10: the cross-channel transform adopted and the colour column re-priced (recolour.go).
	// Report 09's cross-plane pricer was written but never wired into dispatch; wallxexact prices every wall variant on the exact partition of whatever image it is given, which for a published render is that mark's own partition.
	// P3: the decoder-side causality check for the wall variants. Written with report 09, never wired in.
	case "wallcheck": // wallcheck <image.png>
		wallCheck(os.Args[2])
	case "wallxexact": // wallxexact <image.png>
		wallxExact(os.Args[2])
	// P4, the container (container.go): a real file, and a decoder that reads it and nothing else.
	case "p4enc": // p4enc <render.png> <out.shpc>
		p4EncCmd(os.Args[2:])
	case "p4dec": // p4dec <in.shpc> <out.png> [ref.png]
		p4DecCmd(os.Args[2:])
	case "recolour": // recolour <image.png> <tag> [publishedRegions] [src.png]; RCDUMP=<dir> writes the streams
		rcCmd(os.Args[2:])
	case "rcdec": // rcdec <labels.bin> <stream.bin> <rct|a> <ref.png> [coef.bin] — the decodability check
		coef := ""
		if len(os.Args) > 6 {
			coef = os.Args[6]
		}
		rcDec(os.Args[2], os.Args[3], os.Args[4], os.Args[5], coef)
	// Report 09, the CAE context-width arm (wallctx.go). Build without crossplane.go: the two files both define `tap` and `crackPlanes`, which is how they were run when report 09 was produced.
	case "wallctx":
		wallctxCmd(os.Args[2:])
	case "wallsel":
		wallselCmd(os.Args[2:])
	case "walleval":
		wallevalCmd(os.Args[2:])
	case "caecausal":
		caeCausalCmd(os.Args[2:])
	// Report 10, the contour junction-map arm (contourctx.go). Same build set.
	case "csplit":
		csplitCmd(os.Args[2:])
	case "cwidth":
		cwidthCmd(os.Args[2:])
	case "csel":
		cselCmd(os.Args[2:])
	case "ccausal":
		ccausalCmd(os.Args[2:])
	case "alphahist": // alphahist <sprite.png ...> — how much sprite alpha is soft rim vs interior translucency (DESIGN-ALPHA.md)
		alphaHistCmd(os.Args[2:])
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
