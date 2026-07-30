package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
)

// bgcutCmd tests the claim the whole structured-image pitch rests on: that selecting a background is cheaper and cleaner over a REGION GRAPH than over a pixel grid.
//
// Both arms run the SAME algorithm — flood from the image border, absorb a neighbour when its colour is within a tolerance of the seed set's running mean — so the only variable is what the flood traverses. Arm R walks regions of this coder's partition. Arm P walks pixels of the rival's decoded image. Same seeds, same tolerance, same acceptance test: nothing is given to one side that is denied the other (PRINCIPLES.md #2).
//
// What it reports:
//   - steps: how many flood decisions each arm made. This is the O(regions) vs O(pixels) claim,
//     and it is the one thing that cannot be argued with.
//   - mask agreement: how much the two masks actually differ, so "cleaner" is not asserted.
//   - boundary pixels: how many mask-edge pixels are NOT on a region boundary. For arm R this is
//     zero by construction — the mask edge IS a stored boundary. For arm P it is the count of
//     places the mask cuts through the middle of a region, which is where a raster wand frays.
//
// usage: bgcut <source.png> <ours-render.png> <rival.png> <out.png> [tolerance]
func bgcutCmd(args []string) {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: lab bgcut <source.png> <ours-render.png> <rival.png> <out.png> [tolerance]")
		os.Exit(2)
	}
	tol := 28.0
	if len(args) > 4 {
		tol, _ = strconv.ParseFloat(args[4], 64)
	}
	src, ours, rival := load(args[0]), load(args[1]), load(args[2])
	w, h := src.W, src.H
	if ours.W != w || ours.H != h || rival.W != w || rival.H != h {
		fmt.Fprintln(os.Stderr, "fatal: the three images differ in size")
		os.Exit(1)
	}

	// ---- arm R: flood the region graph of our own partition -----------------------------------
	lab, cols, _ := exactPartition(ours)
	n := len(cols)
	adj := make([]map[int32]bool, n)
	for i := range adj {
		adj[i] = map[int32]bool{}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := lab[y*w+x]
			if x+1 < w {
				if b := lab[y*w+x+1]; b != a {
					adj[a][b], adj[b][a] = true, true
				}
			}
			if y+1 < h {
				if b := lab[(y+1)*w+x]; b != a {
					adj[a][b], adj[b][a] = true, true
				}
			}
		}
	}
	area := make([]float64, n)
	for _, l := range lab {
		area[l]++
	}
	selR := make([]bool, n)
	var sum [3]float64
	var cnt float64
	var queue []int32
	push := func(r int32) {
		if selR[r] {
			return
		}
		selR[r] = true
		for c := 0; c < 3; c++ {
			sum[c] += cols[r][c] * area[r]
		}
		cnt += area[r]
		queue = append(queue, r)
	}
	for x := 0; x < w; x++ {
		push(lab[x])
		push(lab[(h-1)*w+x])
	}
	for y := 0; y < h; y++ {
		push(lab[y*w])
		push(lab[y*w+w-1])
	}
	stepsR := 0
	for len(queue) > 0 {
		r := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		mean := [3]float64{sum[0] / cnt, sum[1] / cnt, sum[2] / cnt}
		for nb := range adj[r] {
			stepsR++
			if selR[nb] {
				continue
			}
			d := 0.0
			for c := 0; c < 3; c++ {
				e := cols[nb][c] - mean[c]
				d += e * e
			}
			if math.Sqrt(d) <= tol {
				push(nb)
			}
		}
	}
	maskR := make([]bool, w*h)
	for p, l := range lab {
		maskR[p] = selR[l]
	}

	// ---- arm P: the same flood over the rival's pixels -----------------------------------------
	maskP := make([]bool, w*h)
	var psum [3]float64
	var pcnt float64
	var pq []int
	rgb := func(im *Img, p int) [3]float64 {
		return [3]float64{im.P[p*3], im.P[p*3+1], im.P[p*3+2]}
	}
	ppush := func(p int) {
		if maskP[p] {
			return
		}
		maskP[p] = true
		c := rgb(rival, p)
		for i := 0; i < 3; i++ {
			psum[i] += c[i]
		}
		pcnt++
		pq = append(pq, p)
	}
	for x := 0; x < w; x++ {
		ppush(x)
		ppush((h-1)*w + x)
	}
	for y := 0; y < h; y++ {
		ppush(y * w)
		ppush(y*w + w - 1)
	}
	stepsP := 0
	for len(pq) > 0 {
		p := pq[len(pq)-1]
		pq = pq[:len(pq)-1]
		mean := [3]float64{psum[0] / pcnt, psum[1] / pcnt, psum[2] / pcnt}
		x, y := p%w, p/w
		for _, q := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
			if q[0] < 0 || q[0] >= w || q[1] < 0 || q[1] >= h {
				continue
			}
			stepsP++
			np := q[1]*w + q[0]
			if maskP[np] {
				continue
			}
			c := rgb(rival, np)
			d := 0.0
			for i := 0; i < 3; i++ {
				e := c[i] - mean[i]
				d += e * e
			}
			if math.Sqrt(d) <= tol {
				ppush(np)
			}
		}
	}

	// ---- how ragged is each mask's edge? ------------------------------------------------------
	// A mask edge pixel whose neighbour across the edge is in the SAME region of our partition means the mask cut through a region's interior. Arm R cannot do this; arm P does it wherever its tolerance test flips mid-region, which is the fraying a raster wand is known for.
	fray := func(m []bool) int {
		bad := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				p := y*w + x
				for _, q := range [][2]int{{x + 1, y}, {x, y + 1}} {
					if q[0] >= w || q[1] >= h {
						continue
					}
					np := q[1]*w + q[0]
					if m[p] != m[np] && lab[p] == lab[np] {
						bad++
					}
				}
			}
		}
		return bad
	}
	agree := 0
	selPx, selPxP := 0, 0
	for p := range maskR {
		if maskR[p] == maskP[p] {
			agree++
		}
		if maskR[p] {
			selPx++
		}
		if maskP[p] {
			selPxP++
		}
	}
	nsel := 0
	for _, v := range selR {
		if v {
			nsel++
		}
	}
	fmt.Printf("%-22s %8s %10s %12s %10s\n", "arm", "steps", "selected", "sel pixels", "frayed edge")
	fmt.Printf("%-22s %8d %10s %12d %10d\n", "R  region graph", stepsR,
		fmt.Sprintf("%d/%d rg", nsel, n), selPx, fray(maskR))
	fmt.Printf("%-22s %8d %10s %12d %10d\n", "P  pixel grid", stepsP, "-", selPxP, fray(maskP))
	fmt.Printf("steps ratio %.1fx in favour of the region graph; masks agree on %.2f%% of pixels\n",
		float64(stepsP)/float64(stepsR), 100*float64(agree)/float64(w*h))

	// ---- the picture --------------------------------------------------------------------------
	const gap = 4
	out := image.NewNRGBA(image.Rect(0, 0, w*4+gap*3, h))
	blit := func(px int, f func(x, y int) color.NRGBA) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.SetNRGBA(px+x, y, f(x, y))
			}
		}
	}
	srcAt := func(x, y int) color.NRGBA {
		p := y*w + x
		return color.NRGBA{clamp8(src.P[p*3]), clamp8(src.P[p*3+1]), clamp8(src.P[p*3+2]), 255}
	}
	tint := func(m []bool) func(x, y int) color.NRGBA {
		return func(x, y int) color.NRGBA {
			c := srcAt(x, y)
			if m[y*w+x] {
				return color.NRGBA{uint8((int(c.R) + 255) / 2), uint8(int(c.G) / 2), uint8((int(c.B) + 255) / 2), 255}
			}
			return c
		}
	}
	blit(0, srcAt)
	blit(w+gap, tint(maskR))
	blit((w+gap)*2, tint(maskP))
	// Cutout over a checkerboard: what you would actually ship.
	blit((w+gap)*3, func(x, y int) color.NRGBA {
		if !maskR[y*w+x] {
			return srcAt(x, y)
		}
		if ((x/8)+(y/8))%2 == 0 {
			return color.NRGBA{0xcc, 0xcc, 0xcc, 255}
		}
		return color.NRGBA{0x99, 0x99, 0x99, 255}
	})
	f, err := os.Create(args[3])
	must(err)
	must(png.Encode(f, out))
	must(f.Close())
	fmt.Printf("wrote %s (source | region-flood | pixel-flood | cutout)\n", args[3])
}
