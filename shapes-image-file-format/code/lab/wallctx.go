package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// This file tests one lever and one only: is the 10-bit CAE context of caeBytes starved at 3840x2160?
//
// The 10-bit template was chosen when the eval was 512x288 = 147,456 px, i.e. ~144 samples per context bin.
// At 4K each crack plane has 8,294,400 samples, ~8,100 per bin, so a wider context could be affordable that was not before.
// Nothing here changes caeBytes: the variants are strict supersets of its ten taps, so any delta is the widening and nothing else.
//
// Partitions come from the published renders (flat region colours) rather than from a fresh multi-hour merge.
// wallctx asserts the recovered region count and the recovered caeBytes against the published numbers before it reports anything, so a mis-recovered partition cannot be mistaken for a result.

// wtap is one context bit: a sample of plane p at offset (dx,dy) from the crack edge being coded.
type wtap struct {
	p      int // 0 = V plane (crack between (x,y) and (x+1,y)), 1 = Hz plane (crack between (x,y) and (x,y+1))
	dx, dy int
}

func (t wtap) String() string {
	n := "V"
	if t.p == 1 {
		n = "H"
	}
	return fmt.Sprintf("%s(%+d,%+d)", n, t.dx, t.dy)
}

func tapsStr(ts []wtap) string {
	s := make([]string, len(ts))
	for i, t := range ts {
		s[i] = t.String()
	}
	return strings.Join(s, " ")
}

// baseVTaps and baseHTaps are exactly the ten context bits caeBytes uses, element i being bit i.
// Kept as the prefix of every variant so the comparison is like-for-like.
var baseVTaps = []wtap{
	{0, -1, 0}, {0, -2, 0}, {0, 0, -1}, {0, -1, -1}, {0, 1, -1},
	{0, 2, -1}, {0, 0, -2}, {0, -1, -2}, {0, 1, -2}, {0, -2, -1},
}

// Note bit 4, H(+1,0): that sample is to the right on the same row, so the Hz plane is coded in raster order and it has not been coded yet.
// The baseline is therefore very slightly non-causal and a real decoder could not use it. It is left in place because removing it would change the baseline this study published; caeCausalDelta measures what it is worth.
var baseHTaps = []wtap{
	{1, -1, 0}, {1, 0, -1}, {1, -1, -1}, {1, 1, -1}, {1, 1, 0},
	{0, 0, 0}, {0, -1, 0}, {0, 0, 1}, {0, -1, 1}, {1, -2, 0},
}

// crackPlanes lives in crossplane.go; both files derive the planes identically.

// caePlane prices one crack plane with an arbitrary ordered wtap list, using the same adaptive binary model as caeBytes.
// plane 0 scans x in [0,w-1) as caeBytes does; plane 1 scans y in [0,h-1).
// It also returns the ideal static conditional entropy of the same context, which is what the coder would pay if the model tables were free — the gap between the two is the learning cost that widening has to earn back.
func caePlane(V, Hz []byte, w, h, plane int, ts []wtap) (adaptive, static float64) {
	pl := [2][]byte{V, Hz}
	get := func(p, x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		return int(pl[p][y*w+x])
	}
	n := 1 << uint(len(ts))
	m := make([]binModel, n)
	cnt := make([][2]uint32, n)
	bits := 0.0
	xhi, yhi := w-1, h
	if plane == 1 {
		xhi, yhi = w, h-1
	}
	self := pl[plane]
	for y := 0; y < yhi; y++ {
		for x := 0; x < xhi; x++ {
			ctx := 0
			for i, t := range ts {
				ctx |= get(t.p, x+t.dx, y+t.dy) << uint(i)
			}
			b := int(self[y*w+x])
			bits += m[ctx].cost(b)
			cnt[ctx][b]++
		}
	}
	for _, c := range cnt {
		t := float64(c[0] + c[1])
		if t == 0 {
			continue
		}
		for _, v := range c {
			if v > 0 {
				static += float64(v) * math.Log2(t/float64(v))
			}
		}
	}
	return bits / 8, static / 8
}

// caeTapsBytes is caeBytes with the template swapped out. With baseVTaps/baseHTaps it must agree with caeBytes to the bit.
func caeTapsBytes(V, Hz []byte, w, h int, vt, ht []wtap) (vB, hB, vS, hS float64) {
	vB, vS = caePlane(V, Hz, w, h, 0, vt)
	hB, hS = caePlane(V, Hz, w, h, 1, ht)
	return
}

// Candidate pools for widening. V is coded first and so may only condition on V; Hz is coded second and may condition on the whole of V.
// The pools are the causal neighbourhood out to radius 3-4, plus (for Hz) the V samples that straddle the crack.
func vPool() []wtap {
	var o []wtap
	for _, dx := range []int{-1, -2, -3, -4, -5} {
		o = append(o, wtap{0, dx, 0})
	}
	for _, dx := range []int{-4, -3, -2, -1, 0, 1, 2, 3, 4} {
		o = append(o, wtap{0, dx, -1})
	}
	for _, dx := range []int{-3, -2, -1, 0, 1, 2, 3} {
		o = append(o, wtap{0, dx, -2})
	}
	for _, dx := range []int{-2, -1, 0, 1, 2} {
		o = append(o, wtap{0, dx, -3})
	}
	for _, dx := range []int{-1, 0, 1} {
		o = append(o, wtap{0, dx, -4})
	}
	return o
}

func hPool() []wtap {
	var o []wtap
	for _, dx := range []int{-1, -2, -3, -4, 1, 2} { // +1,+2 on the current row are the baseline's non-causal reach
		o = append(o, wtap{1, dx, 0})
	}
	for _, dx := range []int{-3, -2, -1, 0, 1, 2, 3} {
		o = append(o, wtap{1, dx, -1})
	}
	for _, dx := range []int{-2, -1, 0, 1, 2} {
		o = append(o, wtap{1, dx, -2})
	}
	for _, dx := range []int{-1, 0, 1} {
		o = append(o, wtap{1, dx, -3})
	}
	for dy := -2; dy <= 2; dy++ { // the V plane is fully available to Hz
		for dx := -2; dx <= 2; dx++ {
			o = append(o, wtap{0, dx, dy})
		}
	}
	return o
}

func hasTap(ts []wtap, t wtap) bool {
	for _, u := range ts {
		if u == t {
			return true
		}
	}
	return false
}

// greedyTaps grows a template one bit at a time, each step keeping the candidate that lowers the adaptive code length most.
// Selection is done on the image being measured, which flatters the result; the transfer of a frozen order to the other operating points is reported separately for exactly that reason.
func greedyTaps(V, Hz []byte, w, h, plane int, base []wtap, pool []wtap, want int) []wtap {
	ts := append([]wtap{}, base...)
	cur, _ := caePlane(V, Hz, w, h, plane, ts)
	fmt.Printf("  plane %d start %d bits: %.0f B\n", plane, len(ts), cur)
	for len(ts) < want {
		bestB := math.Inf(1)
		var bestT wtap
		for _, c := range pool {
			if hasTap(ts, c) {
				continue
			}
			b, _ := caePlane(V, Hz, w, h, plane, append(ts, c))
			if b < bestB {
				bestB, bestT = b, c
			}
		}
		if math.IsInf(bestB, 1) {
			break
		}
		ts = append(ts, bestT)
		fmt.Printf("  plane %d +%-10s -> %d bits: %10.0f B  (%+.2f%%)\n",
			plane, bestT, len(ts), bestB, 100*(bestB-cur)/cur)
		cur = bestB
	}
	return ts
}

// distanceOrder is the non-cherry-picked control: extend the baseline by whichever causal wtap is nearest, ties broken by scan order.
func distanceOrder(base, pool []wtap, want int) []wtap {
	ts := append([]wtap{}, base...)
	rest := []wtap{}
	for _, c := range pool {
		if !hasTap(ts, c) {
			rest = append(rest, c)
		}
	}
	sort.SliceStable(rest, func(i, j int) bool {
		di := rest[i].dx*rest[i].dx + rest[i].dy*rest[i].dy
		dj := rest[j].dx*rest[j].dx + rest[j].dy*rest[j].dy
		if di != dj {
			return di < dj
		}
		return rest[i].p < rest[j].p
	})
	for _, c := range rest {
		if len(ts) >= want {
			break
		}
		ts = append(ts, c)
	}
	return ts
}

// partFromRender recovers a partition from one of the published flat renders: every 4-connected run of one exact colour is a region.
// The render is the decoder's own output, so this is the same label field the published numbers were priced on, provided the region count comes back identical — which the caller checks.
func partFromRender(path string) (*Img, []int32, int) {
	im := load(path)
	lab, cols, _ := exactPartition(im)
	return im, lab, len(cols)
}

func wallctxCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lab wallctx <render.png> [expectRegions] [expectCAEBytes]")
		os.Exit(2)
	}
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	fmt.Printf("# %s  %dx%d  regions %d  crack %d\n", args[0], w, h, n, crackLen(lab, w, h))
	if len(args) > 1 {
		want, _ := strconv.Atoi(args[1])
		if want != n {
			fmt.Printf("REGION MISMATCH: recovered %d, published %d -- partition not reproduced, stopping\n", n, want)
			os.Exit(1)
		}
	}
	V, Hz := crackPlanes(lab, w, h)

	// Reproduction gate: the generalised pricer on the baseline template must equal caeBytes on the same labels.
	ref := caeBytes(lab, w, h)
	vB, hB, vS, hS := caeTapsBytes(V, Hz, w, h, baseVTaps, baseHTaps)
	fmt.Printf("caeBytes            %12.2f B\n", ref)
	fmt.Printf("caeTaps(10,10)      %12.2f B   (V %.0f + Hz %.0f)\n", vB+hB, vB, hB)
	if math.Abs(ref-(vB+hB)) > 1e-6 {
		fmt.Printf("PRICER MISMATCH: %.6f B apart -- stopping\n", math.Abs(ref-(vB+hB)))
		os.Exit(1)
	}
	ct, _, _, _ := contourBytes(lab, w, h)
	fmt.Printf("contourBytes        %12.2f B\n", ct)
	fmt.Printf("min(CAE,contour)    %12.2f B\n", math.Min(ref, ct))
	if len(args) > 2 {
		want, _ := strconv.ParseFloat(args[2], 64)
		fmt.Printf("published walls     %12.2f B   (delta %.2f B)\n", want, math.Min(ref, ct)-want)
	}
	fmt.Printf("static H(X|ctx10)   %12.2f B   (V %.0f + Hz %.0f)  learning cost %.0f B\n",
		vS+hS, vS, hS, (vB+hB)-(vS+hS))

	// Runnable check for the claim that makes this whole result what it is.
	// Around the crack-lattice vertex at (x+0.5,y+0.5) the four pixels (x,y),(x+1,y),(x+1,y+1),(x,y+1) form a cycle whose four "label changed" flags are exactly V(x,y), Hz(x+1,y), V(x,y+1), Hz(x,y).
	// When at most two distinct labels meet there, the number of changes round the cycle is even, so Hz(x,y) = V(x,y) XOR V(x,y+1) XOR Hz(x+1,y) -- and those three samples are already bits 5, 7 and 4 of the baseline Hz context.
	// The relation can only fail where three or four labels meet, i.e. at a junction (a junction may still satisfy it by coincidence, as a,b,a,c does), so the check asserts that no violation occurs at a two-label vertex.
	// If badViol is ever nonzero, the explanation for Hz costing under 1% of the bill is wrong.
	viol, junc, badViol := 0, 0, 0
	for y := 0; y < h-1; y++ {
		for x := 0; x < w-1; x++ {
			bad := int(Hz[y*w+x])^int(V[y*w+x])^int(V[(y+1)*w+x])^int(Hz[y*w+x+1]) != 0
			if bad {
				viol++
			}
			a, b, c, d := lab[y*w+x], lab[y*w+x+1], lab[(y+1)*w+x+1], lab[(y+1)*w+x]
			nd := 1
			if b != a {
				nd++
			}
			if c != a && c != b {
				nd++
			}
			if d != a && d != b && d != c {
				nd++
			}
			if nd >= 3 {
				junc++
			} else if bad {
				badViol++
			}
		}
	}
	fmt.Printf("parity check        %12d violations of Hz = V(x,y)^V(x,y+1)^Hz(x+1,y); %d junction vertices (3+ labels); %d violations at a two-label vertex (must be 0)\n", viol, junc, badViol)

	// The two planes are priced independently, because they are independent: V is coded first and conditions on nothing else, and Hz's cost does not enter V's context.
	// Reporting them jointly hid the fact that Hz is already at its floor.
	base := vB + hB
	fmt.Printf("\n# V plane alone (%.1f%% of the CAE bill)\n", 100*vB/base)
	fmt.Printf("%-8s %12s %12s %10s\n", "bits", "adaptive_B", "staticH_B", "vs 10b")
	fmt.Printf("%-8d %12.0f %12.0f %9.2f%%\n", 10, vB, vS, 0.0)
	bestV, bestVk := vB, 10
	for _, k := range []int{11, 12, 13, 14, 15, 16, 17, 18, 20} {
		vt := distanceOrder(baseVTaps, vPool(), k)
		if len(vt) < k {
			break
		}
		a, s := caePlane(V, Hz, w, h, 0, vt)
		fmt.Printf("%-8d %12.0f %12.0f %9.2f%%\n", k, a, s, 100*(a-vB)/vB)
		if a < bestV {
			bestV, bestVk = a, k
		}
	}
	fmt.Printf("\n# Hz plane alone (%.1f%% of the CAE bill)\n", 100*hB/base)
	fmt.Printf("%-8s %12s %12s %10s\n", "bits", "adaptive_B", "staticH_B", "vs 10b")
	fmt.Printf("%-8d %12.0f %12.0f %9.2f%%\n", 10, hB, hS, 0.0)
	bestH, bestHk := hB, 10
	for _, k := range []int{11, 12, 14, 16} {
		ht := distanceOrder(baseHTaps, hPool(), k)
		if len(ht) < k {
			break
		}
		b, s := caePlane(V, Hz, w, h, 1, ht)
		fmt.Printf("%-8d %12.0f %12.0f %9.2f%%\n", k, b, s, 100*(b-hB)/hB)
		if b < bestH {
			bestH, bestHk = b, k
		}
	}
	fmt.Printf("\nbest per-plane: V %d bits + Hz %d bits = %.0f B vs base %.0f B (%+.2f%%)\n",
		bestVk, bestHk, bestV+bestH, base, 100*(bestV+bestH-base)/base)
}

// wallselCmd derives a wtap order by greedy forward selection on this image, then prints it so it can be frozen and applied elsewhere.
func wallselCmd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: lab wallsel <render.png> <maxbits>")
		os.Exit(2)
	}
	k, _ := strconv.Atoi(args[1])
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	fmt.Printf("# %s  %dx%d  regions %d\n", args[0], w, h, n)
	V, Hz := crackPlanes(lab, w, h)
	vt := greedyTaps(V, Hz, w, h, 0, baseVTaps, vPool(), k)
	// The Hz plane is skipped unless asked for: it is under 1% of the bill and every extra bit makes it worse, because H(x,y) = V(x,y) XOR V(x,y+1) XOR H(x+1,y) is already in the baseline's ten taps.
	ht := baseHTaps
	if len(args) > 2 && args[2] == "hz" {
		ht = greedyTaps(V, Hz, w, h, 1, baseHTaps, hPool(), k)
	}
	fmt.Printf("\nV taps: %s\n", tapsStr(vt))
	fmt.Printf("H taps: %s\n", tapsStr(ht))
	fmt.Printf("\nGO:\nvar selVTaps = []wtap{")
	for _, t := range vt {
		fmt.Printf("{%d,%d,%d},", t.p, t.dx, t.dy)
	}
	fmt.Printf("}\nvar selHTaps = []wtap{")
	for _, t := range ht {
		fmt.Printf("{%d,%d,%d},", t.p, t.dx, t.dy)
	}
	fmt.Println("}")
}

// wallevalCmd prices a partition at every width using a frozen wtap order supplied on the command line, so a template selected at one operating point can be tested at the others.
func wallevalCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lab walleval <render.png> [expectRegions]")
		os.Exit(2)
	}
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	// The partition is recovered from the published render, so two adjacent regions whose rounded mean colours collide come back as one.
	// Below ~34k regions recovery is exact; above it a handful of regions merge (worst case 3,378,316 of 3,380,956, 0.08%).
	// That drift is reported rather than hidden, and it cannot contaminate the result being measured: baseline and variant are priced on the identical recovered labels, so the delta is unaffected.
	if len(args) > 1 {
		want, _ := strconv.Atoi(args[1])
		if d := math.Abs(float64(n-want)) / float64(want); d > 0.001 {
			fmt.Printf("REGION MISMATCH: recovered %d, published %d (%.3f%%) -- stopping\n", n, want, 100*d)
			os.Exit(1)
		} else if n != want {
			fmt.Fprintf(os.Stderr, "note: recovered %d regions vs published %d (%.4f%%)\n", n, want, 100*d)
		}
	}
	V, Hz := crackPlanes(lab, w, h)
	vB, hB, _, _ := caeTapsBytes(V, Hz, w, h, baseVTaps, baseHTaps)
	cae := vB + hB
	ct, _, _, _ := contourBytes(lab, w, h)
	// The published wall bill is min(CAE, contour), so that is what a CAE improvement has to be measured against: at coarse partitions the contour coder already wins and a cheaper CAE changes nothing.
	pubWall := math.Min(cae, ct)
	fmt.Printf("%-9d %-9d %10.0f %10.0f %10.0f", n, crackLen(lab, w, h), vB, hB, cae)
	for _, k := range []int{12, 14, 16, 18} {
		a, _ := caePlane(V, Hz, w, h, 0, selVTaps[:k])
		fmt.Printf(" %10.0f", a+hB)
	}
	a16, _ := caePlane(V, Hz, w, h, 0, selVTaps[:16])
	// dist16 is the control: the same 16 bits, but chosen by nothing more than "take the nearest unused causal wtap".
	// It carries no fitting to this image at all, so it is the honest floor of the effect; sel16 is what greedy selection adds on top.
	d16, _ := caePlane(V, Hz, w, h, 0, distanceOrder(baseVTaps, vPool(), 16))
	newWall := math.Min(a16+hB, ct)
	fmt.Printf(" %10.0f %10.0f %10.0f %10.0f %8.2f%% %8.2f%% %8.2f%%\n",
		d16+hB, ct, pubWall, newWall,
		100*(a16+hB-cae)/cae, 100*(d16+hB-cae)/cae, 100*(newWall-pubWall)/pubWall)
}

// caeCausalDelta reports what the baseline's one non-causal wtap, H(+1,0), is actually worth, so the reader knows the size of the concession.
func caeCausalCmd(args []string) {
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	V, Hz := crackPlanes(lab, w, h)
	_, hB, _, _ := caeTapsBytes(V, Hz, w, h, baseVTaps, baseHTaps)
	fix := append([]wtap{}, baseHTaps...)
	fix[4] = wtap{1, 0, -2} // replace the non-causal H(+1,0) with a causal wtap of the same count
	_, hC, _, _ := caeTapsBytes(V, Hz, w, h, baseVTaps, fix)
	fmt.Printf("%s regions %d: Hz with non-causal H(+1,0) %.0f B, causal substitute H(0,-2) %.0f B (%+.2f%%)\n",
		args[0], n, hB, hC, 100*(hC-hB)/hB)
}
