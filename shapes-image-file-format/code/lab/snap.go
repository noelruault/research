package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// snapCmd is the composition test, and it is the one that matters for the product.
//
// A learned model (Apple Vision, rembg, SAM) already solves the SEMANTIC half: which pixels are the subject.
// This format's claim was never that it can out-segment them; it is that it is the cheapest, most stable, most addressable place to PUT that answer.
// So: take the model's pixel mask, snap it onto our partition by per-region majority vote, and measure what the snap costs and what it buys.
//
// Costs: IoU between the snapped mask and the model's original. Anything below ~1.0 is fidelity lost to the region grid.
// Buys: the selection becomes a list of region ids instead of a pixel set — addressable, O(regions) to edit, stable across re-encodes, and storable in the file.
//
// usage: snap <model-mask.png> <ours-render.png> <source.png> <out.png>
func snapCmd(args []string) {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: lab snap <model-mask.png> <ours-render.png> <source.png> <out.png>")
		os.Exit(2)
	}
	maskImg := decodePNG(args[0])
	ours := load(args[1])
	src := load(args[2])
	w, h := ours.W, ours.H
	mb := maskImg.Bounds()
	if mb.Dx() != w || mb.Dy() != h {
		fmt.Fprintf(os.Stderr, "fatal: mask %dx%d, render %dx%d\n", mb.Dx(), mb.Dy(), w, h)
		os.Exit(1)
	}
	// The model mask is greyscale: bright = subject.
	fg := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(maskImg.At(mb.Min.X+x, mb.Min.Y+y)).(color.NRGBA)
			fg[y*w+x] = (int(c.R)+int(c.G)+int(c.B))/3 >= 128
		}
	}

	lab, cols, _ := exactPartition(ours)
	n := len(cols)
	// Majority vote per region.
	inCount := make([]int, n)
	total := make([]int, n)
	for p, l := range lab {
		total[l]++
		if fg[p] {
			inCount[l]++
		}
	}
	sel := make([]bool, n)
	nsel := 0
	for i := range sel {
		sel[i] = inCount[i]*2 > total[i]
		if sel[i] {
			nsel++
		}
	}
	snapped := make([]bool, w*h)
	for p, l := range lab {
		snapped[p] = sel[l]
	}

	inter, union, mArea, sArea := 0, 0, 0, 0
	for p := range fg {
		if fg[p] {
			mArea++
		}
		if snapped[p] {
			sArea++
		}
		if fg[p] && snapped[p] {
			inter++
		}
		if fg[p] || snapped[p] {
			union++
		}
	}
	iou := 0.0
	if union > 0 {
		iou = float64(inter) / float64(union)
	}

	// Edge fidelity against the SOURCE, the neutral referee from METHODOLOGY.md section 2.
	edge := func(m []bool) (float64, int) {
		tot, cnt := 0.0, 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				p := y*w + x
				for _, q := range [][2]int{{x + 1, y}, {x, y + 1}} {
					if q[0] >= w || q[1] >= h {
						continue
					}
					np := q[1]*w + q[0]
					if m[p] == m[np] {
						continue
					}
					a := rgbToLab([3]float64{src.P[p*3], src.P[p*3+1], src.P[p*3+2]})
					b := rgbToLab([3]float64{src.P[np*3], src.P[np*3+1], src.P[np*3+2]})
					d := 0.0
					for i := 0; i < 3; i++ {
						e := a[i] - b[i]
						d += e * e
					}
					tot += math.Sqrt(d)
					cnt++
				}
			}
		}
		if cnt == 0 {
			return 0, 0
		}
		return tot / float64(cnt), cnt
	}
	em, cm := edge(fg)
	es, cs := edge(snapped)

	fmt.Printf("%-26s %10s %12s %12s %10s\n", "mask", "px", "selection is", "edge px", "dE on edge")
	fmt.Printf("%-26s %10d %12s %12d %10.2f\n", "model (pixels)", mArea, fmt.Sprintf("%d px", mArea), cm, em)
	fmt.Printf("%-26s %10d %12s %12d %10.2f\n", "snapped to our regions", sArea, fmt.Sprintf("%d/%d rg", nsel, n), cs, es)
	fmt.Printf("IoU(snapped, model) = %.4f   -- what the region grid costs\n", iou)

	// Picture: source | model cut | snapped cut | disagreement.
	const gap = 4
	out := image.NewNRGBA(image.Rect(0, 0, w*4+gap*3, h))
	srcAt := func(x, y int) color.NRGBA {
		p := y*w + x
		return color.NRGBA{clamp8(src.P[p*3]), clamp8(src.P[p*3+1]), clamp8(src.P[p*3+2]), 255}
	}
	cut := func(m []bool) func(x, y int) color.NRGBA {
		return func(x, y int) color.NRGBA {
			if m[y*w+x] {
				return srcAt(x, y)
			}
			if ((x/8)+(y/8))%2 == 0 {
				return color.NRGBA{0xcc, 0xcc, 0xcc, 255}
			}
			return color.NRGBA{0x99, 0x99, 0x99, 255}
		}
	}
	blit := func(px int, f func(x, y int) color.NRGBA) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.SetNRGBA(px+x, y, f(x, y))
			}
		}
	}
	blit(0, srcAt)
	blit(w+gap, cut(fg))
	blit((w+gap)*2, cut(snapped))
	blit((w+gap)*3, func(x, y int) color.NRGBA {
		p := y*w + x
		if fg[p] != snapped[p] {
			return color.NRGBA{0xff, 0x20, 0x20, 255}
		}
		c := srcAt(x, y)
		g := uint8((int(c.R)+int(c.G)+int(c.B))/3/3 + 170)
		return color.NRGBA{g, g, g, 255}
	})
	f, err := os.Create(args[3])
	must(err)
	must(png.Encode(f, out))
	must(f.Close())
	fmt.Printf("wrote %s (source | model cut | snapped cut | disagreement)\n", args[3])
}

// p4SnapSelection reads a model's foreground mask and snaps it onto a partition by per-region
// majority vote, returning one instance id per region with 0 meaning background.
//
// The encode-time half of the detector evaluation: semantics from a learned model run once, geometry from the
// partition — which is exact on the pixel lattice, where the model's own output is quantised to
// the model's fixed analysis grid.
func p4SnapSelection(maskPath string, lab []int32, n, w, h int) []byte {
	m := decodePNG(maskPath)
	mb := m.Bounds()
	if mb.Dx() != w || mb.Dy() != h {
		fmt.Fprintf(os.Stderr, "fatal: selection mask %dx%d, image %dx%d\n", mb.Dx(), mb.Dy(), w, h)
		os.Exit(1)
	}
	in := make([]int, n)
	tot := make([]int, n)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(m.At(mb.Min.X+x, mb.Min.Y+y)).(color.NRGBA)
			l := lab[y*w+x]
			tot[l]++
			if (int(c.R)+int(c.G)+int(c.B))/3 >= 128 {
				in[l]++
			}
		}
	}
	out := make([]byte, n)
	for i := range out {
		if in[i]*2 > tot[i] {
			out[i] = 1 // one instance for now; the contract allows 0..255
		}
	}
	return out
}
