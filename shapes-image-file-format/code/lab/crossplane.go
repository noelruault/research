package main

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

// Cross-plane wall coding.
//
// caeBytes codes the crack map as two planes in sequence: the whole V plane (vertical cracks, between horizontally adjacent pixels) with a 10-bit V-only context, then the whole Hz plane with a 10-bit context of six Hz taps and four V taps.
// So Hz already conditions on V; V conditions on nothing from Hz, because V is finished before Hz starts.
//
// The lattice makes the two planes locally dependent in a precise way. At the lattice vertex between the four pixels (x-1,y-1),(x,y-1),(x-1,y),(x,y) the four incident crack bits are
//
//	V(x-1,y-1) above, V(x-1,y) below, Hz(x-1,y-1) left, Hz(x,y-1) right
//
// and these are the four "adjacent labels differ" indicators around a 4-cycle, so exactly one of them cannot be set on its own: three zeros force the fourth to zero, and a single one is impossible.
// Knowing three of the four is therefore a hard constraint on the fourth, and that constraint is what cross-plane conditioning can buy.
//
// Counting the baseline's taps against that structure explains where the lever is and is not:
//
//   - Hz(x,y) has all six of its two vertices' other members already — V(x-1,y), V(x-1,y+1), Hz(x-1,y)
//     around the left vertex and V(x,y), V(x,y+1), Hz(x+1,y) around the right one. The cross-plane
//     dependence is already fully spent on the Hz side.
//   - V(x,y) has one: V(x,y-1). Its two vertices' Hz members — Hz(x,y-1), Hz(x+1,y-1) above and
//     Hz(x,y), Hz(x+1,y) below — are all invisible to it.
//
// Hence the two schedules under test: interleave the planes per pixel, or code Hz first, either of which
// lets V see Hz. Both are re-decompositions of the same joint entropy H(V,Hz), so the chain rule says the
// ideal total is unchanged and only model capacity can move the number. That is exactly what gets measured.

// tap is one context bit: plane pl sampled at offset (dx,dy) from the crack edge being coded.
type tap struct {
	pl     int // 0 = V, 1 = Hz
	dx, dy int
}

// Coding schedules. The decoder follows the same order, so a tap is legal only if it lands strictly earlier.
const (
	twoPassV = iota // whole V plane, then whole Hz plane — what caeBytes does
	twoPassH        // whole Hz plane, then whole V plane
	interVH         // single raster scan; at each pixel V(x,y) then Hz(x,y)
	interHV         // single raster scan; at each pixel Hz(x,y) then V(x,y)
)

// ordKey places one crack bit in the schedule; lexicographic order on the result is coding order.
func ordKey(mode, pl, x, y int) [3]int {
	switch mode {
	case twoPassV:
		return [3]int{pl, y, x}
	case twoPassH:
		return [3]int{1 - pl, y, x}
	case interVH:
		return [3]int{y, x, pl}
	default: // interHV
		return [3]int{y, x, 1 - pl}
	}
}

func keyBefore(a, b [3]int) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

type wallVariant struct {
	name     string
	mode     int
	tapsV    []tap // context for a V bit
	tapsH    []tap // context for an Hz bit
	nonCausal bool // set only for the published baseline, whose Hz context reads Hz(x+1,y)
	note     string
}

// checkCausal fails loudly if any tap reads a crack the decoder has not decoded yet.
// The published baseline is the one exemption and is flagged as such; every variant compared against it must pass.
func (v wallVariant) checkCausal() error {
	for _, t := range v.tapsV {
		if !keyBefore(ordKey(v.mode, t.pl, t.dx, t.dy), ordKey(v.mode, 0, 0, 0)) {
			return fmt.Errorf("%s: V context tap %v is not causal under this schedule", v.name, t)
		}
	}
	for _, t := range v.tapsH {
		if !keyBefore(ordKey(v.mode, t.pl, t.dx, t.dy), ordKey(v.mode, 1, 0, 0)) {
			return fmt.Errorf("%s: Hz context tap %v is not causal under this schedule", v.name, t)
		}
	}
	return nil
}

// crackPlanes derives the two crack-edge planes from a label field, identically to caeBytes.
func crackPlanes(lab []int32, w, h int) (V, Hz []byte) {
	V = make([]byte, w*h)
	Hz = make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				V[p] = 1
			}
			if y < h-1 && lab[p] != lab[p+w] {
				Hz[p] = 1
			}
		}
	}
	return
}

// priceVariant returns the cost in bytes of the V plane and of the Hz plane under one schedule and tap set.
// The coder is byte-for-byte the same machinery as caeBytes — one adaptive binary model per context, cost measured as its cross-entropy — so only the schedule and the taps differ.
func priceVariant(v wallVariant, V, Hz []byte, w, h int) (bV, bH float64) {
	get := func(pl, x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		if pl == 0 {
			return int(V[y*w+x])
		}
		return int(Hz[y*w+x])
	}
	mv := make([]binModel, 1<<uint(len(v.tapsV)))
	mh := make([]binModel, 1<<uint(len(v.tapsH)))
	codeV := func(x, y int) {
		ctx := 0
		for i, t := range v.tapsV {
			ctx |= get(t.pl, x+t.dx, y+t.dy) << uint(i)
		}
		bV += mv[ctx].cost(int(V[y*w+x]))
	}
	codeH := func(x, y int) {
		ctx := 0
		for i, t := range v.tapsH {
			ctx |= get(t.pl, x+t.dx, y+t.dy) << uint(i)
		}
		bH += mh[ctx].cost(int(Hz[y*w+x]))
	}
	switch v.mode {
	case twoPassV:
		for y := 0; y < h; y++ {
			for x := 0; x < w-1; x++ {
				codeV(x, y)
			}
		}
		for y := 0; y < h-1; y++ {
			for x := 0; x < w; x++ {
				codeH(x, y)
			}
		}
	case twoPassH:
		for y := 0; y < h-1; y++ {
			for x := 0; x < w; x++ {
				codeH(x, y)
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w-1; x++ {
				codeV(x, y)
			}
		}
	case interVH:
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < w-1 {
					codeV(x, y)
				}
				if y < h-1 {
					codeH(x, y)
				}
			}
		}
	default: // interHV
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if y < h-1 {
					codeH(x, y)
				}
				if x < w-1 {
					codeV(x, y)
				}
			}
		}
	}
	return bV / 8, bH / 8
}

// --- tap sets -----------------------------------------------------------------------------------------
//
// Bit position within a context is irrelevant to the cost: permuting the bits is a bijection on context ids and every model is independent, so only the *set* of taps matters.

// baseline, exactly as caeBytes writes them
var baseTapsV = []tap{{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1}, {0, 2, -1}, {0, 0, -2}, {0, -1, -2}, {0, 1, -2}, {0, -2, -1}}
var baseTapsH = []tap{{1, -1, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 1, 0}, {0, 0, 0}, {0, -1, 0}, {0, 0, 1}, {0, -1, 1}, {1, -2, 0}}

// the same ten-tap shape as baseTapsV, transposed onto the Hz plane: a fully causal own-plane-only context
var ownTapsH = []tap{{1, -1, 0}, {1, -2, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 2, -1}, {1, 0, -2}, {1, -1, -2}, {1, 1, -2}, {1, -2, -1}}

func variants() []wallVariant {
	return []wallVariant{
		{name: "base", mode: twoPassV, tapsV: baseTapsV, tapsH: baseTapsH, nonCausal: true,
			note: "published caeBytes; Hz reads Hz(x+1,y), which is not causal"},

		{name: "noCross", mode: twoPassV, tapsV: baseTapsV, tapsH: ownTapsH,
			note: "Hz loses its four V taps: what the existing cross-plane conditioning is worth"},

		{name: "baseFix", mode: twoPassV, tapsV: baseTapsV,
			tapsH: []tap{{1, -1, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {0, 1, 0}, {0, 0, 0}, {0, -1, 0}, {0, 0, 1}, {0, -1, 1}, {1, -2, 0}},
			note:  "baseline with the non-causal Hz(x+1,y) tap swapped for V(x+1,y)"},

		{name: "hzFirst", mode: twoPassH, tapsV: []tap{
			{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1}, {0, 2, -1},
			{1, 0, -1}, {1, 1, -1}, {1, 0, 0}, {1, 1, 0}},
			tapsH: ownTapsH,
			note:  "planes swapped: Hz alone first, then V with both its vertices' Hz members"},

		{name: "interVH", mode: interVH, tapsV: []tap{
			{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1}, {0, 2, -1}, {0, 0, -2}, {0, -1, -2},
			{1, 0, -1}, {1, 1, -1}},
			tapsH: []tap{
				{1, -1, 0}, {1, -2, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 2, -1},
				{0, 0, 0}, {0, -1, 0}, {0, 0, -1}, {0, 1, -1}},
			note: "one scan, V then Hz per pixel: V completes its upper vertex, Hz loses its lower one"},

		{name: "interHV", mode: interHV, tapsV: []tap{
			{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1}, {0, 2, -1},
			{1, 0, -1}, {1, 1, -1}, {1, 0, 0}, {1, -1, 0}},
			tapsH: []tap{
				{1, -1, 0}, {1, -2, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 2, -1}, {1, 0, -2}, {1, -1, -2},
				{0, -1, 0}, {0, 0, -1}},
			note: "one scan, Hz then V per pixel: V also gets Hz(x,y); Hz keeps only two V taps"},

		// Capacity controls. Twelve taps is 4,096 models per plane, which at 8.3M samples is ~2,000 samples per context and at 147K samples is ~36 — report 06 #2's starvation regime exactly, so these two rows are only meaningful at 4K.
		{name: "base12", mode: twoPassV,
			tapsV: append(append([]tap{}, baseTapsV...), tap{0, -3, 0}, tap{0, 2, -2}),
			tapsH: append(append([]tap{}, baseTapsH...), tap{1, 0, -2}, tap{0, 1, 1}), nonCausal: true,
			note: "baseline plus two own-plane taps each: capacity control, no new cross-plane information"},

		// Capacity-matched pair, 4,352 models each: the same total budget spent on whichever plane the schedule leaves carrying the cost. Under twoPassV that is V; under interVH it is Hz.
		{name: "baseAsym", mode: twoPassV,
			tapsV: append(append([]tap{}, baseTapsV...), tap{0, -3, 0}, tap{0, 2, -2}),
			tapsH: []tap{{1, -1, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {0, 0, 0}, {0, -1, 0}, {0, 0, 1}, {0, -1, 1}},
			note:  "twoPassV with a 12-bit V context and an 8-bit Hz context, fully causal"},

		{name: "interAsym", mode: interVH,
			tapsV: []tap{{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1}, {0, 2, -1}, {1, 0, -1}, {1, 1, -1}},
			tapsH: []tap{
				{1, -1, 0}, {1, -2, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 2, -1},
				{0, 0, 0}, {0, -1, 0}, {0, 0, -1}, {0, 1, -1}, {0, -2, 0}, {0, -1, -1}},
			note: "interVH with an 8-bit V context and a 12-bit Hz context — same model count as baseAsym"},

		{name: "inter12", mode: interVH,
			tapsV: []tap{
				{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1}, {0, 2, -1}, {0, 0, -2}, {0, -1, -2},
				{1, 0, -1}, {1, 1, -1}, {1, -1, -1}, {1, 2, -1}},
			tapsH: []tap{
				{1, -1, 0}, {1, -2, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 2, -1},
				{0, 0, 0}, {0, -1, 0}, {0, 0, -1}, {0, 1, -1}, {0, -2, 0}, {0, -1, -1}},
			note: "interVH with twelve taps per plane"},
	}
}

// wallxRow prices every variant on one partition. Variants are independent, so they run concurrently.
func wallxRow(lab []int32, w, h int) []struct{ bV, bH float64 } {
	V, Hz := crackPlanes(lab, w, h)
	vs := variants()
	out := make([]struct{ bV, bH float64 }, len(vs))
	var wg sync.WaitGroup
	for i := range vs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out[i].bV, out[i].bH = priceVariant(vs[i], V, Hz, w, h)
		}(i)
	}
	wg.Wait()
	return out
}

// wallx walks the same scale-space hd walks — the rate-distortion merge, six sweeps of Ising relaxation, the same geometric ladder of region counts — and prices every wall-coder variant on each partition, so baseline and variants are always compared on identical labels.
func wallx(path string) {
	for _, v := range variants() {
		if err := v.checkCausal(); err != nil {
			if !v.nonCausal {
				fmt.Fprintln(os.Stderr, "fatal:", err)
				os.Exit(1)
			}
			fmt.Printf("# note %s: %v\n", v.name, err)
		}
	}

	im := load(path)
	npix := im.W * im.H
	t0 := time.Now()
	vs := variants()

	fmt.Printf("# %s %dx%d — wall coder variants on identical partitions\n", path, im.W, im.H)
	for _, v := range vs {
		fmt.Printf("#   %-9s ctx %d/%d bits — %s\n", v.name, len(v.tapsV), len(v.tapsH), v.note)
	}
	fmt.Printf("%-9s %10s %8s", "regions", "crack", "psnr")
	for _, v := range vs {
		fmt.Printf(" %11s", v.name)
	}
	fmt.Printf("   | per-plane bytes (V/Hz) per variant\n")

	marks := hdMarks(npix)
	mi := 0
	m := newMerger(im)
	for mi < len(marks) && marks[mi] >= m.nreg {
		mi++
	}
	m.runRD(marks[len(marks)-1], func(mm *merger, lambda float64) {
		for mi < len(marks) && mm.nreg == marks[mi] {
			base := make([]int32, len(mm.parent))
			for i := range base {
				base[i] = mm.find(int32(i))
			}
			lab := relabelComponents(base, im.W, im.H)
			nreg := 0
			for _, l := range lab {
				if int(l)+1 > nreg {
					nreg = int(l) + 1
				}
			}
			lab = relax(im, lab, nreg, lambda*bitsPerEdge, 6)
			nr, ps, bb, _, _ := priceSeg(im, lab)
			cl := crackLen(lab, im.W, im.H)

			rows := wallxRow(lab, im.W, im.H)
			// The "base" variant must reproduce caeBytes exactly; if it does not, nothing below is comparable.
			if d := math.Abs(rows[0].bV + rows[0].bH - bb); d > 1e-6 {
				fmt.Fprintf(os.Stderr, "fatal: base variant disagrees with caeBytes by %.6f B at %d regions\n", d, nr)
				os.Exit(1)
			}
			fmt.Printf("%-9d %10d %8.2f", nr, cl, ps)
			for _, r := range rows {
				fmt.Printf(" %11.0f", r.bV+r.bH)
			}
			fmt.Printf("   |")
			for _, r := range rows {
				fmt.Printf(" %.0f/%.0f", r.bV, r.bH)
			}
			fmt.Println()
			os.Stdout.Sync()
			fmt.Fprintf(os.Stderr, "mark %d (relaxed to %d) done at %s\n", mm.nreg, nr, time.Since(t0).Round(time.Second))
			mi++
		}
	})
	fmt.Printf("# total %s\n", time.Since(t0).Round(time.Second))
}

// wallCheck is the decodability test the byte counts rest on.
// It replays each schedule against a plane pair that starts all-zero and is filled in only as the schedule reaches each bit, and asserts the context the *decoder* can build equals the one the encoder used.
// Any tap that reads a crack edge the decoder does not have yet shows up here as a mismatch, which is what makes the "base" row's known violation a measured fact rather than an argument about raster order.
func wallCheck(path string) {
	im := load(path)
	// A partition with real contours, not the exact partition: junctions and long walls are where a non-causal tap would actually be carrying information.
	q := quantize(path, 8)
	raw := make([]int32, len(q.Idx))
	for i, v := range q.Idx {
		raw[i] = int32(v)
	}
	lab := relabelComponents(raw, q.W, q.H)
	w, h := q.W, q.H
	_ = im
	V, Hz := crackPlanes(lab, w, h)
	fmt.Printf("# decoder-side causality check on a %d-region partition of %s (%dx%d)\n",
		components(lab, w, h), path, w, h)

	bad := false
	for _, v := range variants() {
		Vd := make([]byte, w*h)
		Hd := make([]byte, w*h)
		enc := func(pl, x, y int) int {
			if x < 0 || y < 0 || x >= w || y >= h {
				return 0
			}
			if pl == 0 {
				return int(V[y*w+x])
			}
			return int(Hz[y*w+x])
		}
		dec := func(pl, x, y int) int {
			if x < 0 || y < 0 || x >= w || y >= h {
				return 0
			}
			if pl == 0 {
				return int(Vd[y*w+x])
			}
			return int(Hd[y*w+x])
		}
		mismatch := 0
		step := func(pl, x, y int) {
			taps := v.tapsV
			if pl == 1 {
				taps = v.tapsH
			}
			ce, cd := 0, 0
			for i, t := range taps {
				ce |= enc(t.pl, x+t.dx, y+t.dy) << uint(i)
				cd |= dec(t.pl, x+t.dx, y+t.dy) << uint(i)
			}
			if ce != cd {
				mismatch++
			}
			if pl == 0 {
				Vd[y*w+x] = V[y*w+x]
			} else {
				Hd[y*w+x] = Hz[y*w+x]
			}
		}
		switch v.mode {
		case twoPassV:
			for y := 0; y < h; y++ {
				for x := 0; x < w-1; x++ {
					step(0, x, y)
				}
			}
			for y := 0; y < h-1; y++ {
				for x := 0; x < w; x++ {
					step(1, x, y)
				}
			}
		case twoPassH:
			for y := 0; y < h-1; y++ {
				for x := 0; x < w; x++ {
					step(1, x, y)
				}
			}
			for y := 0; y < h; y++ {
				for x := 0; x < w-1; x++ {
					step(0, x, y)
				}
			}
		case interVH:
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if x < w-1 {
						step(0, x, y)
					}
					if y < h-1 {
						step(1, x, y)
					}
				}
			}
		default:
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if y < h-1 {
						step(1, x, y)
					}
					if x < w-1 {
						step(0, x, y)
					}
				}
			}
		}
		status := "decodable"
		if mismatch > 0 {
			status = fmt.Sprintf("NOT DECODABLE — %d contexts differ from what a decoder can build", mismatch)
			if !v.nonCausal {
				bad = true
			}
		}
		fmt.Printf("%-10s %s\n", v.name, status)
	}
	if bad {
		fmt.Fprintln(os.Stderr, "fatal: a variant declared causal is not")
		os.Exit(1)
	}
}

// wallxExact prices the variants on the exact (lossless) partition, the operating point where the wall map is densest and the plane-to-plane dependence is strongest.
func wallxExact(path string) {
	im := load(path)
	lab, _, _ := exactPartition(im)
	cl := crackLen(lab, im.W, im.H)
	bb := caeBytes(lab, im.W, im.H)
	rows := wallxRow(lab, im.W, im.H)
	fmt.Printf("# %s %dx%d exact partition: %d crack edges, caeBytes %.0f B\n", path, im.W, im.H, cl, bb)
	if d := math.Abs(rows[0].bV + rows[0].bH - bb); d > 1e-6 {
		fmt.Fprintf(os.Stderr, "fatal: base variant disagrees with caeBytes by %.6f B\n", d)
		os.Exit(1)
	}
	for i, v := range variants() {
		t := rows[i].bV + rows[i].bH
		fmt.Printf("%-9s %10.0f B  (V %.0f / Hz %.0f)  %+.2f%% vs base\n",
			v.name, t, rows[i].bV, rows[i].bH, 100*(t-bb)/bb)
	}
}
