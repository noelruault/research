package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
)

// kmeans.go — weighted Lloyd / k-means (piece P3 selection + P5 refinement),
// the workhorse the cross-disciplinary research (report 01) points to. The
// assignment step is a nearest-centroid query — the same primitive pixelize
// ships as an exact kd-tree. Here we use a linear scan over n<=256 centroids
// (fine for the harness); the engine reuses its kd-tree.
//
// Seeding is the lever: research found deterministic maximin (Gonzalez)
// seeding the best-evidenced recipe, with k-means++ and median-cut init as
// alternatives. All seedings here are deterministic (fixed RNG seed) so the
// output is reproducible.

type KMeans struct {
	Space Space
	Init  Quantizer // if non-nil, seed centroids from this palette (e.g. median cut)
	Seed  string    // used when Init==nil: "maximin" | "kmeans++" | "random"
	Iters int
}

func (k KMeans) Name() string {
	if k.Init != nil {
		return fmt.Sprintf("kmeans[%s]/%s/i%d", k.Init.Name(), k.Space.Name(), k.Iters)
	}
	return fmt.Sprintf("kmeans/%s/%s/i%d", k.Seed, k.Space.Name(), k.Iters)
}

func (k KMeans) Quantize(img image.Image, n int) color.Palette {
	h := histogram(img)
	pts := weightedPoints(h, k.Space)
	if len(pts) <= n {
		pal := make(color.Palette, len(pts))
		for i, c := range h {
			pal[i] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
		}
		return pal
	}

	cents := k.seed(img, pts, n)
	for it := 0; it < k.Iters; it++ {
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

	pal := make(color.Palette, len(cents))
	for j, c := range cents {
		pal[j] = k.Space.ToRGBA(c)
	}
	return pal
}

func (k KMeans) seed(img image.Image, pts []WPoint, n int) []Vec3 {
	if k.Init != nil {
		ip := k.Init.Quantize(img, n)
		cents := make([]Vec3, len(ip))
		for i, c := range ip {
			r, g, b, _ := c.RGBA()
			cents[i] = k.Space.FromRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
		}
		return cents
	}
	switch k.Seed {
	case "maximin":
		return seedMaximin(pts, n)
	case "kmeans++":
		return seedKMeansPP(pts, n)
	default:
		return seedRandom(pts, n)
	}
}

func nearestCentroid(p Vec3, cents []Vec3) int {
	best, bestD := 0, p.dist2(cents[0])
	for j := 1; j < len(cents); j++ {
		if d := p.dist2(cents[j]); d < bestD {
			best, bestD = j, d
		}
	}
	return best
}

// seedMaximin is deterministic Gonzalez farthest-point seeding: center 1 is
// the highest-weight point (the most frequent color), each subsequent center
// is the point with the greatest minimum distance to the chosen set. Its
// inner loop is a nearest-color query.
func seedMaximin(pts []WPoint, n int) []Vec3 {
	cents := make([]Vec3, 0, n)
	first := 0
	for i, wp := range pts {
		if wp.W > pts[first].W {
			first = i
		}
	}
	cents = append(cents, pts[first].P)
	minD := make([]float64, len(pts))
	for i := range pts {
		minD[i] = pts[i].P.dist2(pts[first].P)
	}
	for len(cents) < n {
		far := -1
		var farD float64 = -1
		for i := range pts {
			if minD[i] > farD {
				farD, far = minD[i], i
			}
		}
		if far < 0 || farD <= 0 {
			break // no distinct point left
		}
		cents = append(cents, pts[far].P)
		for i := range pts {
			if d := pts[i].P.dist2(pts[far].P); d < minD[i] {
				minD[i] = d
			}
		}
	}
	return cents
}

// seedKMeansPP is D^2-weighted seeding with a fixed RNG seed (deterministic).
func seedKMeansPP(pts []WPoint, n int) []Vec3 {
	rng := rand.New(rand.NewSource(1))
	cents := make([]Vec3, 0, n)
	first := 0
	for i, wp := range pts {
		if wp.W > pts[first].W {
			first = i
		}
	}
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
		pick := len(pts) - 1
		var acc float64
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

// seedRandom is fixed-seed weighted random sampling — the deliberately weak
// baseline that isolates how much the smarter seedings buy.
func seedRandom(pts []WPoint, n int) []Vec3 {
	rng := rand.New(rand.NewSource(1))
	cents := make([]Vec3, 0, n)
	seen := map[int]bool{}
	for len(cents) < n && len(seen) < len(pts) {
		i := rng.Intn(len(pts))
		if seen[i] {
			continue
		}
		seen[i] = true
		cents = append(cents, pts[i].P)
	}
	return cents
}
