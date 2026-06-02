package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
)

// pca.go — divisive quantization on weighted points in any space (piece P3 +
// P1). Generalizes median cut two ways the research flagged:
//   - select the box to split by largest weighted variance (SSE), which
//     targets quantization error directly (Wu's idea), not just population;
//   - split along the box's PRINCIPAL axis (PCA) instead of a coordinate
//     axis, so the cut follows the real shape of the color cloud.
// Space lets us run the same algorithm in RGB or OKLab to test whether a
// perceptual space helps divisive cuts (research warned it often does not).

type Divisive struct {
	Space Space
	PCA   bool // true: principal axis; false: longest coordinate axis
}

func (d Divisive) Name() string {
	ax := "axisaligned"
	if d.PCA {
		ax = "pca"
	}
	return fmt.Sprintf("divisive/%s/%s", ax, d.Space.Name())
}

type pbox struct {
	pts    []WPoint
	weight float64
	mean   Vec3
	sse    float64
}

func newPBox(pts []WPoint) pbox {
	var w float64
	var sum Vec3
	for _, p := range pts {
		w += p.W
		sum = sum.add(p.P.scale(p.W))
	}
	b := pbox{pts: pts, weight: w}
	if w > 0 {
		b.mean = sum.scale(1.0 / w)
	}
	for _, p := range pts {
		b.sse += p.W * p.P.dist2(b.mean)
	}
	return b
}

func (d Divisive) Quantize(img image.Image, n int) color.Palette {
	h := histogram(img)
	pts := weightedPoints(h, d.Space)
	if len(pts) <= n {
		pal := make(color.Palette, len(h))
		for i, c := range h {
			pal[i] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
		}
		return pal
	}

	boxes := []pbox{newPBox(pts)}
	for len(boxes) < n {
		// Split the box with the largest SSE that still has >1 point.
		best := -1
		for i := range boxes {
			if len(boxes[i].pts) < 2 {
				continue
			}
			if best < 0 || boxes[i].sse > boxes[best].sse {
				best = i
			}
		}
		if best < 0 {
			break
		}
		a, b := d.splitBox(boxes[best])
		if len(a.pts) == 0 || len(b.pts) == 0 {
			// Degenerate (all points identical on the axis): mark unsplittable.
			boxes[best].pts = boxes[best].pts[:1]
			continue
		}
		boxes[best] = a
		boxes = append(boxes, b)
	}

	pal := make(color.Palette, len(boxes))
	for i, bx := range boxes {
		pal[i] = d.Space.ToRGBA(bx.mean)
	}
	return pal
}

func (d Divisive) splitBox(b pbox) (pbox, pbox) {
	var axis Vec3
	if d.PCA {
		axis = principalAxis(b.pts, b.mean)
	} else {
		axis = longestCoordAxis(b.pts)
	}
	proj := func(p Vec3) float64 {
		return (p[0]-b.mean[0])*axis[0] + (p[1]-b.mean[1])*axis[1] + (p[2]-b.mean[2])*axis[2]
	}
	sort.Slice(b.pts, func(i, j int) bool { return proj(b.pts[i].P) < proj(b.pts[j].P) })
	half := b.weight / 2
	acc, cut := 0.0, 1
	for i, p := range b.pts {
		acc += p.W
		if acc >= half {
			cut = i + 1
			break
		}
	}
	if cut < 1 {
		cut = 1
	}
	if cut >= len(b.pts) {
		cut = len(b.pts) - 1
	}
	return newPBox(b.pts[:cut]), newPBox(b.pts[cut:])
}

// principalAxis returns the top eigenvector of the weighted covariance via
// power iteration — the direction of greatest spread in the box.
func principalAxis(pts []WPoint, mean Vec3) Vec3 {
	var cxx, cxy, cxz, cyy, cyz, czz, w float64
	for _, p := range pts {
		dx, dy, dz := p.P[0]-mean[0], p.P[1]-mean[1], p.P[2]-mean[2]
		cxx += p.W * dx * dx
		cxy += p.W * dx * dy
		cxz += p.W * dx * dz
		cyy += p.W * dy * dy
		cyz += p.W * dy * dz
		czz += p.W * dz * dz
		w += p.W
	}
	if w > 0 {
		cxx, cxy, cxz, cyy, cyz, czz = cxx/w, cxy/w, cxz/w, cyy/w, cyz/w, czz/w
	}
	v := Vec3{1, 1, 1}
	for it := 0; it < 32; it++ {
		nv := Vec3{
			cxx*v[0] + cxy*v[1] + cxz*v[2],
			cxy*v[0] + cyy*v[1] + cyz*v[2],
			cxz*v[0] + cyz*v[1] + czz*v[2],
		}
		norm := math.Sqrt(nv[0]*nv[0] + nv[1]*nv[1] + nv[2]*nv[2])
		if norm == 0 {
			return Vec3{1, 0, 0}
		}
		v = nv.scale(1.0 / norm)
	}
	return v
}

func longestCoordAxis(pts []WPoint) Vec3 {
	mn := Vec3{math.Inf(1), math.Inf(1), math.Inf(1)}
	mx := Vec3{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, p := range pts {
		for k := 0; k < 3; k++ {
			mn[k] = math.Min(mn[k], p.P[k])
			mx[k] = math.Max(mx[k], p.P[k])
		}
	}
	axis := 0
	best := mx[0] - mn[0]
	for k := 1; k < 3; k++ {
		if d := mx[k] - mn[k]; d > best {
			best, axis = d, k
		}
	}
	var v Vec3
	v[axis] = 1
	return v
}
