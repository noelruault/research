package main

// Turn coding for the contour wall coder, measured and re-scheduled.
//
// contourBytes splits its bill into vertBits (the junction bitmap plus the per-vertex direction bits) and turnBits (the chain turns).
// This file touches turnBits only; vertBits is reproduced byte-for-byte from the published coder so every variant is priced on the identical junction map.
//
// The whole file is built around one decoder-state machine, contourReplay.
// It reconstructs the crack graph from nothing but (special map, direction bits, turn symbols) and asserts the reconstruction equals the original graph.
// Every context feature recorded here is read out of that decoder state, so a feature that is not available to a real decoder cannot be recorded in the first place — this is the guard against falsification #12.

// Wire-up (main.go):
//	case "contourx":   contourx(os.Args[2], atoi(os.Args[3]))   // characterisation
//	case "turnprice":  turnprice(os.Args[2], atoi(os.Args[3]))  // side-by-side pricing, LABDUMP=<dir> to keep partitions
//	case "turnload":   turnload(os.Args[2])                     // re-price dumped partitions

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// histMod keeps the rolling base-3 turn history at ten digits, deeper than any context tested here.
const histMod = 59049 // 3^10

// turnEv is one coded turn, with every feature the decoder holds at the moment the symbol is read.
type turnEv struct {
	s      uint8  // 0 straight, 1 right, 2 left
	d      uint8  // direction of travel arriving at this vertex
	hist3  uint8  // the published order-3 turn history, 0..26, reset at the start of each chain
	hist   uint32 // full base-3 turn history of this chain, low digits most recent, saturating
	run    uint16 // consecutive straights immediately before this symbol, within this chain
	nsym   uint16 // symbols already coded in this chain
	lastNS uint8  // 0 = no non-straight yet in this chain, 1 = right, 2 = left
	spec   uint8  // bit k set when the candidate target for turn k is a special vertex
	spec2  uint8  // bit k set when the vertex two steps along turn k is special
	dstr   uint8  // steps along the straight ray to the nearest special vertex, capped at 7
	allow  uint8  // bit k set when turn k is not excluded by what the decoder already knows
	w16    uint32 // the last sixteen symbols of this chain, two bits each, most recent in the low bits
	pr     uint8  // length of the straight run before the last non-straight symbol, capped at 15
	ppr    uint8  // the run before that, capped at 15
}

// slopeOf reads a local slope estimate out of the rolling symbol window: how many of the last n symbols were straight, and the right-minus-left balance.
// A discretised smooth curve is a Sturmian sequence, so its slope is the thing an order-k history has to spell out digit by digit.
func slopeOf(w uint32, n, avail int) (nStr, bal int) {
	if n > avail {
		n = avail
	}
	for i := 0; i < n; i++ {
		switch (w >> (2 * uint(i))) & 3 {
		case 0:
			nStr++
		case 1:
			bal++
		case 2:
			bal--
		}
	}
	return
}

// contourReplay is contourBytes' traversal run as a decoder: it rebuilds the crack graph edge by edge.
// It returns the published vertBits (junction bitmap + direction bits + loop starts), the published turnBits, the event stream, and the number of edges walked.
// dec is the reconstructed graph; the caller asserts it against the original.
// lastLoops is the number of closed loops the last contourReplay walked; the published coder pays log2(nv)+2 bits for each explicit start but never transmits how many there are, which is a small unpaid side channel.
var lastLoops int

func contourReplay(lab []int32, w, h int) (vertBits, baseTurnBits float64, evs []turnEv, steps int, ok bool) {
	g := buildCrackGraph(lab, w, h)
	nv := g.vw * g.vh

	special := make([]bool, nv)
	for v := 0; v < nv; v++ {
		if d := g.deg(v); d != 2 && d != 0 {
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

	// ---- junction bitmap: unchanged from contourBytes, and fully transmitted before any turn ----
	occ := make([]byte, nv)
	for v := 0; v < nv; v++ {
		if special[v] && !free[v] {
			occ[v] = 1
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
			vertBits += mv[ctx].cost(int(occ[y*g.vw+x]))
		}
	}

	// ---- decoder state: everything below reads only from here ---- decE[d][v] is an edge the decoder has already been told about, by a direction bit or by a turn.
	decE := [4][]bool{}
	for d := 0; d < 4; d++ {
		decE[d] = make([]bool, nv)
	}
	decDeg := make([]uint8, nv)
	// dirKnown[v] is set once all four direction bits at special vertex v have been read, at which point the decoder knows v's edge set exactly.
	dirKnown := make([]bool, nv)

	setEdge := func(v, d int) {
		if decE[d][v] {
			return
		}
		nvv := (v/g.vw+dys[d])*g.vw + (v%g.vw + dxs[d])
		decE[d][v] = true
		decE[(d+2)%4][nvv] = true
		decDeg[v]++
		decDeg[nvv]++
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

	// turnDir maps a relative turn index (0 straight, 1 right, 2 left) to an absolute direction.
	turnDir := func(d, k int) int {
		switch k {
		case 1:
			return (d + 1) % 4
		case 2:
			return (d + 3) % 4
		}
		return d
	}

	// feature reads the decoder's view of the three candidate continuations at vertex v arriving with direction d.
	// Everything it reads is either the junction map (transmitted in full before any turn) or an edge the decoder has already been given.
	feature := func(v, d int) (spec, spec2, dstr, allow uint8) {
		x, y := v%g.vw, v/g.vw
		// distance along the straight ray to the next special vertex: the junction map is fully known, so the decoder can walk it too.
		for t := 1; t <= 7; t++ {
			nx, ny := x+dxs[d]*t, y+dys[d]*t
			if nx < 0 || ny < 0 || nx >= g.vw || ny >= g.vh {
				dstr = uint8(t)
				break
			}
			dstr = uint8(t)
			if special[ny*g.vw+nx] {
				break
			}
		}
		for k := 0; k < 3; k++ {
			nd2 := turnDir(d, k)
			if x2, y2 := x+2*dxs[nd2], y+2*dys[nd2]; x2 >= 0 && y2 >= 0 && x2 < g.vw && y2 < g.vh && special[y2*g.vw+x2] {
				spec2 |= 1 << k
			}
		}
		for k := 0; k < 3; k++ {
			nd := turnDir(d, k)
			nx, ny := x+dxs[nd], y+dys[nd]
			if nx < 0 || ny < 0 || nx >= g.vw || ny >= g.vh {
				continue // off-lattice: cannot be the continuation, so not allowed
			}
			u := ny*g.vw + nx
			if special[u] {
				spec |= 1 << k
			}
			switch {
			case special[u] && dirKnown[u]:
				// The decoder has read every direction bit at u, so it knows whether the edge back to v exists.
				if decE[(nd+2)%4][u] {
					allow |= 1 << k
				}
			case !special[u] && decDeg[u] >= 2:
				// A non-special vertex has degree 0 or 2; two edges are already known, so there is no room for this one.
			default:
				allow |= 1 << k
			}
		}
		return
	}

	// trace walks one chain, recording an event per turn. It is the decoder's loop: it stops at the first special vertex, exactly as the published coder does.
	trace := func(v, d int) {
		ctxh := 0
		var hist, w16 uint32
		var run, nsym uint16
		var lastNS, pr, ppr uint8
		for {
			used[d][v] = true
			setEdge(v, d)
			nx, ny := v%g.vw+dxs[d], v/g.vw+dys[d]
			nvv := ny*g.vw + nx
			used[(d+2)%4][nvv] = true
			steps++
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
			case 1:
				s = 1
			case 3:
				s = 2
			}
			sp, sp2, dst, al := feature(v, d)
			evs = append(evs, turnEv{
				s: uint8(s), d: uint8(d), hist3: uint8(ctxh), hist: hist,
				run: run, nsym: nsym, lastNS: lastNS, spec: sp, spec2: sp2, dstr: dst, allow: al,
				w16: w16, pr: pr, ppr: ppr,
			})
			baseTurnBits += -math.Log2(float64(mturn[ctxh][s]+1) / float64(tturn[ctxh]+3))
			mturn[ctxh][s]++
			tturn[ctxh]++
			ctxh = (ctxh*3 + s) % 27
			hist = (hist*3 + uint32(s)) % histMod
			w16 = (w16 << 2) | uint32(s)
			nsym++
			if s == 0 {
				run++
			} else {
				ppr, pr = pr, uint8(min(int(run), 15))
				run = 0
				lastNS = uint8(s)
			}
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
			vertBits += mdir[d*5+min(seen, 4)].cost(bit)
			if bit == 1 {
				setEdge(v, d)
				seen++
				if !used[d][v] {
					trace(v, d)
				}
			}
		}
		dirKnown[v] = true
	}

	// Closed loops with no special vertex, paid for with an explicit start.
	// trace2 in the published coder erases edges as it walks; the replay below does the same on a scratch copy so the original graph survives for the final assertion.
	type loopStart struct{ v, d int }
	var starts []loopStart
	for v := 0; v < nv; v++ {
		for d := 0; d < 4; d++ {
			if g.e[d][v] && !used[d][v] {
				starts = append(starts, loopStart{v, d})
				vertBits += math.Log2(float64(nv)) + 2
				// Walk the loop, marking it used, mirroring trace2's erase.
				vv, dd := v, d
				ctxh := 0
				var hist, w16 uint32
				var run, nsym uint16
				var lastNS, pr, ppr uint8
				for {
					used[dd][vv] = true
					setEdge(vv, dd)
					nx, ny := vv%g.vw+dxs[dd], vv/g.vw+dys[dd]
					nvv := ny*g.vw + nx
					used[(dd+2)%4][nvv] = true
					steps++
					vv = nvv
					if vv == v {
						break
					}
					nd := -1
					for k := 0; k < 4; k++ {
						if k != (dd+2)%4 && g.e[k][vv] && !used[k][vv] {
							nd = k
							break
						}
					}
					if nd < 0 {
						break
					}
					sym := (nd - dd + 4) % 4
					s := 0
					switch sym {
					case 1:
						s = 1
					case 3:
						s = 2
					}
					sp, sp2, dst, al := feature(vv, dd)
					evs = append(evs, turnEv{
						s: uint8(s), d: uint8(dd), hist3: uint8(ctxh), hist: hist,
						run: run, nsym: nsym, lastNS: lastNS, spec: sp, spec2: sp2, dstr: dst, allow: al,
						w16: w16, pr: pr, ppr: ppr,
					})
					baseTurnBits += -math.Log2(float64(mturn[ctxh][s]+1) / float64(tturn[ctxh]+3))
					mturn[ctxh][s]++
					tturn[ctxh]++
					ctxh = (ctxh*3 + s) % 27
					hist = (hist*3 + uint32(s)) % histMod
					w16 = (w16 << 2) | uint32(s)
					nsym++
					if s == 0 {
						run++
					} else {
						ppr, pr = pr, uint8(min(int(run), 15))
						run = 0
						lastNS = uint8(s)
					}
					dd = nd
				}
			}
		}
	}

	// ---- decodability assertion: the reconstructed edge set must equal the original ----
	ok = true
	for d := 0; d < 4 && ok; d++ {
		for v := 0; v < nv; v++ {
			if decE[d][v] != g.e[d][v] {
				ok = false
				break
			}
		}
	}
	lastLoops = len(starts)
	return vertBits, baseTurnBits, evs, steps, ok
}

// ---------------------------------------------------------------------------
// Context candidates. Each returns a context index and the number of contexts.
// Every field they read is decoder-side by construction of contourReplay.

type turnCtx struct {
	name  string
	n     int
	f     func(e turnEv) int
	excl  bool    // renormalise the model over the allowed set
	alpha float64 // Laplace weight; the published coder uses 1
}

func capRun(r uint16, k int) int {
	if int(r) >= k {
		return k - 1
	}
	return int(r)
}

func histK(e turnEv, k int) int {
	m := uint32(1)
	for i := 0; i < k; i++ {
		m *= 3
	}
	return int(e.hist % m)
}

func turnCtxs() []turnCtx {
	var out []turnCtx
	out = append(out, turnCtx{"base(ord3)", 27, func(e turnEv) int { return int(e.hist3) }, false, 0})
	for _, k := range []int{1, 2, 4, 5, 6} {
		k := k
		n := 1
		for i := 0; i < k; i++ {
			n *= 3
		}
		out = append(out, turnCtx{fmt.Sprintf("ord%d", k), n, func(e turnEv) int { return histK(e, k) }, false, 0})
	}
	for _, R := range []int{8, 16, 32, 64} {
		R := R
		out = append(out, turnCtx{fmt.Sprintf("run%d", R), R, func(e turnEv) int { return capRun(e.run, R) }, false, 0})
		out = append(out, turnCtx{fmt.Sprintf("run%d.lastNS", R), R * 3,
			func(e turnEv) int { return capRun(e.run, R)*3 + int(e.lastNS) }, false, 0})
	}
	out = append(out, turnCtx{"run32.lastNS.dir", 32 * 3 * 4,
		func(e turnEv) int { return (capRun(e.run, 32)*3+int(e.lastNS))*4 + int(e.d) }, false, 0})
	out = append(out, turnCtx{"ord3.dir", 27 * 4,
		func(e turnEv) int { return int(e.hist3)*4 + int(e.d) }, false, 0})
	out = append(out, turnCtx{"ord3.spec", 27 * 8,
		func(e turnEv) int { return int(e.hist3)*8 + int(e.spec) }, false, 0})
	out = append(out, turnCtx{"run32.lastNS.spec", 32 * 3 * 8,
		func(e turnEv) int { return (capRun(e.run, 32)*3+int(e.lastNS))*8 + int(e.spec) }, false, 0})
	out = append(out, turnCtx{"run32.lastNS.allow", 32 * 3 * 8,
		func(e turnEv) int { return (capRun(e.run, 32)*3+int(e.lastNS))*8 + int(e.allow) }, false, 0})
	// Exclusion-aware versions: same context, but the model is renormalised over the symbols the decoder cannot rule out.
	out = append(out, turnCtx{"base+excl", 27, func(e turnEv) int { return int(e.hist3) }, true, 0})
	out = append(out, turnCtx{"run32.lastNS+excl", 32 * 3,
		func(e turnEv) int { return capRun(e.run, 32)*3 + int(e.lastNS) }, true, 0})
	out = append(out, turnCtx{"run32.lastNS.spec+excl", 32 * 3 * 8,
		func(e turnEv) int { return (capRun(e.run, 32)*3+int(e.lastNS))*8 + int(e.spec) }, true, 0})
	out = append(out, turnCtx{"run64.lastNS.ord2+excl", 64 * 3 * 9,
		func(e turnEv) int { return (capRun(e.run, 64)*3+int(e.lastNS))*9 + histK(e, 2) }, true, 0})
	out = append(out, turnCtx{"spec+excl", 8, func(e turnEv) int { return int(e.spec) }, true, 0})
	out = append(out, turnCtx{"spec.spec2+excl", 64,
		func(e turnEv) int { return int(e.spec)*8 + int(e.spec2) }, true, 0})
	out = append(out, turnCtx{"ord2.spec+excl", 9 * 8,
		func(e turnEv) int { return histK(e, 2)*8 + int(e.spec) }, true, 0})
	out = append(out, turnCtx{"ord3.spec+excl", 27 * 8,
		func(e turnEv) int { return int(e.hist3)*8 + int(e.spec) }, true, 0})
	out = append(out, turnCtx{"ord3.spec.spec2+excl", 27 * 64,
		func(e turnEv) int { return int(e.hist3)*64 + int(e.spec)*8 + int(e.spec2) }, true, 0})
	out = append(out, turnCtx{"ord3.spec.dstr+excl", 27 * 8 * 8,
		func(e turnEv) int { return (int(e.hist3)*8+int(e.spec))*8 + int(e.dstr) }, true, 0})
	out = append(out, turnCtx{"run8.lastNS.spec+excl", 8 * 3 * 8,
		func(e turnEv) int { return (capRun(e.run, 8)*3+int(e.lastNS))*8 + int(e.spec) }, true, 0})
	out = append(out, turnCtx{"run8.lastNS.spec.spec2+excl", 8 * 3 * 64,
		func(e turnEv) int { return ((capRun(e.run, 8)*3+int(e.lastNS))*8+int(e.spec))*8 + int(e.spec2) }, true, 0})
	out = append(out, turnCtx{"run8.lastNS.spec.dir+excl", 8 * 3 * 8 * 4,
		func(e turnEv) int { return ((capRun(e.run, 8)*3+int(e.lastNS))*8+int(e.spec))*4 + int(e.d) }, true, 0})
	out = append(out, turnCtx{"ord3.spec.dir+excl", 27 * 8 * 4,
		func(e turnEv) int { return (int(e.hist3)*8+int(e.spec))*4 + int(e.d) }, true, 0})
	// Slope contexts: a discretised smooth curve is Sturmian, so what predicts the next symbol is the local slope, not the exact digit string.
	out = append(out, turnCtx{"slope8", 9 * 17,
		func(e turnEv) int { n, b := slopeOf(e.w16, 8, int(e.nsym)); return n*17 + b + 8 }, false, 0})
	out = append(out, turnCtx{"slope8.lastNS", 9 * 17 * 3,
		func(e turnEv) int { n, b := slopeOf(e.w16, 8, int(e.nsym)); return (n*17+b+8)*3 + int(e.lastNS) }, false, 0})
	out = append(out, turnCtx{"slope16", 17 * 33,
		func(e turnEv) int { n, b := slopeOf(e.w16, 16, int(e.nsym)); return n*33 + b + 16 }, false, 0})
	out = append(out, turnCtx{"ord3.slope8", 27 * 9 * 17,
		func(e turnEv) int { n, b := slopeOf(e.w16, 8, int(e.nsym)); return int(e.hist3)*153 + n*17 + b + 8 }, false, 0})
	out = append(out, turnCtx{"ord2.slope8", 9 * 9 * 17,
		func(e turnEv) int { n, b := slopeOf(e.w16, 8, int(e.nsym)); return histK(e, 2)*153 + n*17 + b + 8 }, false, 0})
	out = append(out, turnCtx{"runpair.lastNS", 16 * 16 * 3,
		func(e turnEv) int { return (int(e.pr)*16+int(e.ppr))*3 + int(e.lastNS) }, false, 0})
	out = append(out, turnCtx{"run8.runpair.lastNS", 8 * 16 * 16 * 3,
		func(e turnEv) int { return ((capRun(e.run, 8)*16+int(e.pr))*16+int(e.ppr))*3 + int(e.lastNS) }, false, 0})
	out = append(out, turnCtx{"ord3.spec.dir.slope8+excl", 27 * 8 * 4 * 153,
		func(e turnEv) int {
			n, b := slopeOf(e.w16, 8, int(e.nsym))
			return ((int(e.hist3)*8+int(e.spec))*4+int(e.d))*153 + n*17 + b + 8
		}, true, 0})
	// Estimator controls: the published coder adds one; a Krichevsky-Trofimov half is the textbook choice for a skewed alphabet.
	out = append(out, turnCtx{"base(ord3) a=0.5", 27, func(e turnEv) int { return int(e.hist3) }, false, 0.5})
	out = append(out, turnCtx{"base(ord3) a=0.2", 27, func(e turnEv) int { return int(e.hist3) }, false, 0.2})
	out = append(out, turnCtx{"ord3.spec.dir+excl a=0.5", 27 * 8 * 4,
		func(e turnEv) int { return (int(e.hist3)*8+int(e.spec))*4 + int(e.d) }, true, 0.5})
	out = append(out, turnCtx{"ord3.spec.dir+excl a=0.2", 27 * 8 * 4,
		func(e turnEv) int { return (int(e.hist3)*8+int(e.spec))*4 + int(e.d) }, true, 0.2})
	// Capacity controls: the same number of contexts built from information that is causal but should carry nothing extra.
	out = append(out, turnCtx{"ctrl:ord3.nsym(8)", 27 * 8,
		func(e turnEv) int { return int(e.hist3)*8 + int(e.nsym%8) }, false, 0})
	out = append(out, turnCtx{"ctrl:ord3.nsym(32)", 27 * 32,
		func(e turnEv) int { return int(e.hist3)*32 + int(e.nsym%32) }, false, 0})
	out = append(out, turnCtx{"ctrl:ord3.nsym(8)+excl", 27 * 8,
		func(e turnEv) int { return int(e.hist3)*8 + int(e.nsym%8) }, true, 0})
	return out
}

// adaptiveTurnBits prices the event stream with the same Laplace estimator the published coder uses, so the numbers are directly comparable.
func adaptiveTurnBits(evs []turnEv, c turnCtx) float64 {
	a := c.alpha
	if a == 0 {
		a = 1
	}
	cnt := make([]uint32, c.n*3)
	tot := make([]uint32, c.n)
	bits := 0.0
	for _, e := range evs {
		ci := c.f(e)
		base := ci * 3
		if !c.excl {
			bits += -math.Log2((float64(cnt[base+int(e.s)]) + a) / (float64(tot[ci]) + 3*a))
		} else {
			if e.allow&(1<<e.s) == 0 {
				// The exclusion rule ruled out the symbol that actually occurred: the rule is unsound, so make it visibly expensive.
				return math.Inf(1)
			}
			den := 0.0
			for k := 0; k < 3; k++ {
				if e.allow&(1<<k) != 0 {
					den += float64(cnt[base+k]) + a
				}
			}
			if den > 0 {
				bits += -math.Log2((float64(cnt[base+int(e.s)]) + a) / den)
			}
		}
		cnt[base+int(e.s)]++
		tot[ci]++
	}
	return bits
}

// staticTurnBits is the empirical conditional entropy: what the stream would cost with the counts known in advance and free.
// It is the floor for the context, and the gap to adaptiveTurnBits is the model-learning cost.
func staticTurnBits(evs []turnEv, c turnCtx) float64 {
	cnt := make([]uint32, c.n*3)
	tot := make([]uint32, c.n)
	for _, e := range evs {
		cnt[c.f(e)*3+int(e.s)]++
		tot[c.f(e)]++
	}
	bits := 0.0
	for ci := 0; ci < c.n; ci++ {
		if tot[ci] == 0 {
			continue
		}
		for k := 0; k < 3; k++ {
			if cnt[ci*3+k] > 0 {
				p := float64(cnt[ci*3+k]) / float64(tot[ci])
				bits += -float64(cnt[ci*3+k]) * math.Log2(p)
			}
		}
	}
	return bits
}

// ---------------------------------------------------------------------------

// turnStats prints the characterisation of one partition's turn stream.
func turnStats(lab []int32, w, h int, nreg int, psnr float64) {
	vb, tb, evs, steps, ok := contourReplay(lab, w, h)
	ref, refV, refT, refSteps := contourBytes(lab, w, h)
	fmt.Printf("## regions %d  psnr %.2f\n", nreg, psnr)
	fmt.Printf("published contourBytes: total %.0f B  vert %.0f B  turn %.0f B  steps %d\n", ref, refV, refT, refSteps)
	fmt.Printf("replay                : total %.0f B  vert %.0f B  turn %.0f B  steps %d  reconstructs_graph=%v\n",
		(vb+tb)/8, vb/8, tb/8, steps, ok)
	if math.Abs((vb+tb)/8-ref) > 1e-6 || steps != refSteps {
		fmt.Printf("!! REPLAY MISMATCH\n")
	}
	if !ok {
		fmt.Printf("!! REPLAY DID NOT RECONSTRUCT THE GRAPH\n")
	}

	// symbol distribution
	var ns [3]int
	for _, e := range evs {
		ns[e.s]++
	}
	n := len(evs)
	fmt.Printf("closed loops %d (unpaid: the coder prices each explicit start but never their count)\n", lastLoops)
	fmt.Printf("turns %d (%.1f%% of %d steps)  straight %.4f  right %.4f  left %.4f\n",
		n, 100*float64(n)/float64(steps), steps,
		float64(ns[0])/float64(n), float64(ns[1])/float64(n), float64(ns[2])/float64(n))
	fmt.Printf("baseline bits/turn %.4f   order-0 entropy %.4f b/turn\n", tb/float64(n), order0(ns[:], n))

	// run-length distribution of straight segments
	rl := map[int]int{}
	cur := 0
	for _, e := range evs {
		if e.s == 0 {
			cur++
		} else {
			rl[cur]++
			cur = 0
		}
	}
	if cur > 0 {
		rl[cur]++
	}
	keys := make([]int, 0, len(rl))
	tot := 0
	sum := 0
	for k, v := range rl {
		keys = append(keys, k)
		tot += v
		sum += k * v
	}
	sort.Ints(keys)
	fmt.Printf("straight-run segments %d  mean run %.3f  max run %d\n", tot, float64(sum)/float64(tot), keys[len(keys)-1])
	fmt.Printf("run-length histogram (len:share):")
	acc := 0
	for _, k := range keys {
		if k <= 12 {
			fmt.Printf(" %d:%.3f", k, float64(rl[k])/float64(tot))
			acc += rl[k]
		}
	}
	fmt.Printf("  >12:%.3f\n", float64(tot-acc)/float64(tot))

	// how often the exclusion rule bites
	var exc [4]int
	for _, e := range evs {
		na := 0
		for k := 0; k < 3; k++ {
			if e.allow&(1<<k) != 0 {
				na++
			}
		}
		exc[na]++
	}
	fmt.Printf("allowed-symbol count at coding time: 1 -> %.4f, 2 -> %.4f, 3 -> %.4f (0 -> %d, must be 0)\n",
		float64(exc[1])/float64(n), float64(exc[2])/float64(n), float64(exc[3])/float64(n), exc[0])

	// context table
	fmt.Printf("%-26s %10s %10s %12s %10s %10s\n", "context", "adapt_B", "static_B", "vs_base", "b/turn", "ctxs")
	baseAd := 0.0
	for i, c := range turnCtxs() {
		ad := adaptiveTurnBits(evs, c)
		st := staticTurnBits(evs, c)
		if i == 0 {
			baseAd = ad
			if math.Abs(ad-tb) > 1e-6 {
				fmt.Printf("!! base context does not reproduce contourBytes turn cost: %.6f vs %.6f\n", ad, tb)
			}
		}
		fmt.Printf("%-26s %10.0f %10.0f %11.2f%% %10.4f %10d\n",
			c.name, ad/8, st/8, 100*(ad-baseAd)/baseAd, ad/float64(n), c.n)
	}
	for _, th := range turnHiers() {
		ad := th.price(evs)
		fmt.Printf("%-26s %10.0f %10s %11.2f%% %10.4f %10s\n",
			th.name, ad/8, "-", 100*(ad-baseAd)/baseAd, ad/float64(n), "-")
	}
	fmt.Println()
	os.Stdout.Sync()
}

func order0(ns []int, n int) float64 {
	b := 0.0
	for _, c := range ns {
		if c > 0 {
			p := float64(c) / float64(n)
			b += -p * math.Log2(p)
		}
	}
	return b
}

// contourx walks the same scale-space hd walks and characterises the turn stream at every mark at or below maxReg.
func contourx(path string, maxReg int) {
	im := load(path)
	npix := im.W * im.H
	t0 := time.Now()
	fmt.Printf("# %s %dx%d — contour turn-coding characterisation, identical partitions to hd()\n", path, im.W, im.H)

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
			// Marks far above the pricing window cost minutes of relaxation for a row that is thrown away; the merger state they come from is untouched either way, so the priced partitions are bit-identical.
			if nreg > 2*maxReg {
				fmt.Fprintf(os.Stderr, "mark %d skipped (above pricing window) at %s\n", mm.nreg, time.Since(t0).Round(time.Second))
				mi++
				continue
			}
			lab = relax(im, lab, nreg, lambda*bitsPerEdge, 6)
			nr, ps, _, _, _ := priceSeg(im, lab)
			if nr <= maxReg {
				turnStats(lab, im.W, im.H, nr, ps)
			}
			fmt.Fprintf(os.Stderr, "mark %d (relaxed to %d) done at %s\n", mm.nreg, nr, time.Since(t0).Round(time.Second))
			mi++
		}
	})
	fmt.Printf("# total %s\n", time.Since(t0).Round(time.Second))
}

// ---------------------------------------------------------------------------
// Hierarchical blending.
//
// The characterisation says the same thing at every sparse operating point: the static conditional entropy under a wide context keeps falling while the adaptive cost turns over and rises.
// That is model-learning cost, not missing information — report 09 hit the same ceiling on the CAE side.
// Interpolating a coarse context into a fine one is the standard cure and costs nothing to transmit: no table, no side channel, no mode flag.

type turnHier struct {
	name string
	lv   []turnCtx // coarsest first
	excl bool
	k    float64 // interpolation weight; larger means slower to trust the finer context
}

func (th turnHier) price(evs []turnEv) float64 {
	cnt := make([][]uint32, len(th.lv))
	tot := make([][]uint32, len(th.lv))
	for i, c := range th.lv {
		cnt[i] = make([]uint32, c.n*3)
		tot[i] = make([]uint32, c.n)
	}
	bits := 0.0
	var p [3]float64
	for _, e := range evs {
		na := 0.0
		for k := 0; k < 3; k++ {
			if !th.excl || e.allow&(1<<k) != 0 {
				na++
			}
		}
		for k := 0; k < 3; k++ {
			if !th.excl || e.allow&(1<<k) != 0 {
				p[k] = 1 / na
			} else {
				p[k] = 0
			}
		}
		idx := make([]int, len(th.lv))
		for i, c := range th.lv {
			ci := c.f(e)
			idx[i] = ci
			den := float64(tot[i][ci]) + th.k
			for k := 0; k < 3; k++ {
				p[k] = (float64(cnt[i][ci*3+k]) + th.k*p[k]) / den
			}
		}
		if th.excl {
			sum := 0.0
			for k := 0; k < 3; k++ {
				if e.allow&(1<<k) != 0 {
					sum += p[k]
				}
			}
			if e.allow&(1<<e.s) == 0 {
				return math.Inf(1)
			}
			bits += -math.Log2(p[e.s] / sum)
		} else {
			bits += -math.Log2(p[e.s])
		}
		for i := range th.lv {
			cnt[i][idx[i]*3+int(e.s)]++
			tot[i][idx[i]]++
		}
	}
	return bits
}

// pickedCtxs is the curated set turnprice reports: the published coder, each new tap on its own, the frozen variant, and the controls.
func pickedCtxs() []turnCtx {
	want := map[string]bool{
		"base(ord3)": true, "ord3.dir": true, "ord3.spec": true, "base+excl": true,
		"ord3.spec.dir+excl": true, "ord5": true, "ctrl:ord3.nsym(32)": true,
	}
	var out []turnCtx
	for _, c := range turnCtxs() {
		if want[c.name] {
			out = append(out, c)
		}
	}
	return out
}

// pickedHiers is the curated hierarchy set: the frozen choice first.
func pickedHiers() []turnHier {
	want := map[string]bool{
		"H:ord2>3>spec>dir k=16": true, "H:ord2>3>4>5>spec>dir k=32": true,
		"H:ord2>3>spec>dir>slope8 k=8": true, "ctrl H:ord2>3>nsym k=8": true,
	}
	var out []turnHier
	for _, h := range turnHiers() {
		if want[h.name] {
			out = append(out, h)
		}
	}
	return out
}

func turnHiers() []turnHier {
	ord3 := turnCtx{"", 27, func(e turnEv) int { return int(e.hist3) }, false, 0}
	ord2 := turnCtx{"", 9, func(e turnEv) int { return histK(e, 2) }, false, 0}
	ord4 := turnCtx{"", 81, func(e turnEv) int { return histK(e, 4) }, false, 0}
	ord5 := turnCtx{"", 243, func(e turnEv) int { return histK(e, 5) }, false, 0}
	spec := turnCtx{"", 27 * 8, func(e turnEv) int { return int(e.hist3)*8 + int(e.spec) }, false, 0}
	specDir := turnCtx{"", 27 * 8 * 4, func(e turnEv) int { return (int(e.hist3)*8+int(e.spec))*4 + int(e.d) }, false, 0}
	sd8 := turnCtx{"", 27 * 8 * 4 * 153, func(e turnEv) int {
		n, b := slopeOf(e.w16, 8, int(e.nsym))
		return ((int(e.hist3)*8+int(e.spec))*4+int(e.d))*153 + n*17 + b + 8
	}, false, 0}
	var out []turnHier
	for _, k := range []float64{4, 8, 16} {
		out = append(out, turnHier{fmt.Sprintf("H:ord2>3>spec>dir k=%g", k), []turnCtx{ord2, ord3, spec, specDir}, true, k})
	}
	out = append(out, turnHier{"H:ord2>3>spec>dir (no excl) k=8", []turnCtx{ord2, ord3, spec, specDir}, false, 8})
	out = append(out, turnHier{"H:ord2>3>4>5 k=8", []turnCtx{ord2, ord3, ord4, ord5}, true, 8})
	out = append(out, turnHier{"H:ord2>3>spec>dir>slope8 k=8", []turnCtx{ord2, ord3, spec, specDir, sd8}, true, 8})
	out = append(out, turnHier{"H:ord2>3>4>5>spec>dir>slope8 k=8", []turnCtx{ord2, ord3, ord4, ord5, spec, specDir, sd8}, true, 8})
	ord5spec := turnCtx{"", 243 * 8, func(e turnEv) int { return histK(e, 5)*8 + int(e.spec) }, false, 0}
	ord5sd := turnCtx{"", 243 * 8 * 4, func(e turnEv) int { return (histK(e, 5)*8+int(e.spec))*4 + int(e.d) }, false, 0}
	ord6c := turnCtx{"", 729, func(e turnEv) int { return histK(e, 6) }, false, 0}
	for _, k := range []float64{8, 16} {
		out = append(out, turnHier{fmt.Sprintf("H:ord2>3>4>5>spec>dir k=%g", k),
			[]turnCtx{ord2, ord3, ord4, ord5, ord5spec, ord5sd}, true, k})
	}
	out = append(out, turnHier{"H:ord2>3>4>5>6>spec>dir k=16",
		[]turnCtx{ord2, ord3, ord4, ord5, ord6c, ord5spec, ord5sd}, true, 16})
	out = append(out, turnHier{"H:ord2>3>4>5>spec>dir k=32",
		[]turnCtx{ord2, ord3, ord4, ord5, ord5spec, ord5sd}, true, 32})
	// Control: the same depth of hierarchy, last level built from a causal field that carries nothing.
	ctrl := turnCtx{"", 27 * 8, func(e turnEv) int { return int(e.hist3)*8 + int(e.nsym%8) }, false, 0}
	out = append(out, turnHier{"ctrl H:ord2>3>nsym k=8", []turnCtx{ord2, ord3, ctrl}, true, 8})
	return out
}
