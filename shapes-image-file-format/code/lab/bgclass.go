package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
)

// bgclassCmd tests the supervised version of background removal.
// Report 33 flooded from the border with a drifting tolerance and failed.
// Here the caller POINTS AT examples instead: this is grass, remove it; this is the dog, keep it.
// Every region is then classified by which example its colour is nearest to.
// That is the cheapest possible learned rule, a 1-nearest-neighbour classifier in CIELAB, and it is the shape of what a "lift subject" gesture provides.
//
// Both arms run the IDENTICAL classifier on the IDENTICAL example colours.
// The only variable is what gets classified:
//
//	arm R — the region colours of our partition. One decision per region.
//	arm P — every pixel of the rival's decoded image. One decision per pixel.
//
// The metric that matters is not only cost.
// A per-pixel classifier can flip inside a smooth area: a shadowed white hair reads as grass, a sunlit blade reads as dog.
// That produces speckle needing morphological cleanup.
// A per-region classifier CANNOT speckle within a region, because the partition regularises the decision spatially for free.
// Mask component counts measure exactly that.
//
// usage: bgclass <source.png> <ours.png> <rival.png> <out.png> keep=x,y;x,y remove=x,y;x,y
func bgclassCmd(args []string) {
	if len(args) < 6 {
		fmt.Fprintln(os.Stderr, "usage: lab bgclass <source.png> <ours.png> <rival.png> <out.png> keep=x,y;.. remove=x,y;..")
		os.Exit(2)
	}
	src, ours, rival := load(args[0]), load(args[1]), load(args[2])
	w, h := src.W, src.H
	if ours.W != w || ours.H != h || rival.W != w || rival.H != h {
		fmt.Fprintln(os.Stderr, "fatal: the three images differ in size")
		os.Exit(1)
	}
	parse := func(s, prefix string) [][2]int {
		s = strings.TrimPrefix(s, prefix)
		var out [][2]int
		for _, pt := range strings.Split(s, ";") {
			if pt == "" {
				continue
			}
			xy := strings.Split(pt, ",")
			if len(xy) != 2 {
				fmt.Fprintf(os.Stderr, "fatal: bad point %q\n", pt)
				os.Exit(2)
			}
			x, _ := strconv.Atoi(xy[0])
			y, _ := strconv.Atoi(xy[1])
			if x < 0 || x >= w || y < 0 || y >= h {
				fmt.Fprintf(os.Stderr, "fatal: point %d,%d is outside %dx%d\n", x, y, w, h)
				os.Exit(2)
			}
			out = append(out, [2]int{x, y})
		}
		return out
	}
	keepPts, removePts := parse(args[4], "keep="), parse(args[5], "remove=")
	if len(keepPts) == 0 || len(removePts) == 0 {
		fmt.Fprintln(os.Stderr, "fatal: need at least one keep point and one remove point")
		os.Exit(2)
	}
	// Examples are sampled from the SOURCE, which is what a user is actually pointing at.
	sample := func(pts [][2]int) [][3]float64 {
		out := make([][3]float64, 0, len(pts))
		for _, p := range pts {
			i := (p[1]*w + p[0]) * 3
			out = append(out, rgbToLab([3]float64{src.P[i], src.P[i+1], src.P[i+2]}))
		}
		return out
	}
	keepLab, removeLab := sample(keepPts), sample(removePts)
	show := func(tag string, pts [][2]int) {
		for _, q := range pts {
			i := (q[1]*w + q[0]) * 3
			fmt.Printf("  %-6s (%3d,%3d) rgb(%3.0f,%3.0f,%3.0f)\n", tag, q[0], q[1], src.P[i], src.P[i+1], src.P[i+2])
		}
	}
	fmt.Printf("examples, sampled from the source and classified in CIELAB:\n")
	show("KEEP", keepPts)
	show("REMOVE", removePts)

	// isRemove: nearest example wins. Ties go to keep, so the subject is never lost on a tie.
	nearest := func(lab [3]float64, set [][3]float64) float64 {
		best := math.Inf(1)
		for _, c := range set {
			d := 0.0
			for i := 0; i < 3; i++ {
				e := lab[i] - c[i]
				d += e * e
			}
			if d < best {
				best = d
			}
		}
		return best
	}
	isRemove := func(rgb [3]float64) bool {
		lab := rgbToLab(rgb)
		return nearest(lab, removeLab) < nearest(lab, keepLab)
	}

	// ---- arm R: one decision per region --------------------------------------------------------
	lab, cols, _ := exactPartition(ours)
	n := len(cols)
	selR := make([]bool, n)
	for i, c := range cols {
		selR[i] = isRemove(c)
	}
	maskR := make([]bool, w*h)
	for p, l := range lab {
		maskR[p] = selR[l]
	}
	// ---- arm P: one decision per pixel ---------------------------------------------------------
	maskP := make([]bool, w*h)
	for p := 0; p < w*h; p++ {
		maskP[p] = isRemove([3]float64{rival.P[p*3], rival.P[p*3+1], rival.P[p*3+2]})
	}

	// STEELMAN THE PIXEL ARM.
	// Arm R classifies region colours that the partition has ALREADY spatially averaged; arm P classifies raw pixels with no regularisation whatever.
	// Comparing those two directly measures "averaged vs not averaged" as much as it measures the region graph, and no practitioner ships a raw per-pixel mask.
	// So give arm P the standard cheap fix: a majority (median) filter over the mask.
	// Applied to BOTH arms, so neither gets a knob the other is denied.
	majority := func(m []bool, r int) []bool {
		out := make([]bool, len(m))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				on, tot := 0, 0
				for dy := -r; dy <= r; dy++ {
					for dx := -r; dx <= r; dx++ {
						nx, ny := x+dx, y+dy
						if nx < 0 || nx >= w || ny < 0 || ny >= h {
							continue
						}
						tot++
						if m[ny*w+nx] {
							on++
						}
					}
				}
				out[y*w+x] = on*2 > tot
			}
		}
		return out
	}

	// components counts 4-connected runs of one mask value: the speckle measure.
	components := func(m []bool, val bool) int {
		seen := make([]bool, w*h)
		cnt := 0
		stack := make([]int, 0, 1024)
		for s := 0; s < w*h; s++ {
			if seen[s] || m[s] != val {
				continue
			}
			cnt++
			stack = append(stack[:0], s)
			seen[s] = true
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				x, y := p%w, p/w
				try := func(q int) {
					if !seen[q] && m[q] == val {
						seen[q] = true
						stack = append(stack, q)
					}
				}
				if x > 0 {
					try(p - 1)
				}
				if x < w-1 {
					try(p + 1)
				}
				if y > 0 {
					try(p - w)
				}
				if y < h-1 {
					try(p + w)
				}
			}
		}
		return cnt
	}
	// CONNECTIVITY POST-PASS — the owner's observation, and it is two distinct operations.
	//
	// 1. Hole filling. A background component that does not touch the image border is enclosed by
	//    the subject, so it is almost certainly a misclassified part of it (the cat's dark
	//    markings, the dog's eyes). Flip it to keep.
	// 2. Disconnected rejection. Of what remains kept, keep only the component containing the
	//    subject seed, taken as the centroid of the keep examples. Anything not connected to the
	//    subject is background however much it looks like it (the dog's tree block).
	//
	// Applied to BOTH arms with identical code. Falsification #14 was giving one arm a cleanup the
	// other did not get; this comment exists so nobody does it again.
	//
	// On the region arm both operations are available on the region adjacency graph at O(regions)
	// rather than O(pixels). They are run on the pixel mask here so the two arms are compared on
	// exactly the same computation; the cost claim is separate and is stated in the report.
	seedX, seedY := 0, 0
	for _, q := range keepPts {
		seedX += q[0]
		seedY += q[1]
	}
	seedX /= len(keepPts)
	seedY /= len(keepPts)

	label := func(m []bool, val bool) ([]int, int) {
		id := make([]int, w*h)
		for i := range id {
			id[i] = -1
		}
		n := 0
		stack := make([]int, 0, 1024)
		for s := 0; s < w*h; s++ {
			if id[s] >= 0 || m[s] != val {
				continue
			}
			stack = append(stack[:0], s)
			id[s] = n
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				x, y := p%w, p/w
				try := func(q int) {
					if id[q] < 0 && m[q] == val {
						id[q] = n
						stack = append(stack, q)
					}
				}
				if x > 0 {
					try(p - 1)
				}
				if x < w-1 {
					try(p + 1)
				}
				if y > 0 {
					try(p - w)
				}
				if y < h-1 {
					try(p + w)
				}
			}
			n++
		}
		return id, n
	}

	connectivity := func(m []bool) []bool {
		out := make([]bool, len(m))
		copy(out, m)
		// 1. fill background components that never touch the border
		bg, nbg := label(out, true)
		touches := make([]bool, nbg)
		for x := 0; x < w; x++ {
			if c := bg[x]; c >= 0 {
				touches[c] = true
			}
			if c := bg[(h-1)*w+x]; c >= 0 {
				touches[c] = true
			}
		}
		for y := 0; y < h; y++ {
			if c := bg[y*w]; c >= 0 {
				touches[c] = true
			}
			if c := bg[y*w+w-1]; c >= 0 {
				touches[c] = true
			}
		}
		for p := range out {
			if out[p] && !touches[bg[p]] {
				out[p] = false // enclosed "background" is really subject
			}
		}
		// 2. of the kept pixels, keep only the component holding the seed
		fg, nfg := label(out, false)
		if nfg > 0 {
			want := fg[seedY*w+seedX]
			if want < 0 {
				// The seed centroid landed on a removed pixel; fall back to the largest component.
				sz := make([]int, nfg)
				for p := range out {
					if !out[p] {
						sz[fg[p]]++
					}
				}
				best := 0
				for i, v := range sz {
					if v > sz[best] {
						best = i
					}
				}
				want = best
			}
			for p := range out {
				if !out[p] && fg[p] != want {
					out[p] = true // kept, but not attached to the subject
				}
			}
		}
		return out
	}
	// CONN=0 disables the post-pass and restores the raw classifier output, so an idea that turns
	// out to be wrong can be rolled back without reverting code.
	connR, connP := maskR, maskP
	connOn := os.Getenv("CONN") != "0"
	if connOn {
		connR, connP = connectivity(maskR), connectivity(maskP)
	}

	pxR, pxP, agree := 0, 0, 0
	for p := range maskR {
		if maskR[p] {
			pxR++
		}
		if maskP[p] {
			pxP++
		}
		if maskR[p] == maskP[p] {
			agree++
		}
	}
	fmt.Printf("%-18s %10s %12s %12s %12s\n", "arm", "decisions", "removed px", "bg blobs", "fg blobs")
	fmt.Printf("%-18s %10d %12d %12d %12d\n", "R  per region", n, pxR, components(maskR, true), components(maskR, false))
	fmt.Printf("%-18s %10d %12d %12d %12d\n", "P  per pixel", w*h, pxP, components(maskP, true), components(maskP, false))
	fmt.Printf("decisions ratio %.0fx; masks agree on %.2f%% of pixels\n",
		float64(w*h)/float64(n), 100*float64(agree)/float64(w*h))

	// EDGE FIDELITY, judged by a NEUTRAL referee.
	// Report 33's "frayed edge" was scored against our own partition, which is a biased referee.
	// This scores against the SOURCE image instead: for every mask-edge pixel pair, how big is the real colour step there, in CIELAB?
	// A mask whose edges sit on genuine image edges scores high; a mask that cuts through flat areas scores low.
	// Neither arm owns the referee.
	edgeFidelity := func(m []bool) (float64, int) {
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
	er, ner := edgeFidelity(maskR)
	ep, nep := edgeFidelity(maskP)
	cR, cP := 0, 0
	for i := range connR {
		if connR[i] {
			cR++
		}
		if connP[i] {
			cP++
		}
	}
	ecR, _ := 0.0, 0
	_ = ecR
	if !connOn {
		fmt.Printf("\n-- connectivity post-pass DISABLED (CONN=0); the rows below repeat the raw masks --\n")
	} else {
		fmt.Printf("\n-- connectivity post-pass: fill enclosed background, then drop what is not attached to the subject --\n")
	}
	fmt.Printf("%-18s %12s %12s %12s\n", "arm", "removed px", "bg blobs", "fg blobs")
	fmt.Printf("%-18s %12d %12d %12d\n", "R  per region", cR, components(connR, true), components(connR, false))
	fmt.Printf("%-18s %12d %12d %12d\n", "P  per pixel", cP, components(connP, true), components(connP, false))

	fmt.Printf("\n-- edge fidelity: mean CIELAB step across the mask edge, judged on the SOURCE --\n")
	fmt.Printf("%-18s %10s %12s\n", "arm", "dE on edge", "edge px")
	fmt.Printf("%-18s %10.2f %12d\n", "R  per region", er, ner)
	fmt.Printf("%-18s %10.2f %12d\n", "P  per pixel", ep, nep)

	fmt.Printf("\n-- with a majority filter on the mask, applied to BOTH arms (the steelman) --\n")
	fmt.Printf("%-18s %8s %12s %12s %12s %10s\n", "arm", "radius", "removed px", "bg blobs", "fg blobs", "dE on edge")
	for _, r := range []int{1, 2, 3} {
		mr, mp := majority(maskR, r), majority(maskP, r)
		cr, cp := 0, 0
		for i := range mr {
			if mr[i] {
				cr++
			}
			if mp[i] {
				cp++
			}
		}
		fr, _ := edgeFidelity(mr)
		fp, _ := edgeFidelity(mp)
		fmt.Printf("%-18s %8d %12d %12d %12d %10.2f\n", "R  per region", 2*r+1, cr, components(mr, true), components(mr, false), fr)
		fmt.Printf("%-18s %8d %12d %12d %12d %10.2f\n", "P  per pixel", 2*r+1, cp, components(mp, true), components(mp, false), fp)
	}

	// ---- the picture ---------------------------------------------------------------------------
	const gap = 4
	out := image.NewNRGBA(image.Rect(0, 0, w*4+gap*3, h))
	srcAt := func(x, y int) color.NRGBA {
		p := y*w + x
		return color.NRGBA{clamp8(src.P[p*3]), clamp8(src.P[p*3+1]), clamp8(src.P[p*3+2]), 255}
	}
	cut := func(m []bool) func(x, y int) color.NRGBA {
		return func(x, y int) color.NRGBA {
			if !m[y*w+x] {
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
	blit(w+gap, cut(connR))
	blit((w+gap)*2, cut(connP))
	// Disagreement: where the two masks differ, so the speckle is legible rather than asserted.
	blit((w+gap)*3, func(x, y int) color.NRGBA {
		p := y*w + x
		c := srcAt(x, y)
		if maskR[p] != maskP[p] {
			return color.NRGBA{0xff, 0x20, 0x20, 255}
		}
		g := uint8((int(c.R) + int(c.G) + int(c.B)) / 3)
		g = uint8(int(g)/3 + 170)
		return color.NRGBA{g, g, g, 255}
	})
	f, err := os.Create(args[3])
	must(err)
	must(png.Encode(f, out))
	must(f.Close())
	fmt.Printf("wrote %s (source | per-region cut | per-pixel cut, both after the connectivity pass | raw-mask disagreement in red)\n", args[3])
}

// rgbToLab converts sRGB 0..255 to CIELAB (D65). Perceptual distance matters here: a shadowed white hair and a sunlit blade of grass are far apart in hue and close in RGB magnitude, which is exactly the confusion that sank the unsupervised flood in report 33.
func rgbToLab(c [3]float64) [3]float64 {
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	r, g, b := lin(c[0]), lin(c[1]), lin(c[2])
	x := (0.4124*r + 0.3576*g + 0.1805*b) / 0.95047
	y := 0.2126*r + 0.7152*g + 0.0722*b
	z := (0.0193*r + 0.1192*g + 0.9505*b) / 1.08883
	f := func(t float64) float64 {
		if t > 216.0/24389.0 {
			return math.Cbrt(t)
		}
		return (24389.0/27.0*t + 16) / 116
	}
	fx, fy, fz := f(x), f(y), f(z)
	return [3]float64{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}
