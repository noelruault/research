package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"sort"
)

// extras.go — the interdisciplinary candidates from research report 07, each
// implemented so it can be measured and then kept or discarded with a number
// rather than an argument:
//   - MultiRestart : k-means from R deterministic seeds, keep lowest-SSE (the
//                    cheap, deterministic substitute for simulated annealing).
//   - HyABKMeans   : k-means under the HyAB metric (city-block L + Euclidean
//                    chroma) — the large-difference perceptual metric.
//   - PNN          : Pairwise Nearest Neighbor agglomerative init (Ward merge
//                    cost), the best-known deterministic VQ initializer.

// ---- shared Lloyd helpers ---------------------------------------------------

func lloyd(pts []WPoint, cents []Vec3, iters int) []Vec3 {
	for it := 0; it < iters; it++ {
		sums := make([]Vec3, len(cents))
		wts := make([]float64, len(cents))
		for _, wp := range pts {
			j := nearestCentroid(wp.P, cents)
			sums[j] = sums[j].add(wp.P.scale(wp.W))
			wts[j] += wp.W
		}
		for j := range cents {
			if wts[j] > 0 {
				cents[j] = sums[j].scale(1.0 / wts[j])
			}
		}
	}
	return cents
}

func weightedSSE(pts []WPoint, cents []Vec3) float64 {
	var sse float64
	for _, wp := range pts {
		j := nearestCentroid(wp.P, cents)
		sse += wp.W * wp.P.dist2(cents[j])
	}
	return sse
}

// ---- MultiRestart -----------------------------------------------------------

type MultiRestart struct {
	Space    Space
	Restarts int
	Iters    int
}

func (m MultiRestart) Name() string {
	return fmt.Sprintf("multirestart/%s/r%d/i%d", m.Space.Name(), m.Restarts, m.Iters)
}

func (m MultiRestart) Quantize(img image.Image, n int) color.Palette {
	pts := weightedPoints(histogram(img), m.Space)
	if len(pts) <= n {
		return paletteOfPoints(pts, m.Space)
	}
	var best []Vec3
	bestSSE := math.Inf(1)
	for r := 0; r < m.Restarts; r++ {
		cents := seedKMeansPPSeed(pts, n, int64(r+1))
		cents = lloyd(pts, cents, m.Iters)
		if sse := weightedSSE(pts, cents); sse < bestSSE {
			bestSSE, best = sse, cents
		}
	}
	return centsToPalette(best, m.Space)
}

// seedKMeansPPSeed is seedKMeansPP with an explicit RNG seed (for restarts).
func seedKMeansPPSeed(pts []WPoint, n int, seed int64) []Vec3 {
	rng := rand.New(rand.NewSource(seed))
	cents := make([]Vec3, 0, n)
	first := rng.Intn(len(pts))
	cents = append(cents, pts[first].P)
	d2 := make([]float64, len(pts))
	for i := range pts {
		d2[i] = pts[i].P.dist2(pts[first].P)
	}
	for len(cents) < n {
		var total float64
		for i := range pts {
			total += d2[i] * pts[i].W
		}
		if total <= 0 {
			break
		}
		target := rng.Float64() * total
		pick, acc := len(pts)-1, 0.0
		for i := range pts {
			acc += d2[i] * pts[i].W
			if acc >= target {
				pick = i
				break
			}
		}
		cents = append(cents, pts[pick].P)
		for i := range pts {
			if d := pts[i].P.dist2(pts[pick].P); d < d2[i] {
				d2[i] = d
			}
		}
	}
	return cents
}

// ---- HyAB k-means -----------------------------------------------------------

// HyABKMeans clusters in OKLab under HyAB = |dL| + sqrt(da^2+db^2). The
// centroid that minimizes HyAB is the weighted MEDIAN of L and the weighted
// MEAN of (a,b). Seeded from Init (OKLab divisive).
type HyABKMeans struct {
	Init  Quantizer
	Iters int
}

func (h HyABKMeans) Name() string { return fmt.Sprintf("hyab[%s]/i%d", h.Init.Name(), h.Iters) }

func hyab(a, b Vec3) float64 {
	dL := a[0] - b[0]
	da, db := a[1]-b[1], a[2]-b[2]
	return math.Abs(dL) + math.Sqrt(da*da+db*db)
}

func (h HyABKMeans) Quantize(img image.Image, n int) color.Palette {
	sp := OKLabSpace{}
	pts := weightedPoints(histogram(img), sp)
	if len(pts) <= n {
		return paletteOfPoints(pts, sp)
	}
	ip := h.Init.Quantize(img, n)
	cents := make([]Vec3, len(ip))
	for i, c := range ip {
		r, g, b, _ := c.RGBA()
		cents[i] = sp.FromRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
	for it := 0; it < h.Iters; it++ {
		groups := make([][]WPoint, len(cents))
		for _, wp := range pts {
			best, bd := 0, hyab(wp.P, cents[0])
			for j := 1; j < len(cents); j++ {
				if d := hyab(wp.P, cents[j]); d < bd {
					best, bd = j, d
				}
			}
			groups[best] = append(groups[best], wp)
		}
		for j := range cents {
			if len(groups[j]) == 0 {
				continue
			}
			cents[j] = Vec3{weightedMedianL(groups[j]), weightedMean(groups[j], 1), weightedMean(groups[j], 2)}
		}
	}
	return centsToPalette(cents, sp)
}

func weightedMean(g []WPoint, ax int) float64 {
	var s, w float64
	for _, p := range g {
		s += p.P[ax] * p.W
		w += p.W
	}
	if w == 0 {
		return 0
	}
	return s / w
}

func weightedMedianL(g []WPoint) float64 {
	sort.Slice(g, func(i, j int) bool { return g[i].P[0] < g[j].P[0] })
	var total float64
	for _, p := range g {
		total += p.W
	}
	half, acc := total/2, 0.0
	for _, p := range g {
		acc += p.W
		if acc >= half {
			return p.P[0]
		}
	}
	return g[len(g)-1].P[0]
}

// ---- PNN (Pairwise Nearest Neighbor) ---------------------------------------

// PNN builds the palette by agglomerative merging: start from the (binned)
// histogram, repeatedly merge the pair with the least Ward distortion increase
// until n clusters remain. Deterministic. Binned to <=pnnCap start points to
// keep the O(B^2) merge tractable.
type PNN struct{ Space Space }

func (p PNN) Name() string { return fmt.Sprintf("pnn/%s", p.Space.Name()) }

const pnnCap = 2000

type pnnCl struct {
	c     Vec3
	w     float64
	alive bool
	nn    int
	cost  float64
}

func wardCost(a, b pnnCl) float64 {
	return (a.w * b.w / (a.w + b.w)) * a.c.dist2(b.c)
}

func (p PNN) Quantize(img image.Image, n int) color.Palette {
	h := binHistogram(histogram(img), pnnCap)
	cl := make([]pnnCl, len(h))
	for i, cc := range h {
		cl[i] = pnnCl{c: p.Space.FromRGB(cc.R, cc.G, cc.B), w: float64(cc.Count), alive: true}
	}
	if len(cl) <= n {
		return centsToPalette(clCentroids(cl), p.Space)
	}
	findNN := func(i int) {
		cl[i].cost = math.Inf(1)
		for j := range cl {
			if j == i || !cl[j].alive {
				continue
			}
			if c := wardCost(cl[i], cl[j]); c < cl[i].cost {
				cl[i].cost, cl[i].nn = c, j
			}
		}
	}
	for i := range cl {
		findNN(i)
	}
	count := len(cl)
	for count > n {
		// global min-cost alive cluster
		a := -1
		for i := range cl {
			if cl[i].alive && (a < 0 || cl[i].cost < cl[a].cost) {
				a = i
			}
		}
		b := cl[a].nn
		// merge b into a
		w := cl[a].w + cl[b].w
		cl[a].c = cl[a].c.scale(cl[a].w / w).add(cl[b].c.scale(cl[b].w / w))
		cl[a].w = w
		cl[b].alive = false
		count--
		// recompute NN for a and any cluster pointing at a or b
		findNN(a)
		for i := range cl {
			if cl[i].alive && i != a && (cl[i].nn == a || cl[i].nn == b) {
				findNN(i)
			}
		}
	}
	return centsToPalette(clCentroids(cl), p.Space)
}

func clCentroids(cl []pnnCl) []Vec3 {
	var out []Vec3
	for _, c := range cl {
		if c.alive {
			out = append(out, c.c)
		}
	}
	return out
}

// binHistogram coarsens an RGB histogram until it has <= cap distinct entries,
// summing weights — the standard "bin first" prepass that makes PNN tractable.
func binHistogram(h []ColorCount, cap int) []ColorCount {
	for shift := 0; shift <= 6; shift++ {
		m := make(map[uint32]int, len(h))
		for _, c := range h {
			r := (c.R >> shift) << shift
			g := (c.G >> shift) << shift
			b := (c.B >> shift) << shift
			m[uint32(r)<<16|uint32(g)<<8|uint32(b)] += c.Count
		}
		if len(m) <= cap || shift == 6 {
			out := make([]ColorCount, 0, len(m))
			for k, cnt := range m {
				out = append(out, ColorCount{R: uint8(k >> 16), G: uint8(k >> 8), B: uint8(k), Count: cnt})
			}
			return out
		}
	}
	return h
}

// ---- shared palette builders ------------------------------------------------

func centsToPalette(cents []Vec3, sp Space) color.Palette {
	pal := make(color.Palette, len(cents))
	for i, c := range cents {
		pal[i] = sp.ToRGBA(c)
	}
	return pal
}

func paletteOfPoints(pts []WPoint, sp Space) color.Palette {
	pal := make(color.Palette, len(pts))
	for i, p := range pts {
		pal[i] = sp.ToRGBA(p.P)
	}
	return pal
}
