package main

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
)

// This file runs the physicist's segmentation: grain coarsening under a Potts / piecewise-constant Mumford-Shah energy
//
//	E = sum_R sum_{p in R} |I_p - c_R|^2  +  lambda * L
//
// where L is the total boundary length and lambda is a surface tension.
// Starting from one region per pixel and repeatedly performing the merge with the lowest lambda* = dSSE/dL is exactly the Koepfler-Lopez-Morel scale-space, and is the direct analogue of curvature-driven grain growth in a polycrystal: small grains with weak colour contrast are eaten first, and the region count falls as the surface energy is paid down.
// The output is the honest region count of an energy-minimising segmentation at a given fidelity, plus what it costs in bits.

type cand struct {
	a, b int32
	key  float64
}

type candHeap []cand

func (h candHeap) Len() int { return len(h) }

// Total order, not just key order. Ties on key are common (many pairs share an identical dSSE/dL early in the coarsening) and Go randomises map iteration, so ordering by key alone let the initial heap layout vary between runs and produced a different scale-space each time — up to 7% spread in bytes at the coarse end. Breaking ties on (a, b) makes the merge deterministic.
func (h candHeap) Less(i, j int) bool {
	if h[i].key != h[j].key {
		return h[i].key < h[j].key
	}
	if h[i].a != h[j].a {
		return h[i].a < h[j].a
	}
	return h[i].b < h[j].b
}
func (h candHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *candHeap) Push(x interface{}) { *h = append(*h, x.(cand)) }
func (h *candHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type merger struct {
	w, h   int
	parent []int32
	area   []int32
	sum    [][3]float64
	adj    []map[int32]int32
	sse    float64
	nreg   int
	blen   int
}

func (m *merger) find(x int32) int32 {
	for m.parent[x] != x {
		m.parent[x] = m.parent[m.parent[x]]
		x = m.parent[x]
	}
	return x
}

func (m *merger) mean(r int32) [3]float64 {
	a := float64(m.area[r])
	s := m.sum[r]
	return [3]float64{s[0] / a, s[1] / a, s[2] / a}
}

// dSSE is the exact squared-error increase from replacing two region means with their joint mean.
func (m *merger) dSSE(a, b int32) float64 {
	ca, cb := m.mean(a), m.mean(b)
	wa, wb := float64(m.area[a]), float64(m.area[b])
	f := wa * wb / (wa + wb)
	d := 0.0
	for i := 0; i < 3; i++ {
		e := ca[i] - cb[i]
		d += e * e
	}
	return f * d
}

func newMerger(im *Img) *merger {
	n := im.W * im.H
	m := &merger{w: im.W, h: im.H,
		parent: make([]int32, n), area: make([]int32, n),
		sum: make([][3]float64, n), adj: make([]map[int32]int32, n),
		nreg: n, blen: (im.W-1)*im.H + im.W*(im.H-1)}
	for i := 0; i < n; i++ {
		m.parent[i] = int32(i)
		m.area[i] = 1
		m.sum[i] = [3]float64{im.P[i*3], im.P[i*3+1], im.P[i*3+2]}
		m.adj[i] = make(map[int32]int32, 4)
	}
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			p := int32(y*im.W + x)
			if x < im.W-1 {
				m.adj[p][p+1] = 1
				m.adj[p+1][p] = 1
			}
			if y < im.H-1 {
				q := p + int32(im.W)
				m.adj[p][q] = 1
				m.adj[q][p] = 1
			}
		}
	}
	return m
}

// run coarsens until only `stop` regions remain, invoking snap after every merge so the caller can sample the scale-space.
func (m *merger) run(stop int, snap func(m *merger, lambda float64)) {
	h := &candHeap{}
	for i := range m.adj {
		for j := range m.adj[i] {
			if int32(i) < j {
				*h = append(*h, cand{int32(i), j, m.dSSE(int32(i), j) / float64(m.adj[i][j])})
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
		if key > c.key*1.000001 { // stale: the pair got more expensive after an earlier merge, so re-queue at its true scale
			heap.Push(h, cand{a, b, key})
			continue
		}
		// merge b into a (keep the larger adjacency map to bound total work)
		if len(m.adj[a]) < len(m.adj[b]) {
			a, b = b, a
		}
		m.sse += m.dSSE(a, b)
		m.blen -= int(l)
		m.nreg--
		m.area[a] += m.area[b]
		for i := 0; i < 3; i++ {
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
			heap.Push(h, cand{a, nb, m.dSSE(a, nb) / float64(m.adj[a][nb])})
		}
		snap(m, key)
	}
}

// labels materialises the current partition as a compact label field plus the per-region rounded mean colour.
func (m *merger) labels() ([]int32, [][3]float64) {
	lab := make([]int32, len(m.parent))
	idOf := map[int32]int32{}
	var cols [][3]float64
	for i := range m.parent {
		r := m.find(int32(i))
		id, ok := idOf[r]
		if !ok {
			id = int32(len(cols))
			idOf[r] = id
			c := m.mean(r)
			cols = append(cols, [3]float64{math.Round(c[0]), math.Round(c[1]), math.Round(c[2])})
		}
		lab[i] = id
	}
	return lab, cols
}

func potts(path string) {
	im := load(path)
	npix := im.W * im.H
	targetPSNR := 28.61
	targetSSE := math.Pow(10, -targetPSNR/10) * 255 * 255 * float64(npix*3)

	m := newMerger(im)
	fmt.Printf("image %dx%d, target PSNR %.2f dB -> SSE %.3e\n", im.W, im.H, targetPSNR, targetSSE)
	fmt.Printf("%10s %10s %8s %10s %8s\n", "regions", "crackLen", "PSNR", "lambda", "L/pix")

	marks := []int{65536, 32768, 16384, 8192, 4096, 2048, 1024, 512, 256, 128, 64}
	mi := 0
	var hitRegions, hitLen int
	var hitLab []int32
	var hitCols [][3]float64
	hitDone := false

	m.run(32, func(mm *merger, lambda float64) {
		if !hitDone && mm.sse >= targetSSE {
			hitDone = true
			hitRegions, hitLen = mm.nreg, mm.blen
			hitLab, hitCols = mm.labels()
			fmt.Printf("%10d %10d %8.2f %10.1f %8.3f   <== at target fidelity\n",
				mm.nreg, mm.blen, psnrSSE(mm.sse, npix), lambda, float64(mm.blen)/float64(npix))
		}
		for mi < len(marks) && mm.nreg == marks[mi] {
			fmt.Printf("%10d %10d %8.2f %10.1f %8.3f\n",
				mm.nreg, mm.blen, psnrSSE(mm.sse, npix), lambda, float64(mm.blen)/float64(npix))
			mi++
		}
	})

	if !hitDone {
		fmt.Println("never reached target SSE")
		return
	}

	// --- what the segmentation costs in bits -------------------------------
	bBoundary := caeBytes(hitLab, im.W, im.H)
	bColor := colorBytes(hitLab, hitCols, im.W, im.H)
	total := bBoundary + bColor
	fmt.Printf("\nenergy-minimising segmentation at %.2f dB: %d regions, %d crack edges (%.2f px boundary per region)\n",
		targetPSNR, hitRegions, hitLen, float64(hitLen)/float64(hitRegions))
	fmt.Printf("  boundary (CAE, 2 crack planes, 10-bit ctx) : %s  (%.3f bits/crack edge)\n",
		kb(bBoundary), bBoundary*8/float64(hitLen))
	fmt.Printf("  region colours (predicted from neighbour)  : %s  (%.2f bits/region)\n",
		kb(bColor), bColor*8/float64(len(hitCols)))
	fmt.Printf("  TOTAL region coder                         : %s  (%.3f bpp)\n",
		kb(total), total*8/float64(npix))

	// --- region-size distribution: is it heavy tailed? ---------------------
	sizes := make([]int, len(hitCols))
	for _, l := range hitLab {
		sizes[l]++
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))
	fmt.Printf("\nregion-size distribution: max %d, median %d, mean %.1f px\n",
		sizes[0], sizes[len(sizes)/2], float64(npix)/float64(len(sizes)))
	// Maximum-likelihood Pareto exponent for the tail above xmin=4 px (Clauset et al.).
	xmin := 4.0
	var s float64
	nTail := 0
	for _, sz := range sizes {
		if float64(sz) >= xmin {
			s += math.Log(float64(sz) / xmin)
			nTail++
		}
	}
	alpha := 1 + float64(nTail)/s
	fmt.Printf("tail (>=%.0f px, n=%d): Pareto alpha = %.3f\n", xmin, nTail, alpha)
	fmt.Printf("decile sizes: ")
	for i := 0; i < 10; i++ {
		fmt.Printf("%d ", sizes[i*len(sizes)/10])
	}
	fmt.Println()

	// Paint the segmentation so it can be inspected and fed to external codecs.
	out := &Img{W: im.W, H: im.H, P: make([]float64, npix*3)}
	for i, l := range hitLab {
		c := hitCols[l]
		out.P[i*3], out.P[i*3+1], out.P[i*3+2] = c[0], c[1], c[2]
	}
	out.writePNG("potts_seg.png")
	fmt.Printf("wrote potts_seg.png (PSNR after rounding colours: %.2f dB)\n", psnrSSE(sseBetween(im, out), npix))
}

// caeBytes codes the partition as its two crack-edge planes with an adaptive binary arithmetic coder and a 10-bit causal template.
// This is exactly the context-based arithmetic coding that MPEG-4 chose over explicit vertex/polygon shape coding, so it is the strongest defensible cost for an explicit region map.
func caeBytes(lab []int32, w, h int) float64 {
	V := make([]byte, w*h) // crack between (x,y) and (x+1,y)
	Hz := make([]byte, w*h)
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
	get := func(a []byte, x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		return int(a[y*w+x])
	}
	bits := 0.0
	mv := make([]binModel, 1024)
	for y := 0; y < h; y++ {
		for x := 0; x < w-1; x++ {
			ctx := get(V, x-1, y) | get(V, x-2, y)<<1 | get(V, x, y-1)<<2 | get(V, x-1, y-1)<<3 |
				get(V, x+1, y-1)<<4 | get(V, x+2, y-1)<<5 | get(V, x, y-2)<<6 | get(V, x-1, y-2)<<7 |
				get(V, x+1, y-2)<<8 | get(V, x-2, y-1)<<9
			bits += mv[ctx].cost(int(V[y*w+x]))
		}
	}
	mh := make([]binModel, 1024)
	for y := 0; y < h-1; y++ {
		for x := 0; x < w; x++ {
			ctx := get(Hz, x-1, y) | get(Hz, x, y-1)<<1 | get(Hz, x-1, y-1)<<2 | get(Hz, x+1, y-1)<<3 |
				get(Hz, x+1, y)<<4 | get(V, x, y)<<5 | get(V, x-1, y)<<6 | get(V, x, y+1)<<7 |
				get(V, x-1, y+1)<<8 | get(Hz, x-2, y)<<9
			bits += mh[ctx].cost(int(Hz[y*w+x]))
		}
	}
	return bits / 8
}

// colorBytes codes each region's colour once, predicted from the already-decoded neighbouring region with the longest shared boundary, with an adaptive per-channel residual model.
// The decoder already owns the partition at this point, so this is the whole remaining cost.
func colorBytes(lab []int32, cols [][3]float64, w, h int) float64 {
	n := len(cols)
	// First raster appearance defines coding order; neighbours seen earlier are available as predictors.
	firstAt := make([]int, n)
	for i := range firstAt {
		firstAt[i] = -1
	}
	for i, l := range lab {
		if firstAt[l] < 0 {
			firstAt[l] = i
		}
	}
	// For each region, the earlier-ordered neighbour sharing the longest boundary.
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
	models := make([][]binModel, 3) // 3 channels x (8 bit-planes of a 9-bit residual) is overkill; use symbol counts instead
	_ = models
	counts := make([][]uint32, 3)
	totals := make([]uint32, 3)
	for c := range counts {
		counts[c] = make([]uint32, 512)
	}
	bits := 0.0
	for r := 0; r < n; r++ {
		var pred [3]float64
		best, bestLen := int32(-1), 0
		for nb, ln := range share[r] {
			if int(nb) < r && ln > bestLen {
				best, bestLen = nb, ln
			}
		}
		if best >= 0 {
			pred = cols[best]
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
