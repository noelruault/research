package main

import "math"

// After relaxation the walls are minimal-curvature curves, so their information is in their turns, not in a raster of wall/no-wall decisions.
// This codes the partition the way a physicist would describe a foam: the vertices where three walls meet, then each wall as a sequence of turns from one vertex to the next.
// It is the explicit-geometry coder that MPEG-4 rejected in favour of CAE, measured here on walls that have been straightened first, which is the one condition under which it could plausibly win.
//
// Decodability: the decoder first receives the set of special vertices (wall junctions and endpoints); the image frame is special by definition and free.
// It then walks the special vertices in raster order and, for each of the four outgoing directions, receives a bit for whether a wall leaves there.
// Each wall is traced by decoding turns until it arrives at a vertex already known to be special, so no length or terminator is ever transmitted.
// Closed loops that touch no special vertex are sent separately with an explicit start.

type crackGraph struct {
	vw, vh int
	// e[d][v] is true when the wall edge leaving vertex v in direction d exists; d: 0=+x, 1=+y, 2=-x, 3=-y.
	e [4][]bool
}

var dxs = [4]int{1, 0, -1, 0}
var dys = [4]int{0, 1, 0, -1}

func buildCrackGraph(lab []int32, w, h int) *crackGraph {
	g := &crackGraph{vw: w + 1, vh: h + 1}
	for d := 0; d < 4; d++ {
		g.e[d] = make([]bool, g.vw*g.vh)
	}
	set := func(vx, vy, d int) {
		v := vy*g.vw + vx
		g.e[d][v] = true
		nv := (vy+dys[d])*g.vw + (vx + dxs[d])
		g.e[(d+2)%4][nv] = true
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				set(x+1, y, 1) // wall segment running downwards between the two pixels
			}
			if y < h-1 && lab[p] != lab[p+w] {
				set(x, y+1, 0) // wall segment running rightwards
			}
		}
	}
	return g
}

func (g *crackGraph) deg(v int) int {
	n := 0
	for d := 0; d < 4; d++ {
		if g.e[d][v] {
			n++
		}
	}
	return n
}

func (g *crackGraph) onFrame(v int) bool {
	x, y := v%g.vw, v/g.vw
	return x == 0 || y == 0 || x == g.vw-1 || y == g.vh-1
}

// contourBytes is the total cost of the partition under the turn-sequence description.
func contourBytes(lab []int32, w, h int) (total, vertBits, turnBits float64, steps int) {
	g := buildCrackGraph(lab, w, h)
	nv := g.vw * g.vh

	special := make([]bool, nv)
	for v := 0; v < nv; v++ {
		d := g.deg(v)
		if d != 2 && d != 0 {
			special[v] = true
		}
	}
	// The frame is known to the decoder, so its vertices cost nothing and are always usable as endpoints.
	free := make([]bool, nv)
	for v := 0; v < nv; v++ {
		if g.onFrame(v) {
			free[v] = true
			special[v] = true
		}
	}

	// Cost of telling the decoder where the junctions are: a context-coded sparse bitmap over the interior lattice.
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

	// One presence bit per outgoing direction at every special vertex, conditioned on the vertex's position class and how many walls have already been seen there.
	mdir := make([]binModel, 4*5)
	used := [4][]bool{}
	for d := 0; d < 4; d++ {
		used[d] = make([]bool, nv)
	}
	// Turn models: order-3 over the alphabet {straight, left, right}.
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
			steps++
			v = nvv
			if special[v] {
				return
			}
			// Degree is exactly 2 here, so the wall has one continuation and it is one of three turns.
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
			sym := (nd - d + 4) % 4 // 0 straight, 1 right, 3 left
			s := 0
			switch sym {
			case 0:
				s = 0
			case 1:
				s = 1
			case 3:
				s = 2
			default:
				s = 0
			}
			turnBits += -math.Log2(float64(mturn[ctxh][s]+1) / float64(tturn[ctxh]+3))
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
			vertBits += mdir[d*5+min(seen, 4)].cost(bit)
			if bit == 1 {
				seen++
				if !used[d][v] {
					trace(v, d)
				}
			}
		}
	}
	// Closed loops with no special vertex: pay an explicit start (position plus direction) for each.
	loops := 0
	for v := 0; v < nv; v++ {
		for d := 0; d < 4; d++ {
			if g.e[d][v] && !used[d][v] {
				loops++
				vertBits += math.Log2(float64(nv)) + 2
				trace2(g, v, d, mturn, tturn, &turnBits, &steps)
			}
		}
	}
	total = (vertBits + turnBits) / 8
	return total, vertBits / 8, turnBits / 8, steps
}

// trace2 walks a closed loop, which by construction has no special vertex, so it terminates when it returns to its start.
func trace2(g *crackGraph, v0, d0 int, mturn [][]uint32, tturn []uint32, turnBits *float64, steps *int) {
	v, d := v0, d0
	ctxh := 0
	for {
		g.e[d][v] = false
		nx, ny := v%g.vw+dxs[d], v/g.vw+dys[d]
		nvv := ny*g.vw + nx
		g.e[(d+2)%4][nvv] = false
		*steps++
		v = nvv
		if v == v0 {
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
		if sym == 1 {
			s = 1
		} else if sym == 3 {
			s = 2
		}
		*turnBits += -math.Log2(float64(mturn[ctxh][s]+1) / float64(tturn[ctxh]+3))
		mturn[ctxh][s]++
		tturn[ctxh]++
		ctxh = (ctxh*3 + s) % 27
		d = nd
	}
}
