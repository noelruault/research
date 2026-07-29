package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// This file attacks one component and one only: the junction map of the contour coder.
//
// contourBytes reports two numbers, vertBits and turnBits, but vertBits is three different things added together:
// the context-coded sparse bitmap that says where the interior junctions are (contour.go:97-105),
// the four presence bits paid at every special vertex (contour.go:177),
// and the explicit start paid for each closed loop that touches no special vertex (contour.go:192).
// Nothing has ever separated them, so nobody knows what fraction of the contour bill the junction bitmap is.
// contourSplit measures that first; every optimisation below is only worth running if the answer is large.

// contourStats is contourBytes with vertBits decomposed and the population counted.
type contourStats struct {
	totalB, juncB, dirB, loopB, turnB float64
	steps, nTurns                     int
	nvTotal, nvInterior               int // lattice vertices; interior = the ones the junction bitmap codes
	nJunc, nFrame, nSpecial           int
	nLoops                            int
}

// contourSplit is a line-for-line copy of contourBytes with separate accumulators.
// It is asserted against contourBytes on every partition it is run on, so it cannot drift.
func contourSplit(lab []int32, w, h int) contourStats {
	var st contourStats
	g := buildCrackGraph(lab, w, h)
	nv := g.vw * g.vh
	st.nvTotal = nv
	st.nvInterior = (g.vw - 2) * (g.vh - 2)

	special := make([]bool, nv)
	for v := 0; v < nv; v++ {
		d := g.deg(v)
		if d != 2 && d != 0 {
			special[v] = true
		}
	}
	free := make([]bool, nv)
	for v := 0; v < nv; v++ {
		if g.onFrame(v) {
			free[v] = true
			special[v] = true
		}
	}

	occ := make([]byte, nv)
	for v := 0; v < nv; v++ {
		if special[v] && !free[v] {
			occ[v] = 1
			st.nJunc++
		}
		if free[v] {
			st.nFrame++
		}
		if special[v] {
			st.nSpecial++
		}
	}
	get := func(x, y int) int {
		if x < 0 || y < 0 || x >= g.vw || y >= g.vh {
			return 0
		}
		return int(occ[y*g.vw+x])
	}
	mv := make([]binModel, 1024)
	for y := 1; y < g.vh-1; y++ {
		for x := 1; x < g.vw-1; x++ {
			ctx := get(x-1, y) | get(x-2, y)<<1 | get(x, y-1)<<2 | get(x-1, y-1)<<3 |
				get(x+1, y-1)<<4 | get(x+2, y-1)<<5 | get(x, y-2)<<6 | get(x-1, y-2)<<7 |
				get(x+1, y-2)<<8 | get(x-3, y)<<9
			st.juncB += mv[ctx].cost(int(occ[y*g.vw+x]))
		}
	}

	mdir := make([]binModel, 4*5)
	used := [4][]bool{}
	for d := 0; d < 4; d++ {
		used[d] = make([]bool, nv)
	}
	mturn := make([][]uint32, 27)
	tturn := make([]uint32, 27)
	for i := range mturn {
		mturn[i] = make([]uint32, 3)
	}

	trace := func(v, d int) {
		ctxh := 0
		for {
			used[d][v] = true
			nx, ny := v%g.vw+dxs[d], v/g.vw+dys[d]
			nvv := ny*g.vw + nx
			used[(d+2)%4][nvv] = true
			st.steps++
			v = nvv
			if special[v] {
				return
			}
			nd := -1
			for k := 0; k < 4; k++ {
				if k != (d+2)%4 && g.e[k][v] {
					nd = k
					break
				}
			}
			if nd < 0 {
				return
			}
			sym := (nd - d + 4) % 4
			s := 0
			switch sym {
			case 0:
				s = 0
			case 1:
				s = 1
			case 3:
				s = 2
			}
			st.turnB += -math.Log2(float64(mturn[ctxh][s]+1) / float64(tturn[ctxh]+3))
			st.nTurns++
			mturn[ctxh][s]++
			tturn[ctxh]++
			ctxh = (ctxh*3 + s) % 27
			d = nd
		}
	}

	for v := 0; v < nv; v++ {
		if !special[v] {
			continue
		}
		seen := 0
		for d := 0; d < 4; d++ {
			nx, ny := v%g.vw+dxs[d], v/g.vw+dys[d]
			if nx < 0 || ny < 0 || nx >= g.vw || ny >= g.vh {
				continue
			}
			bit := 0
			if g.e[d][v] {
				bit = 1
			}
			st.dirB += mdir[d*5+min(seen, 4)].cost(bit)
			if bit == 1 {
				seen++
				if !used[d][v] {
					trace(v, d)
				}
			}
		}
	}
	for v := 0; v < nv; v++ {
		for d := 0; d < 4; d++ {
			if g.e[d][v] && !used[d][v] {
				st.nLoops++
				st.loopB += math.Log2(float64(nv)) + 2
				nt0 := st.nTurns
				trace2(g, v, d, mturn, tturn, &st.turnB, &st.steps)
				// trace2 does not report its own turn count; recover it from the model totals.
				_ = nt0
			}
		}
	}
	tt := 0
	for _, v := range tturn {
		tt += int(v)
	}
	st.nTurns = tt
	st.juncB /= 8
	st.dirB /= 8
	st.loopB /= 8
	st.turnB /= 8
	st.totalB = st.juncB + st.dirB + st.loopB + st.turnB
	return st
}

// ---------------------------------------------------------------- junction map, generalised

// jtap is one context bit of the junction bitmap: a sample of the occupancy lattice at (dx,dy).
type jtap struct{ dx, dy int }

func (t jtap) String() string { return fmt.Sprintf("J(%+d,%+d)", t.dx, t.dy) }

func jtapsStr(ts []jtap) string {
	s := make([]string, len(ts))
	for i, t := range ts {
		s[i] = t.String()
	}
	return strings.Join(s, " ")
}

// causal reports whether a tap is supplied by the raster scan the junction bitmap actually uses.
func (t jtap) causal() bool { return t.dy < 0 || (t.dy == 0 && t.dx < 0) }

// baseJTaps is exactly the ten context bits of contour.go:100-102, element i being bit i.
var baseJTaps = []jtap{
	{-1, 0}, {-2, 0}, {0, -1}, {-1, -1}, {1, -1},
	{2, -1}, {0, -2}, {-1, -2}, {1, -2}, {-3, 0},
}

// occLattice rebuilds the junction occupancy plane exactly as contourBytes does.
func occLattice(lab []int32, w, h int) (occ []byte, vw, vh int) {
	g := buildCrackGraph(lab, w, h)
	nv := g.vw * g.vh
	occ = make([]byte, nv)
	for v := 0; v < nv; v++ {
		if g.onFrame(v) {
			continue
		}
		if d := g.deg(v); d != 2 && d != 0 {
			occ[v] = 1
		}
	}
	return occ, g.vw, g.vh
}

// juncPlane prices the junction bitmap with an arbitrary ordered tap list, same adaptive binary model, same scan.
// With baseJTaps it must equal contourSplit's juncB to the bit.
// It also returns the ideal static conditional entropy, which is what the same context would cost with free model tables.
func juncPlane(occ []byte, vw, vh int, ts []jtap) (adaptive, static float64) {
	get := func(x, y int) int {
		if x < 0 || y < 0 || x >= vw || y >= vh {
			return 0
		}
		return int(occ[y*vw+x])
	}
	n := 1 << uint(len(ts))
	m := make([]binModel, n)
	cnt := make([][2]uint32, n)
	bits := 0.0
	for y := 1; y < vh-1; y++ {
		for x := 1; x < vw-1; x++ {
			ctx := 0
			for i, t := range ts {
				ctx |= get(x+t.dx, y+t.dy) << uint(i)
			}
			b := int(occ[y*vw+x])
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

// jPool is the causal candidate neighbourhood for widening, out to radius 5 horizontally and 4 vertically.
func jPool() []jtap {
	var o []jtap
	for dx := -6; dx <= -1; dx++ {
		o = append(o, jtap{dx, 0})
	}
	for dy := -1; dy >= -4; dy-- {
		lim := 5 + dy // -1 -> 4, -2 -> 3, -3 -> 2, -4 -> 1
		if lim < 1 {
			lim = 1
		}
		for dx := -lim; dx <= lim; dx++ {
			o = append(o, jtap{dx, dy})
		}
	}
	return o
}

func hasJTap(ts []jtap, t jtap) bool {
	for _, u := range ts {
		if u == t {
			return true
		}
	}
	return false
}

func greedyJTaps(occ []byte, vw, vh int, base, pool []jtap, want int) []jtap {
	ts := append([]jtap{}, base...)
	cur, _ := juncPlane(occ, vw, vh, ts)
	fmt.Printf("  start %d bits: %.1f B\n", len(ts), cur)
	for len(ts) < want {
		bestB := math.Inf(1)
		var bestT jtap
		for _, c := range pool {
			if hasJTap(ts, c) {
				continue
			}
			b, _ := juncPlane(occ, vw, vh, append(ts, c))
			if b < bestB {
				bestB, bestT = b, c
			}
		}
		if math.IsInf(bestB, 1) {
			break
		}
		ts = append(ts, bestT)
		fmt.Printf("  +%-10s -> %2d bits: %10.1f B  (%+.2f%%)\n", bestT, len(ts), bestB, 100*(bestB-cur)/cur)
		cur = bestB
	}
	return ts
}

// jDistanceOrder is the zero-fitting control: extend the baseline by whichever causal tap is merely nearest.
func jDistanceOrder(base, pool []jtap, want int) []jtap {
	ts := append([]jtap{}, base...)
	rest := []jtap{}
	for _, c := range pool {
		if !hasJTap(ts, c) {
			rest = append(rest, c)
		}
	}
	sort.SliceStable(rest, func(i, j int) bool {
		di := rest[i].dx*rest[i].dx + rest[i].dy*rest[i].dy
		dj := rest[j].dx*rest[j].dx + rest[j].dy*rest[j].dy
		return di < dj
	})
	for _, c := range rest {
		if len(ts) >= want {
			break
		}
		ts = append(ts, c)
	}
	return ts
}

// ---------------------------------------------------------------- commands

// csplitCmd is deliverable one: reproduce the published contour total, then break it into its four parts.
func csplitCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: labx csplit <render.png> [expectRegions] [expectContourB]")
		os.Exit(2)
	}
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	if len(args) > 1 {
		want, _ := strconv.Atoi(args[1])
		if d := math.Abs(float64(n-want)) / float64(want); d > 0.001 {
			fmt.Printf("REGION MISMATCH: recovered %d, published %d (%.3f%%) -- stopping\n", n, want, 100*d)
			os.Exit(1)
		} else if n != want {
			fmt.Fprintf(os.Stderr, "note: recovered %d regions vs published %d (%.4f%%)\n", n, want, 100*d)
		}
	}
	ref, refVert, refTurn, refSteps := contourBytes(lab, w, h)
	st := contourSplit(lab, w, h)
	if math.Abs(ref-st.totalB) > 1e-6 || math.Abs(refVert-(st.juncB+st.dirB+st.loopB)) > 1e-6 ||
		math.Abs(refTurn-st.turnB) > 1e-6 || refSteps != st.steps {
		fmt.Printf("SPLIT MISMATCH: contourBytes %.6f/%.6f/%.6f/%d vs split %.6f/%.6f/%.6f/%d -- stopping\n",
			ref, refVert, refTurn, refSteps, st.totalB, st.juncB+st.dirB+st.loopB, st.turnB, st.steps)
		os.Exit(1)
	}
	cae := caeBytes(lab, w, h)
	pub := math.Min(cae, ref)
	if len(args) > 2 {
		want, _ := strconv.ParseFloat(args[2], 64)
		if math.Abs(ref-want) > 1 {
			fmt.Printf("CONTOUR MISMATCH: got %.2f B, published %.0f B -- stopping\n", ref, want)
			os.Exit(1)
		}
	}
	fmt.Printf("%-9d %9d %10.0f %10.0f %10.0f %8.2f%% %10.0f %8.2f%% %8.1f %7.2f%% %8.1f %7.2f%% %8d %8d %8d %8d %7.3f %10.0f\n",
		n, crackLen(lab, w, h), ref, cae, st.juncB, 100*st.juncB/ref, st.turnB, 100*st.turnB/ref,
		st.dirB, 100*st.dirB/ref, st.loopB, 100*st.loopB/ref, st.steps, st.nTurns, st.nJunc, st.nLoops,
		st.turnB*8/float64(st.nTurns), pub)
}

func csplitHeader() {
	fmt.Printf("%-9s %9s %10s %10s %10s %9s %10s %9s %8s %8s %8s %8s %8s %8s %8s %8s %7s %10s\n",
		"regions", "crack", "contourB", "caeB", "juncB", "junc%", "turnB", "turn%", "dirB", "dir%",
		"loopB", "loop%", "steps", "turns", "njunc", "nloops", "b/turn", "pubWall")
}

// cwidthCmd prices the junction bitmap at a range of context widths beside the baseline, on identical labels.
// dist is the zero-fitting control; sel is the frozen greedy template from selectedJ.go.
func cwidthCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: labx cwidth <render.png> [expectRegions]")
		os.Exit(2)
	}
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	if len(args) > 1 {
		want, _ := strconv.Atoi(args[1])
		if d := math.Abs(float64(n-want)) / float64(want); d > 0.001 {
			fmt.Printf("REGION MISMATCH: recovered %d, published %d (%.3f%%) -- stopping\n", n, want, 100*d)
			os.Exit(1)
		}
	}
	occ, vw, vh := occLattice(lab, w, h)
	st := contourSplit(lab, w, h)
	base, baseS := juncPlane(occ, vw, vh, baseJTaps)
	if math.Abs(base-st.juncB) > 1e-6 {
		fmt.Printf("PRICER MISMATCH: juncPlane %.6f vs contourSplit %.6f -- stopping\n", base, st.juncB)
		os.Exit(1)
	}
	cae := caeBytes(lab, w, h)
	pubWall := math.Min(cae, st.totalB)
	// Two reference floors, so "how much is left in this component" is answered rather than guessed.
	// order0 is what the map costs with no context at all; comb is log2 C(N,k), the cost of naming k positions
	// out of N with no structure assumed at all -- an enumerative coder's exact bill.
	// Neither is achievable by a context coder, but the adaptive number sitting between them bounds the prize.
	nInterior := float64((vw - 2) * (vh - 2))
	nOnes := 0.0
	for y := 1; y < vh-1; y++ {
		for x := 1; x < vw-1; x++ {
			nOnes += float64(occ[y*vw+x])
		}
	}
	p := nOnes / nInterior
	order0 := nInterior * (-p*math.Log2(p) - (1-p)*math.Log2(1-p)) / 8
	lg := func(v float64) float64 { r, _ := math.Lgamma(v + 1); return r / math.Ln2 }
	comb := (lg(nInterior) - lg(nOnes) - lg(nInterior-nOnes)) / 8
	fmt.Printf("%-9d %9.0f %9.0f %9.0f %9.0f %9.0f", n, st.totalB, base, baseS, order0, comb)
	bestSel, bestSelK := base, 10
	selS20 := baseS
	for _, k := range []int{12, 14, 16, 18, 20} {
		if k > len(selJTaps) {
			break
		}
		a, s := juncPlane(occ, vw, vh, selJTaps[:k])
		fmt.Printf(" %9.0f", a)
		if a < bestSel {
			bestSel, bestSelK = a, k
		}
		if k == 20 {
			selS20 = s
		}
	}
	d16, _ := juncPlane(occ, vw, vh, jDistanceOrder(baseJTaps, jPool(), 16))
	newContour := st.totalB - base + bestSel
	newWall := math.Min(cae, newContour)
	// The prize ceiling: what the whole junction map would be worth if it cost its widest static conditional entropy, i.e. if the model tables were free. No adaptive coder reaches it; nothing can beat it at this width.
	ceilPct := 100 * (base - selS20) / st.totalB
	fmt.Printf(" %9.0f %9.0f %2d %8.2f%% %8.2f%% %8.2f%% %8.2f%% %9.0f %8.3f%% %8.3f%%\n",
		d16, bestSel, bestSelK,
		100*(bestSel-base)/base, 100*(d16-base)/base,
		100*(newContour-st.totalB)/st.totalB,
		100*(newWall-pubWall)/pubWall,
		selS20, 100*(bestSel-base)/st.totalB, ceilPct)
}

func cwidthHeader() {
	fmt.Printf("%-9s %9s %9s %9s %9s %9s %9s %9s %9s %9s %9s %9s %9s %2s %9s %9s %9s %9s %9s %9s %9s\n",
		"regions", "contourB", "junc10", "staticH10", "order0", "combLB", "sel12", "sel14", "sel16", "sel18", "sel20",
		"dist16", "best", "k", "dJunc", "dDist16", "dContour", "dWall", "staticH20", "dCt_pp", "ceil_pp")
}

// cselCmd runs greedy forward selection on the junction bitmap of one partition and prints the frozen order.
func cselCmd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: labx csel <render.png> <maxbits>")
		os.Exit(2)
	}
	k, _ := strconv.Atoi(args[1])
	im, lab, n := partFromRender(args[0])
	occ, vw, vh := occLattice(lab, im.W, im.H)
	ones := 0
	for _, v := range occ {
		if v == 1 {
			ones++
		}
	}
	fmt.Printf("# %s  %dx%d  regions %d  junctions %d  lattice %dx%d\n", args[0], im.W, im.H, n, ones, vw, vh)
	ts := greedyJTaps(occ, vw, vh, baseJTaps, jPool(), k)
	fmt.Printf("\nJ taps: %s\n", jtapsStr(ts))
	fmt.Printf("\nGO:\nvar selJTaps = []jtap{")
	for _, t := range ts {
		fmt.Printf("{%d,%d},", t.dx, t.dy)
	}
	fmt.Println("}")
}

// ccausalCmd is the decoder-side replay demanded by invariant "causality".
//
// Three separate checks:
//  1. every tap of the junction bitmap template is supplied by its own raster scan (static predicate);
//  2. a replay that rebuilds each context from a lattice filled in only as the scan reaches each bit
//     reproduces every context the encoder used (empirical);
//  3. the rest of the contour coder -- the presence bits and the chain traces -- is walked in decoder order
//     and every decision it makes is checked against information the decoder already holds.
func ccausalCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: labx ccausal <render.png>")
		os.Exit(2)
	}
	im, lab, n := partFromRender(args[0])
	w, h := im.W, im.H
	fmt.Printf("# %s  %dx%d  regions %d\n", args[0], w, h, n)

	for _, name := range []string{"base10", "sel"} {
		ts := baseJTaps
		if name == "sel" {
			ts = selJTaps
		}
		bad := []jtap{}
		for _, t := range ts {
			if !t.causal() {
				bad = append(bad, t)
			}
		}
		fmt.Printf("static predicate  %-8s %d taps, %d non-causal %v\n", name, len(ts), len(bad), bad)
	}

	occ, vw, vh := occLattice(lab, w, h)
	for _, tc := range []struct {
		name string
		ts   []jtap
	}{{"base10", baseJTaps}, {"sel", selJTaps}} {
		known := make([]byte, len(occ)) // filled only as the scan reaches each bit
		mism := 0
		get := func(a []byte, x, y int) int {
			if x < 0 || y < 0 || x >= vw || y >= vh {
				return 0
			}
			return int(a[y*vw+x])
		}
		for y := 1; y < vh-1; y++ {
			for x := 1; x < vw-1; x++ {
				enc, dec := 0, 0
				for i, t := range tc.ts {
					enc |= get(occ, x+t.dx, y+t.dy) << uint(i)
					dec |= get(known, x+t.dx, y+t.dy) << uint(i)
				}
				if enc != dec {
					mism++
				}
				known[y*vw+x] = occ[y*vw+x]
			}
		}
		fmt.Printf("replay junction   %-8s %d mismatching contexts (must be 0)\n", tc.name, mism)
	}

	// The rest of the coder, walked the way a decoder would.
	g := buildCrackGraph(lab, w, h)
	nv := g.vw * g.vh
	special := make([]bool, nv)
	for v := 0; v < nv; v++ {
		if d := g.deg(v); d != 2 && d != 0 {
			special[v] = true
		}
		if g.onFrame(v) {
			special[v] = true
		}
	}
	// A decoder holds: the special set (frame by definition, interior from the bitmap it just decoded),
	// the presence bits it has decoded so far, and the turns of the chain it is currently tracing.
	used := [4][]bool{}
	for d := 0; d < 4; d++ {
		used[d] = make([]bool, nv)
	}
	badTerm, badTurn, degTrace := 0, 0, 0
	var trace func(v, d int)
	trace = func(v, d int) {
		for {
			used[d][v] = true
			nx, ny := v%g.vw+dxs[d], v/g.vw+dys[d]
			nvv := ny*g.vw + nx
			used[(d+2)%4][nvv] = true
			v = nvv
			if special[v] {
				return // decoder knows special[], so it stops here too
			}
			// The decoder must be able to resolve the continuation from the turn symbol alone, which requires this vertex to have exactly one continuation, i.e. degree exactly 2.
			if g.deg(v) != 2 {
				degTrace++
			}
			nd := -1
			cont := 0
			for k := 0; k < 4; k++ {
				if k != (d+2)%4 && g.e[k][v] {
					if nd < 0 {
						nd = k
					}
					cont++
				}
			}
			if nd < 0 {
				badTerm++ // a chain that ends at a non-special vertex: the decoder has no terminator
				return
			}
			if cont != 1 {
				badTurn++ // ambiguous continuation
			}
			d = nd
		}
	}
	for v := 0; v < nv; v++ {
		if !special[v] {
			continue
		}
		for d := 0; d < 4; d++ {
			nx, ny := v%g.vw+dxs[d], v/g.vw+dys[d]
			if nx < 0 || ny < 0 || nx >= g.vw || ny >= g.vh {
				continue
			}
			if g.e[d][v] && !used[d][v] {
				trace(v, d)
			}
		}
	}
	// Anything left is a closed loop; the encoder pays log2(nv)+2 for each but sends no count and no
	// end-of-loops flag, which is a real (small) under-charge rather than a causality break.
	loops, loopEdges := 0, 0
	for v := 0; v < nv; v++ {
		for d := 0; d < 4; d++ {
			if g.e[d][v] && !used[d][v] {
				loopEdges++
			}
		}
	}
	seenLoop := make([]bool, nv)
	for v := 0; v < nv; v++ {
		for d := 0; d < 4; d++ {
			if g.e[d][v] && !used[d][v] && !seenLoop[v] {
				loops++
				vv, dd := v, d
				for {
					used[dd][vv] = true
					seenLoop[vv] = true
					nx, ny := vv%g.vw+dxs[dd], vv/g.vw+dys[dd]
					nvv := ny*g.vw + nx
					used[(dd+2)%4][nvv] = true
					vv = nvv
					if vv == v {
						break
					}
					nd := -1
					for k := 0; k < 4; k++ {
						if k != (dd+2)%4 && g.e[k][vv] {
							nd = k
							break
						}
					}
					if nd < 0 {
						break
					}
					dd = nd
				}
			}
		}
	}
	// Degree-1 vertices are walls that run off the edge of the image, so they must all sit on the frame, which the decoder knows for free. An interior one would be a wall the decoder could not terminate.
	deg1, deg1Interior := 0, 0
	for v := 0; v < nv; v++ {
		if g.deg(v) == 1 {
			deg1++
			if !g.onFrame(v) {
				deg1Interior++
			}
		}
	}
	fmt.Printf("replay chains     traces terminate at a non-special vertex: %d (must be 0)\n", badTerm)
	fmt.Printf("replay chains     ambiguous continuations: %d (must be 0); degree!=2 mid-chain: %d (must be 0)\n", badTurn, degTrace)
	fmt.Printf("replay chains     degree-1 vertices: %d, of which interior: %d (must be 0; frame ones are free to the decoder)\n", deg1, deg1Interior)
	fmt.Printf("loop channel      %d closed loops, %d edges; encoder pays log2(nv)+2 each and sends no terminator\n", loops, loopEdges)
}
