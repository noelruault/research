package main

import (
	"container/heap"
	"fmt"
	"math"
)

// The first Potts run minimised E = SSE + lambda*L, which prices boundary length but not the per-region colour that also has to be transmitted.
// Here the energy is the true Lagrangian of the coder, E = SSE + lambda*(bitsForBoundary + bitsForColour), so the coarsening is rate-distortion optimal for the actual bitstream rather than for an idealised surface energy.
// Then the interfaces are relaxed at fixed topology: single pixels hop across a wall whenever that lowers E, which is zero-temperature Ising/Potts dynamics and is the discrete analogue of mean-curvature flow.
// Smoother walls are literally cheaper walls, because the context coder pays for every unpredictable turn.

const (
	bitsPerEdge = 1.73 // measured cost of one crack edge under the CAE coder
	bitsPerReg  = 25.0 // measured cost of one region's colour
)

// runRD is the merger loop with the rate-aware key.
func (m *merger) runRD(stop int, snap func(*merger, float64)) {
	key := func(a, b int32, l int32) float64 {
		return m.dSSE(a, b) / (float64(l)*bitsPerEdge + bitsPerReg)
	}
	h := &candHeap{}
	for i := range m.adj {
		for j, l := range m.adj[i] {
			if int32(i) < j {
				*h = append(*h, cand{int32(i), j, key(int32(i), j, l)})
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
		k := key(a, b, l)
		if k > c.key*1.000001 {
			heap.Push(h, cand{a, b, k})
			continue
		}
		if len(m.adj[a]) < len(m.adj[b]) {
			a, b = b, a
		}
		m.sse += m.dSSE(a, b)
		m.blen -= int(l)
		m.nreg--
		m.area[a] += m.area[b]
		for i := 0; i < 4; i++ {
			m.sum[a][i] += m.sum[b][i]
		}
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
			heap.Push(h, cand{a, nb, key(a, nb, m.adj[a][nb])})
		}
		snap(m, k)
	}
}

// relax runs zero-temperature Potts dynamics: a pixel joins a neighbouring region whenever that lowers SSE + lambda*L.
// Walls straighten and short spurs evaporate, which is Plateau's law doing the coder's compaction for it.
func relax(im *Img, lab []int32, n int, lambda float64, sweeps int) []int32 {
	w, h := im.W, im.H
	area := make([]float64, n)
	sum := make([][4]float64, n)
	for p, l := range lab {
		area[l]++
		for c := 0; c < 3; c++ {
			sum[l][c] += im.P[p*3+c]
		}
		sum[l][3] += im.alphaAt(p)
	}
	mean := func(r int32) [4]float64 {
		a := area[r]
		if a == 0 {
			return [4]float64{0, 0, 0, 0}
		}
		return [4]float64{sum[r][0] / a, sum[r][1] / a, sum[r][2] / a, sum[r][3] / a}
	}
	// Four channels here too, or relaxation would undo what the merge protected: a pixel would hop across a transparency edge whenever the colours matched, which is the A1 failure again one sweep later.
	d2 := func(p int, c [4]float64) float64 {
		s := 0.0
		for i := 0; i < 3; i++ {
			e := im.P[p*3+i] - c[i]
			s += e * e
		}
		e := im.alphaAt(p) - c[3]
		return s + e*e
	}
	nbrs := func(p int) []int {
		var o []int
		x, y := p%w, p/w
		if x > 0 {
			o = append(o, p-1)
		}
		if x < w-1 {
			o = append(o, p+1)
		}
		if y > 0 {
			o = append(o, p-w)
		}
		if y < h-1 {
			o = append(o, p+w)
		}
		return o
	}
	for s := 0; s < sweeps; s++ {
		moved := 0
		for p := range lab {
			a := lab[p]
			if area[a] <= 1 {
				continue
			}
			ns := nbrs(p)
			// A 4-neighbourhood offers at most four distinct destination regions, so collect them into a fixed array kept in ascending id order rather than a map.
			// The map this replaces was both an allocation per pixel and a third place where Go's randomised iteration chose between equally good moves; ascending id makes the tie-break deterministic.
			var cand [4]int32
			nc := 0
			for _, q := range ns {
				r := lab[q]
				if r == a {
					continue
				}
				i := 0
				for i < nc && cand[i] < r {
					i++
				}
				if i < nc && cand[i] == r {
					continue
				}
				copy(cand[i+1:nc+1], cand[i:nc])
				cand[i] = r
				nc++
			}
			if nc == 0 {
				continue
			}
			ca := mean(a)
			// leaving a: SSE rises by -(na/(na-1))*|I-ca|^2, i.e. it falls
			dOut := -(area[a] / (area[a] - 1)) * d2(p, ca)
			var bestR int32 = -1
			bestD := -1e-9
			for _, r := range cand[:nc] {
				cb := mean(r)
				dIn := (area[r] / (area[r] + 1)) * d2(p, cb)
				// boundary change: edges to a become walls, edges to r stop being walls
				dL := 0
				for _, q := range ns {
					if lab[q] == a {
						dL++
					} else if lab[q] == r {
						dL--
					}
				}
				dE := dOut + dIn + lambda*float64(dL)
				if dE < bestD {
					bestD, bestR = dE, r
				}
			}
			if bestR >= 0 {
				for c := 0; c < 3; c++ {
					sum[a][c] -= im.P[p*3+c]
					sum[bestR][c] += im.P[p*3+c]
				}
				sum[a][3] -= im.alphaAt(p)
				sum[bestR][3] += im.alphaAt(p)
				area[a]--
				area[bestR]++
				lab[p] = bestR
				moved++
			}
		}
		if moved == 0 {
			break
		}
	}
	// The decoder recovers regions as connected components of the crack map, so relabel that way and price what it actually sees.
	return relabelComponents(lab, w, h)
}

func relabelComponents(lab []int32, w, h int) []int32 {
	out := make([]int32, len(lab))
	for i := range out {
		out[i] = -1
	}
	var next int32
	stack := make([]int32, 0, 1024)
	for i := range lab {
		if out[i] >= 0 {
			continue
		}
		id := next
		next++
		out[i] = id
		v := lab[i]
		stack = append(stack[:0], int32(i))
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := int(p)%w, int(p)/w
			try := func(q int32) {
				if out[q] < 0 && lab[q] == v {
					out[q] = id
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
				try(p - int32(w))
			}
			if y < h-1 {
				try(p + int32(w))
			}
		}
	}
	return out
}

// priceSeg is the full honest cost and fidelity of a partition: region means rounded to integers, boundary by CAE, colours predicted from already-decoded neighbours.
func priceSeg(im *Img, lab []int32) (regions int, psnr, bBound, bCol float64, rec *Img) {
	n := 0
	for _, l := range lab {
		if int(l)+1 > n {
			n = int(l) + 1
		}
	}
	sum := make([][3]float64, n)
	cnt := make([]float64, n)
	for p, l := range lab {
		cnt[l]++
		for c := 0; c < 3; c++ {
			sum[l][c] += im.P[p*3+c]
		}
	}
	cols := make([][3]float64, n)
	for k := 0; k < n; k++ {
		if cnt[k] == 0 {
			continue
		}
		for c := 0; c < 3; c++ {
			cols[k][c] = math.Round(sum[k][c] / cnt[k])
		}
	}
	rec = &Img{W: im.W, H: im.H, P: make([]float64, len(im.P))}
	// Carry the partition's alpha into the render. Without this the render is opaque everywhere,
	// the silhouette the merge just protected is invisible in the output, and any measurement
	// taken on the render reads the alpha work as having done nothing.
	if a := regionAlphas(im, lab, n); a != nil {
		rec.A = make([]float64, im.W*im.H)
		for p, l := range lab {
			rec.A[p] = a[l]
		}
	}
	for p, l := range lab {
		for c := 0; c < 3; c++ {
			rec.P[p*3+c] = cols[l][c]
		}
	}
	return n, psnrSSE(sseBetween(im, rec), im.W*im.H), caeBytes(lab, im.W, im.H), colorBytes2(lab, cols, im.W, im.H), rec
}

// colorBytes2 predicts a region's colour from the boundary-length-weighted mean of its already-decoded neighbours, which is strictly more information than the single longest neighbour.
func colorBytes2(lab []int32, cols [][3]float64, w, h int) float64 {
	n := len(cols)
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
	counts := make([][]uint32, 3)
	totals := make([]uint32, 3)
	for c := range counts {
		counts[c] = make([]uint32, 512)
	}
	bits := 0.0
	for r := 0; r < n; r++ {
		var acc [3]float64
		wsum := 0.0
		for nb, ln := range share[r] {
			if int(nb) < r {
				for c := 0; c < 3; c++ {
					acc[c] += float64(ln) * cols[nb][c]
				}
				wsum += float64(ln)
			}
		}
		var pred [3]float64
		if wsum > 0 {
			for c := 0; c < 3; c++ {
				pred[c] = math.Round(acc[c] / wsum)
			}
		} else {
			pred = [3]float64{128, 128, 128}
		}
		for c := 0; c < 3; c++ {
			d := int(cols[r][c]-pred[c]) + 256
			if d < 0 {
				d = 0
			}
			if d > 511 {
				d = 511
			}
			bits += -math.Log2(float64(counts[c][d]+1) / float64(totals[c]+512))
			counts[c][d]++
			totals[c]++
		}
	}
	return bits / 8
}

// frontier walks the rate-distortion frontier of the region coder and prints the operating point nearest the eval fidelity.
func frontier(path string) {
	im := load(path)
	npix := im.W * im.H
	fmt.Printf("image %dx%d\n", im.W, im.H)

	m := newMerger(im)
	marks := map[int]bool{}
	// pushed well past the original ceiling: the fixed eval sits at 28.61 dB, and 512 regions only reached 26.5 dB on this image, so the frontier has to run out to a few thousand regions before the comparison is on the right operating point
	// extended down into the very-low-rate regime: WebP's known weakness is deep quantization, and shape cost falls as sqrt(region count), so this is the only band where the two curves could still cross
	for _, k := range []int{8000, 6000, 4500, 3500, 2800, 2200, 1800, 1400, 1100, 900, 700, 512, 384, 256, 180, 128, 90, 64, 45, 32} {
		marks[k] = true
	}
	type snapT struct {
		n      int
		par    []int32
		lambda float64
	}
	var snaps []snapT
	m.runRD(96, func(mm *merger, lambda float64) {
		if marks[mm.nreg] {
			p := make([]int32, len(mm.parent))
			for i := range p {
				p[i] = mm.find(int32(i))
			}
			snaps = append(snaps, snapT{mm.nreg, p, lambda})
		}
	})

	fmt.Printf("\n%-20s %7s %7s %8s %8s %8s %8s %8s %8s %7s\n",
		"variant", "regions", "PSNR", "crackLen", "CAE", "b/edge", "contour", "colours", "total", "bpp")
	for i := len(snaps) - 1; i >= 0; i-- {
		s := snaps[i]
		base := relabelComponents(s.par, im.W, im.H)
		for _, sw := range []int{0, 6, 30} {
			lab := make([]int32, len(base))
			copy(lab, base)
			tag := fmt.Sprintf("merge only n=%d", s.n)
			if sw > 0 {
				nreg := 0
				for _, l := range lab {
					if int(l)+1 > nreg {
						nreg = int(l) + 1
					}
				}
				lab = relax(im, lab, nreg, s.lambda*bitsPerEdge, sw)
				tag = fmt.Sprintf("+relax%d n=%d", sw, s.n)
			}
			nr, ps, bb, bc, rec := priceSeg(im, lab)
			cl := crackLen(lab, im.W, im.H)
			ct, _, _, _ := contourBytes(lab, im.W, im.H)
			tot := math.Min(bb, ct) + bc
			fmt.Printf("%-20s %7d %7.2f %8d %8s %8.3f %8s %8s %8s %7.3f\n",
				tag, nr, ps, cl, kb(bb), bb*8/float64(cl), kb(ct), kb(bc), kb(tot), tot*8/float64(npix))
			if sw == 30 {
				// every fully-relaxed mark gets a render, so the dashboard can show the actual picture at each operating point instead of only at the eval's
				rec.writePNG(fmt.Sprintf("render_%04d_%.2f.png", nr, ps))
			}
			if ps >= 28.60 && ps <= 28.70 && sw == 6 {
				rec.writePNG(fmt.Sprintf("frontier_%d_%d.png", nr, sw))
				scaling(im, lab, nr, cl, bb, bc)
			}
		}
	}
}
