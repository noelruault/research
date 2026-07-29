package main

import (
	"container/heap"
	"fmt"
	"math"
)

// Piecewise-constant regions are the wrong physics for a photograph: a smooth sky is not a set of plateaus, it is one surface with a slope, and forcing plateaus onto it is exactly what manufactures the false contours that inflate the region count.
// This file runs the same coarsening under a first-order Mumford-Shah energy, where every region carries an affine colour plane instead of a constant.
// Physically it is the difference between a Potts model (integer spins, domain walls everywhere) and an elastic sheet with defects (smooth field, walls only at genuine discontinuities).

// astat holds the sufficient statistics for a least-squares plane fit, so two regions can be merged and refitted in O(1).
type astat struct {
	n, sx, sy, sxx, sxy, syy float64
	sc, sxc, syc, scc        [3]float64
}

func (a *astat) add(b *astat) {
	a.n += b.n
	a.sx += b.sx
	a.sy += b.sy
	a.sxx += b.sxx
	a.sxy += b.sxy
	a.syy += b.syy
	for c := 0; c < 3; c++ {
		a.sc[c] += b.sc[c]
		a.sxc[c] += b.sxc[c]
		a.syc[c] += b.syc[c]
		a.scc[c] += b.scc[c]
	}
}

// fit returns the residual sum of squares of the best affine plane through the region, using centred moments so the 2x2 system stays well conditioned.
func (a *astat) fit() (sse float64) {
	if a.n < 1 {
		return 0
	}
	cx, cy := a.sx/a.n, a.sy/a.n
	mxx := a.sxx - a.n*cx*cx
	mxy := a.sxy - a.n*cx*cy
	myy := a.syy - a.n*cy*cy
	det := mxx*myy - mxy*mxy
	for c := 0; c < 3; c++ {
		base := a.scc[c] - a.sc[c]*a.sc[c]/a.n
		if det > 1e-6 {
			bx := a.sxc[c] - cx*a.sc[c]
			by := a.syc[c] - cy*a.sc[c]
			gb := (myy*bx - mxy*by) / det
			gd := (mxx*by - mxy*bx) / det
			base -= gb*bx + gd*by
		}
		if base < 0 {
			base = 0
		}
		sse += base
	}
	return sse
}

// plane returns the quantised affine coefficients actually transmitted: an integer offset at the region centroid and a slope on a fixed step, so the reported fidelity is the fidelity of the decoded parameters, not of an unquantised fit.
func (a *astat) plane(qg float64) (off [3]int, gx, gy [3]int, cx, cy float64) {
	cx, cy = a.sx/a.n, a.sy/a.n
	mxx := a.sxx - a.n*cx*cx
	mxy := a.sxy - a.n*cx*cy
	myy := a.syy - a.n*cy*cy
	det := mxx*myy - mxy*mxy
	for c := 0; c < 3; c++ {
		var gb, gd float64
		if det > 1e-6 {
			bx := a.sxc[c] - cx*a.sc[c]
			by := a.syc[c] - cy*a.sc[c]
			gb = (myy*bx - mxy*by) / det
			gd = (mxx*by - mxy*bx) / det
		}
		off[c] = int(math.Round(a.sc[c] / a.n))
		gx[c] = int(math.Round(gb / qg))
		gy[c] = int(math.Round(gd / qg))
		if gx[c] > 127 {
			gx[c] = 127
		}
		if gx[c] < -127 {
			gx[c] = -127
		}
		if gy[c] > 127 {
			gy[c] = 127
		}
		if gy[c] < -127 {
			gy[c] = -127
		}
	}
	return
}

// amerger coarsens an existing partition under the first-order energy.
type amerger struct {
	parent []int32
	st     []astat
	adj    []map[int32]int32
	sse    []float64
	nreg   int
	blen   int
}

func (m *amerger) find(x int32) int32 {
	for m.parent[x] != x {
		m.parent[x] = m.parent[m.parent[x]]
		x = m.parent[x]
	}
	return x
}

func (m *amerger) dSSE(a, b int32) float64 {
	j := m.st[a]
	j.add(&m.st[b])
	d := j.fit() - m.sse[a] - m.sse[b]
	if d < 0 {
		d = 0
	}
	return d
}

func newAmerger(im *Img, lab []int32, nlab int) *amerger {
	m := &amerger{parent: make([]int32, nlab), st: make([]astat, nlab),
		adj: make([]map[int32]int32, nlab), sse: make([]float64, nlab), nreg: nlab}
	for i := range m.parent {
		m.parent[i] = int32(i)
		m.adj[i] = map[int32]int32{}
	}
	for p, l := range lab {
		x, y := float64(p%im.W), float64(p/im.W)
		s := &m.st[l]
		s.n++
		s.sx += x
		s.sy += y
		s.sxx += x * x
		s.sxy += x * y
		s.syy += y * y
		for c := 0; c < 3; c++ {
			v := im.P[p*3+c]
			s.sc[c] += v
			s.sxc[c] += x * v
			s.syc[c] += y * v
			s.scc[c] += v * v
		}
	}
	for i := range m.st {
		m.sse[i] = m.st[i].fit()
	}
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			p := y*im.W + x
			if x < im.W-1 && lab[p] != lab[p+1] {
				m.adj[lab[p]][lab[p+1]]++
				m.adj[lab[p+1]][lab[p]]++
				m.blen++
			}
			if y < im.H-1 && lab[p] != lab[p+im.W] {
				m.adj[lab[p]][lab[p+im.W]]++
				m.adj[lab[p+im.W]][lab[p]]++
				m.blen++
			}
		}
	}
	return m
}

func (m *amerger) run(stop int, snap func(*amerger)) {
	h := &candHeap{}
	for i := range m.adj {
		for j, l := range m.adj[i] {
			if int32(i) < j {
				*h = append(*h, cand{int32(i), j, m.dSSE(int32(i), j) / float64(l)})
			}
		}
	}
	heap.Init(h)
	for m.nreg > stop && h.Len() > 0 {
		c := heap.Pop(h).(cand)
		a, b := m.find(c.a), m.find(c.b)
		if a == b {
			continue
		}
		l, ok := m.adj[a][b]
		if !ok {
			continue
		}
		key := m.dSSE(a, b) / float64(l)
		if key > c.key*1.000001 {
			heap.Push(h, cand{a, b, key})
			continue
		}
		if len(m.adj[a]) < len(m.adj[b]) {
			a, b = b, a
		}
		m.st[a].add(&m.st[b])
		m.sse[a] = m.st[a].fit()
		m.blen -= int(l)
		m.nreg--
		m.parent[b] = a
		delete(m.adj[a], b)
		for nb, ln := range m.adj[b] {
			if nb == a {
				continue
			}
			m.adj[a][nb] += ln
			m.adj[nb][a] = m.adj[a][nb]
			delete(m.adj[nb], b)
		}
		m.adj[b] = nil
		for nb := range m.adj[a] {
			heap.Push(h, cand{a, nb, m.dSSE(a, nb) / float64(m.adj[a][nb])})
		}
		snap(m)
	}
}

// affine runs constant coarsening down to a fine partition, then first-order coarsening, and prices the result honestly at every scale.
func affine(path string) {
	im := load(path)
	npix := im.W * im.H

	// Phase 1: constant-model coarsening produces a well-structured fine partition (a plane fits any 1-2 pixel region exactly, so starting the first-order merge from single pixels has a degenerate, arbitrary ordering).
	m0 := newMerger(im)
	m0.run(16384, func(*merger, float64) {})
	lab0, _ := m0.labels()
	nlab := 0
	for _, l := range lab0 {
		if int(l) >= nlab {
			nlab = int(l) + 1
		}
	}
	fmt.Printf("seed partition: %d regions, %d crack edges\n", nlab, crackLen(lab0, im.W, im.H))

	am := newAmerger(im, lab0, nlab)
	marks := map[int]bool{2048: true, 1024: true, 768: true, 512: true, 384: true, 256: true, 192: true, 128: true, 96: true, 64: true, 48: true, 32: true}
	type snapT struct {
		n   int
		par []int32
	}
	var snaps []snapT
	am.run(24, func(a *amerger) {
		if marks[a.nreg] {
			p := make([]int32, len(a.parent))
			copy(p, a.parent)
			snaps = append(snaps, snapT{a.nreg, p})
		}
	})

	fmt.Printf("\n%8s %10s %8s %8s %12s %12s %12s %8s\n", "regions", "crackLen", "PSNRq", "PSNRexact", "boundary", "planes", "total", "bpp")
	for i := len(snaps) - 1; i >= 0; i-- {
		s := snaps[i]
		// Re-label the pixel grid through the snapshot's union-find.
		find := func(x int32) int32 {
			for s.par[x] != x {
				x = s.par[x]
			}
			return x
		}
		idOf := map[int32]int32{}
		lab := make([]int32, npix)
		for p, l := range lab0 {
			r := find(l)
			id, ok := idOf[r]
			if !ok {
				id = int32(len(idOf))
				idOf[r] = id
			}
			lab[p] = id
		}
		n := len(idOf)
		rec, off, gx, gy, cxs, cys := renderAffine(im, lab, n)
		ps := psnrSSE(sseBetween(im, rec), npix)
		psExact := psnrSSE(exactAffineSSE(im, lab, n), npix)
		bB := caeBytes(lab, im.W, im.H)
		bP := planeBytes(lab, off, gx, gy, cxs, cys, im.W, im.H)
		tot := bB + bP
		fmt.Printf("%8d %10d %8.2f %8.2f %12s %12s %12s %8.3f\n",
			n, crackLen(lab, im.W, im.H), ps, psExact, kb(bB), kb(bP), kb(tot), tot*8/float64(npix))
		if ps >= 28.61 && (i == 0 || true) {
			rec.writePNG(fmt.Sprintf("affine_%d.png", n))
		}
	}
}

const qGrad = 0.125 // slope quantisation step, in colour levels per pixel

// renderAffine paints every region with its quantised colour plane, so the measured fidelity is what a decoder would actually reconstruct.
func renderAffine(im *Img, lab []int32, n int) (*Img, [][3]int, [][3]int, [][3]int, []float64, []float64) {
	st := make([]astat, n)
	for p, l := range lab {
		x, y := float64(p%im.W), float64(p/im.W)
		s := &st[l]
		s.n++
		s.sx += x
		s.sy += y
		s.sxx += x * x
		s.sxy += x * y
		s.syy += y * y
		for c := 0; c < 3; c++ {
			v := im.P[p*3+c]
			s.sc[c] += v
			s.sxc[c] += x * v
			s.syc[c] += y * v
			s.scc[c] += v * v
		}
	}
	off := make([][3]int, n)
	gx := make([][3]int, n)
	gy := make([][3]int, n)
	cxs := make([]float64, n)
	cys := make([]float64, n)
	for k := 0; k < n; k++ {
		off[k], gx[k], gy[k], cxs[k], cys[k] = st[k].plane(qGrad)
	}
	out := &Img{W: im.W, H: im.H, P: make([]float64, len(im.P))}
	for p, l := range lab {
		x, y := float64(p%im.W), float64(p/im.W)
		for c := 0; c < 3; c++ {
			v := float64(off[l][c]) + float64(gx[l][c])*qGrad*(x-cxs[l]) + float64(gy[l][c])*qGrad*(y-cys[l])
			out.P[p*3+c] = float64(clamp8(v))
		}
	}
	return out, off, gx, gy, cxs, cys
}

// planeBytes prices the per-region parameters: the offset is predicted from the neighbouring region's plane evaluated at this region's centroid, the slopes from that neighbour's slopes.
func planeBytes(lab []int32, off, gx, gy [][3]int, cxs, cys []float64, w, h int) float64 {
	n := len(off)
	share := make([]map[int32]int, n)
	for i := range share {
		share[i] = map[int32]int{}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				share[lab[p]][lab[p+1]]++
				share[lab[p+1]][lab[p]]++
			}
			if y < h-1 && lab[p] != lab[p+w] {
				share[lab[p]][lab[p+w]]++
				share[lab[p+w]][lab[p]]++
			}
		}
	}
	cOff := make([][]uint32, 3)
	cG := make([][]uint32, 3)
	tOff := make([]uint32, 3)
	tG := make([]uint32, 3)
	for c := 0; c < 3; c++ {
		cOff[c] = make([]uint32, 512)
		cG[c] = make([]uint32, 512)
	}
	bits := 0.0
	code := func(counts []uint32, tot *uint32, v int) {
		d := v + 256
		if d < 0 {
			d = 0
		}
		if d > 511 {
			d = 511
		}
		bits += -math.Log2(float64(counts[d]+1) / float64(*tot+512))
		counts[d]++
		*tot++
	}
	for r := 0; r < n; r++ {
		best, bestLen := int32(-1), 0
		for nb, ln := range share[r] {
			if int(nb) < r && ln > bestLen {
				best, bestLen = nb, ln
			}
		}
		for c := 0; c < 3; c++ {
			predOff, predGx, predGy := 128, 0, 0
			if best >= 0 {
				b := int(best)
				predOff = int(math.Round(float64(off[b][c]) +
					float64(gx[b][c])*qGrad*(cxs[r]-cxs[b]) + float64(gy[b][c])*qGrad*(cys[r]-cys[b])))
				predGx, predGy = gx[b][c], gy[b][c]
			}
			code(cOff[c], &tOff[c], off[r][c]-predOff)
			code(cG[c], &tG[c], gx[r][c]-predGx)
			code(cG[c], &tG[c], gy[r][c]-predGy)
		}
	}
	return bits / 8
}

// exactAffineSSE is the residual of the unquantised plane fits, which isolates how much of the first-order model's advantage the parameter quantisation gives back.
func exactAffineSSE(im *Img, lab []int32, n int) float64 {
	st := make([]astat, n)
	for p, l := range lab {
		x, y := float64(p%im.W), float64(p/im.W)
		s := &st[l]
		s.n++
		s.sx += x
		s.sy += y
		s.sxx += x * x
		s.sxy += x * y
		s.syy += y * y
		for c := 0; c < 3; c++ {
			v := im.P[p*3+c]
			s.sc[c] += v
			s.sxc[c] += x * v
			s.syc[c] += y * v
			s.scc[c] += v * v
		}
	}
	t := 0.0
	for k := 0; k < n; k++ {
		t += st[k].fit()
	}
	return t
}
