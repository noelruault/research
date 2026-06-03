package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
)

// crossdomain.go — palette selection borrowed from disciplines far outside
// color science, each implemented so it can be measured against the champion
// and kept or discarded with a number (research report 09):
//
//   - SpaceCurve : crypto/databases. A Morton (Z-order) space-filling curve
//     linearizes 3-D OKLab; cut the sorted run into N equal-weight segments.
//   - MSTCluster : astrophysics. Friends-of-Friends / minimum-spanning-tree
//     single-linkage — the cosmic-web halo finder on the color cloud.
//   - DetAnneal  : statistical mechanics. Deterministic annealing — free-energy
//     soft assignment with a cooling temperature, the principled escape from
//     k-means local minima.
//
// All operate in OKLab (the champion's space) and are deterministic.

// okQuant maps an OKLab vector to 8-bit-per-axis integers for the curve.
func okQuant(v Vec3) (uint8, uint8, uint8) {
	l := clamp8(v[0] * 255)
	a := clamp8((v[1] + 0.4) / 0.8 * 255)
	b := clamp8((v[2] + 0.4) / 0.8 * 255)
	return l, a, b
}

// morton3 interleaves three 8-bit values into a 24-bit Z-order key.
func morton3(r, g, b uint8) uint32 {
	var d uint32
	for i := uint(0); i < 8; i++ {
		d |= uint32((r>>i)&1) << (3*i + 0)
		d |= uint32((g>>i)&1) << (3*i + 1)
		d |= uint32((b>>i)&1) << (3*i + 2)
	}
	return d
}

// ---- SpaceCurve (Morton / Z-order) -----------------------------------------

type SpaceCurve struct{}

func (SpaceCurve) Name() string { return "spacecurve/morton/oklab" }

func (SpaceCurve) Quantize(img image.Image, n int) color.Palette {
	sp := OKLabSpace{}
	h := histogram(img)
	type item struct {
		key uint32
		p   Vec3
		w   float64
	}
	items := make([]item, len(h))
	var total float64
	for i, c := range h {
		p := sp.FromRGB(c.R, c.G, c.B)
		l, a, b := okQuant(p)
		items[i] = item{key: morton3(l, a, b), p: p, w: float64(c.Count)}
		total += float64(c.Count)
	}
	if len(items) <= n {
		return paletteOfPoints(weightedPoints(h, sp), sp)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })

	// Cut into N equal-weight segments; each segment's weighted mean is a color.
	cents := make([]Vec3, 0, n)
	seg := total / float64(n)
	var acc, sum Vec3Acc
	var thresh = seg
	for _, it := range items {
		sum.add(it.p, it.w)
		acc.w += it.w
		if acc.w >= thresh && len(cents) < n-1 {
			cents = append(cents, sum.mean())
			sum = Vec3Acc{}
			thresh += seg
			_ = acc
		}
	}
	cents = append(cents, sum.mean())
	return centsToPalette(cents, sp)
}

// Vec3Acc accumulates a weighted mean.
type Vec3Acc struct {
	s Vec3
	w float64
}

func (a *Vec3Acc) add(p Vec3, w float64) { a.s = a.s.add(p.scale(w)); a.w += w }
func (a Vec3Acc) mean() Vec3 {
	if a.w == 0 {
		return Vec3{}
	}
	return a.s.scale(1.0 / a.w)
}

// ---- MSTCluster (Friends-of-Friends / single-linkage) ----------------------

type MSTCluster struct{}

func (MSTCluster) Name() string { return "mst/foF/oklab" }

const mstCap = 1500

func (MSTCluster) Quantize(img image.Image, n int) color.Palette {
	sp := OKLabSpace{}
	h := binHistogram(histogram(img), mstCap)
	pts := weightedPoints(h, sp)
	B := len(pts)
	if B <= n {
		return paletteOfPoints(pts, sp)
	}
	// Prim's MST over the complete graph; record each node's connecting edge.
	const inf = math.MaxFloat64
	inTree := make([]bool, B)
	best := make([]float64, B)
	from := make([]int, B)
	for i := range best {
		best[i] = inf
		from[i] = -1
	}
	best[0] = 0
	type edge struct {
		u, v int
		w    float64
	}
	edges := make([]edge, 0, B-1)
	for k := 0; k < B; k++ {
		u, ud := -1, inf
		for i := 0; i < B; i++ {
			if !inTree[i] && best[i] < ud {
				u, ud = i, best[i]
			}
		}
		if u < 0 {
			break
		}
		inTree[u] = true
		if from[u] >= 0 {
			edges = append(edges, edge{from[u], u, best[u]})
		}
		for vtx := 0; vtx < B; vtx++ {
			if !inTree[vtx] {
				if d := pts[u].P.dist2(pts[vtx].P); d < best[vtx] {
					best[vtx], from[vtx] = d, u
				}
			}
		}
	}
	// Cut the n-1 longest edges → n connected components (single-linkage).
	sort.Slice(edges, func(i, j int) bool { return edges[i].w > edges[j].w })
	uf := newUF(B)
	cut := n - 1
	for _, e := range edges {
		if cut > 0 {
			cut--
			continue // skip the longest n-1 edges
		}
		uf.union(e.u, e.v)
	}
	// Centroid each component.
	acc := map[int]*Vec3Acc{}
	for i := 0; i < B; i++ {
		r := uf.find(i)
		if acc[r] == nil {
			acc[r] = &Vec3Acc{}
		}
		acc[r].add(pts[i].P, pts[i].W)
	}
	cents := make([]Vec3, 0, len(acc))
	for _, a := range acc {
		cents = append(cents, a.mean())
	}
	return centsToPalette(cents, sp)
}

type uf struct{ p, r []int }

func newUF(n int) *uf {
	u := &uf{p: make([]int, n), r: make([]int, n)}
	for i := range u.p {
		u.p[i] = i
	}
	return u
}
func (u *uf) find(x int) int {
	for u.p[x] != x {
		u.p[x] = u.p[u.p[x]]
		x = u.p[x]
	}
	return x
}
func (u *uf) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return
	}
	if u.r[ra] < u.r[rb] {
		ra, rb = rb, ra
	}
	u.p[rb] = ra
	if u.r[ra] == u.r[rb] {
		u.r[ra]++
	}
}

// ---- DetAnneal (deterministic annealing) -----------------------------------

type DetAnneal struct {
	Init  Quantizer
	Steps int
}

func (d DetAnneal) Name() string { return fmt.Sprintf("detanneal[%s]/s%d", d.Init.Name(), d.Steps) }

func (d DetAnneal) Quantize(img image.Image, n int) color.Palette {
	sp := OKLabSpace{}
	pts := weightedPoints(binHistogram(histogram(img), mstCap), sp)
	if len(pts) <= n {
		return paletteOfPoints(pts, sp)
	}
	ip := d.Init.Quantize(img, n)
	cents := make([]Vec3, len(ip))
	for i, c := range ip {
		r, g, b, _ := c.RGBA()
		cents[i] = sp.FromRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
	// Cool the temperature geometrically; at high T assignment is soft (free-
	// energy minimization), at low T it hardens into k-means.
	T := 0.02
	const tmin = 2e-5
	factor := math.Pow(tmin/T, 1.0/float64(d.Steps))
	probs := make([]float64, len(cents))
	for step := 0; step < d.Steps; step++ {
		sums := make([]Vec3, len(cents))
		wts := make([]float64, len(cents))
		for _, wp := range pts {
			var z float64
			for j, c := range cents {
				probs[j] = math.Exp(-wp.P.dist2(c) / T)
				z += probs[j]
			}
			if z == 0 {
				continue
			}
			for j := range cents {
				p := probs[j] / z * wp.W
				sums[j] = sums[j].add(wp.P.scale(p))
				wts[j] += p
			}
		}
		for j := range cents {
			if wts[j] > 0 {
				cents[j] = sums[j].scale(1.0 / wts[j])
			}
		}
		T *= factor
	}
	return centsToPalette(cents, sp)
}
