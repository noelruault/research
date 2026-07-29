package main

import (
	"fmt"
	"math"
)

// This file tests the "encode the forces, not the result" idea in its strongest form: a centroidal Voronoi tessellation (foam at equilibrium / Lloyd relaxation under a density field).
// The decoder is given seeds and regenerates every cell boundary itself, so the geometry cost collapses to the cost of a point set instead of the cost of a boundary.
// Variant A transmits the seeds. Variant B transmits nothing at all for geometry: both sides derive the density field from a coarse image they already share and run the identical relaxation, so the partition emerges rather than being sent.

// jfa computes the nearest-seed assignment for every pixel by jump flooding, which is O(P log W) instead of O(P*N).
func jfa(w, h int, sx, sy []int32) []int32 {
	n := w * h
	site := make([]int32, n)
	for i := range site {
		site[i] = -1
	}
	for k := range sx {
		site[int(sy[k])*w+int(sx[k])] = int32(k)
	}
	d2 := func(p int, s int32) float64 {
		dx := float64(p%w) - float64(sx[s])
		dy := float64(p/w) - float64(sy[s])
		return dx*dx + dy*dy
	}
	step := 1
	for step < w || step < h {
		step *= 2
	}
	next := make([]int32, n)
	for ; step >= 1; step /= 2 {
		copy(next, site)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				p := y*w + x
				for dy := -step; dy <= step; dy += step {
					for dx := -step; dx <= step; dx += step {
						qx, qy := x+dx, y+dy
						if qx < 0 || qy < 0 || qx >= w || qy >= h {
							continue
						}
						s := site[qy*w+qx]
						if s < 0 {
							continue
						}
						if next[p] < 0 || d2(p, s) < d2(p, next[p]) {
							next[p] = s
						}
					}
				}
			}
		}
		copy(site, next)
	}
	return site
}

// gradMag is the density field the relaxation minimises against: cells crowd where the image changes fastest, the same way a foam's small cells sit where the driving field is strongest.
func gradMag(im *Img) []float64 {
	g := make([]float64, im.W*im.H)
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			xp, xm := min(x+1, im.W-1), max(x-1, 0)
			yp, ym := min(y+1, im.H-1), max(y-1, 0)
			var s float64
			for c := 0; c < 3; c++ {
				dx := im.P[(y*im.W+xp)*3+c] - im.P[(y*im.W+xm)*3+c]
				dy := im.P[(yp*im.W+x)*3+c] - im.P[(ym*im.W+x)*3+c]
				s += dx*dx + dy*dy
			}
			g[y*im.W+x] = math.Sqrt(s)
		}
	}
	return g
}

// seedsFromDensity picks n sites by deterministic stratified sampling of the density CDF; identical code runs on both sides in variant B.
func seedsFromDensity(dens []float64, w, h, n int) ([]int32, []int32) {
	tot := 0.0
	for _, d := range dens {
		tot += d + 1e-3
	}
	stepv := tot / float64(n)
	var sx, sy []int32
	acc, nextT := 0.0, stepv*0.5
	for i := 0; i < len(dens) && len(sx) < n; i++ {
		acc += dens[i] + 1e-3
		for acc >= nextT && len(sx) < n {
			sx = append(sx, int32(i%w))
			sy = append(sy, int32(i/w))
			nextT += stepv
		}
	}
	for len(sx) < n {
		sx = append(sx, int32((len(sx)*7919)%w))
		sy = append(sy, int32((len(sy)*7907)%h))
	}
	return sx, sy
}

// lloyd relaxes the seeds to the density-weighted centroids of their cells: the equilibrium condition of a foam.
func lloyd(dens []float64, w, h int, sx, sy []int32, iters int) []int32 {
	site := jfa(w, h, sx, sy)
	for it := 0; it < iters; it++ {
		n := len(sx)
		wx := make([]float64, n)
		wy := make([]float64, n)
		ww := make([]float64, n)
		for p, s := range site {
			d := dens[p] + 1e-3
			wx[s] += d * float64(p%w)
			wy[s] += d * float64(p/w)
			ww[s] += d
		}
		moved := false
		for k := 0; k < n; k++ {
			if ww[k] == 0 {
				continue
			}
			nx := int32(math.Round(wx[k] / ww[k]))
			ny := int32(math.Round(wy[k] / ww[k]))
			if nx != sx[k] || ny != sy[k] {
				moved = true
			}
			sx[k], sy[k] = nx, ny
		}
		// Distinct sites are required for the tessellation to be well defined.
		occ := map[int32]bool{}
		for k := 0; k < n; k++ {
			key := sy[k]*int32(w) + sx[k]
			for occ[key] {
				key++
				if int(key) >= w*h {
					key = 0
				}
			}
			occ[key] = true
			sx[k], sy[k] = key%int32(w), key/int32(w)
		}
		site = jfa(w, h, sx, sy)
		if !moved {
			break
		}
	}
	return site
}

// paintCells replaces every cell by its mean colour, the MSE-optimal constant per cell.
func paintCells(im *Img, site []int32, n int) (*Img, [][3]float64) {
	sum := make([][3]float64, n)
	cnt := make([]float64, n)
	for p, s := range site {
		for c := 0; c < 3; c++ {
			sum[s][c] += im.P[p*3+c]
		}
		cnt[s]++
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
	out := &Img{W: im.W, H: im.H, P: make([]float64, len(im.P))}
	for p, s := range site {
		for c := 0; c < 3; c++ {
			out.P[p*3+c] = cols[s][c]
		}
	}
	return out, cols
}

// seedMapBytes is the real cost of the point set: an adaptive context-coded occupancy bitmap, which is cheaper than log2 C(P,N) wherever the seeds cluster.
func seedMapBytes(w, h int, sx, sy []int32) float64 {
	occ := make([]byte, w*h)
	for k := range sx {
		occ[int(sy[k])*w+int(sx[k])] = 1
	}
	get := func(x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		return int(occ[y*w+x])
	}
	m := make([]binModel, 1024)
	bits := 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ctx := get(x-1, y) | get(x-2, y)<<1 | get(x, y-1)<<2 | get(x-1, y-1)<<3 |
				get(x+1, y-1)<<4 | get(x+2, y-1)<<5 | get(x, y-2)<<6 | get(x-1, y-2)<<7 |
				get(x+1, y-2)<<8 | get(x-3, y)<<9
			bits += m[ctx].cost(int(occ[y*w+x]))
		}
	}
	return bits / 8
}

func combinatorialBits(p, n int) float64 { // log2 C(p,n): the neutral cost of an unordered point set
	b := 0.0
	for i := 0; i < n; i++ {
		b += math.Log2(float64(p-i)) - math.Log2(float64(i+1))
	}
	return b
}

func voronoi(path string) {
	im := load(path)
	npix := im.W * im.H
	dens := gradMag(im)

	fmt.Println("=== A. centroidal Voronoi tessellation, seeds transmitted ===")
	fmt.Printf("%8s %8s %12s %12s %12s %12s %8s\n", "cells", "PSNR", "seed(ctx)", "seed(comb)", "colours", "total", "bpp")
	for _, n := range []int{256, 512, 1024, 2048, 4096, 8192, 16384, 32768} {
		if n >= npix/2 {
			break
		}
		sx, sy := seedsFromDensity(dens, im.W, im.H, n)
		site := lloyd(dens, im.W, im.H, sx, sy, 8)
		rec, cols := paintCells(im, site, n)
		ps := psnrSSE(sseBetween(im, rec), npix)
		bs := seedMapBytes(im.W, im.H, sx, sy)
		bc := combinatorialBits(npix, n) / 8
		bcol := colorBytes(site, cols, im.W, im.H)
		tot := math.Min(bs, bc) + bcol
		fmt.Printf("%8d %8.2f %12s %12s %12s %12s %8.3f\n",
			n, ps, kb(bs), kb(bc), kb(bcol), kb(tot), tot*8/float64(npix))
	}

	fmt.Println("\n=== B. seeds emergent from a shared coarse field (geometry cost = 0) ===")
	fmt.Println("both sides derive the density from the same decoded coarse image and run the identical relaxation")
	fmt.Printf("%8s %6s %8s %12s %12s %8s\n", "cells", "down", "PSNR", "coarse png", "colours", "total*")
	for _, f := range []int{8, 12, 16} {
		coarse := downUp(im, f)
		cdens := gradMag(coarse)
		for _, n := range []int{2048, 4096, 8192} {
			sx, sy := seedsFromDensity(cdens, im.W, im.H, n)
			site := lloyd(cdens, im.W, im.H, sx, sy, 8)
			rec, cols := paintCells(im, site, n)
			ps := psnrSSE(sseBetween(im, rec), npix)
			// Colours predicted from the coarse image the decoder already holds.
			bcol := colorBytesVsRef(site, cols, coarse, len(cols))
			fmt.Printf("%8d %6d %8.2f %12s %12s %8s\n",
				n, f, ps, "see bash", kb(bcol), kb(bcol))
			_ = rec
		}
		coarse.writePNG(fmt.Sprintf("coarse_%d.png", f))
	}
}

// downUp block-averages by f and replicates back, the coarse field both sides share.
func downUp(im *Img, f int) *Img {
	out := &Img{W: im.W, H: im.H, P: make([]float64, len(im.P))}
	for by := 0; by < im.H; by += f {
		for bx := 0; bx < im.W; bx += f {
			var s [3]float64
			c := 0.0
			for y := by; y < min(by+f, im.H); y++ {
				for x := bx; x < min(bx+f, im.W); x++ {
					for ch := 0; ch < 3; ch++ {
						s[ch] += im.P[(y*im.W+x)*3+ch]
					}
					c++
				}
			}
			for y := by; y < min(by+f, im.H); y++ {
				for x := bx; x < min(bx+f, im.W); x++ {
					for ch := 0; ch < 3; ch++ {
						out.P[(y*im.W+x)*3+ch] = math.Round(s[ch] / c)
					}
				}
			}
		}
	}
	return out
}

// colorBytesVsRef codes each cell's colour as a residual against the mean of a reference image over the same cell, which the decoder already has.
func colorBytesVsRef(site []int32, cols [][3]float64, ref *Img, n int) float64 {
	sum := make([][3]float64, n)
	cnt := make([]float64, n)
	for p, s := range site {
		for c := 0; c < 3; c++ {
			sum[s][c] += ref.P[p*3+c]
		}
		cnt[s]++
	}
	counts := make([][]uint32, 3)
	totals := make([]uint32, 3)
	for c := range counts {
		counts[c] = make([]uint32, 512)
	}
	bits := 0.0
	for k := 0; k < n; k++ {
		if cnt[k] == 0 {
			continue
		}
		for c := 0; c < 3; c++ {
			d := int(cols[k][c]-math.Round(sum[k][c]/cnt[k])) + 256
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

// lattice is the null hypothesis for every emergent-geometry idea: put the seeds on a fixed grid so their positions cost literally nothing.
// The Voronoi cells are then squares and the whole scheme collapses to block averaging plus nearest-neighbour upsampling, i.e. subsampling.
func lattice(path string) {
	im := load(path)
	npix := im.W * im.H
	fmt.Printf("%6s %10s %8s %s\n", "factor", "cells", "PSNR", "note")
	for _, f := range []int{2, 3, 4, 6, 8, 12, 16} {
		rec := downUp(im, f)
		cells := ((im.W + f - 1) / f) * ((im.H + f - 1) / f)
		fmt.Printf("%6d %10d %8.2f  wrote lattice_%d.png (coarse %dx%d)\n",
			f, cells, psnrSSE(sseBetween(im, rec), npix), f, (im.W+f-1)/f, (im.H+f-1)/f)
		writeCoarse(im, f, fmt.Sprintf("lattice_%d.png", f))
	}
}

func writeCoarse(im *Img, f int, path string) {
	cw, ch := (im.W+f-1)/f, (im.H+f-1)/f
	out := &Img{W: cw, H: ch, P: make([]float64, cw*ch*3)}
	for by := 0; by < ch; by++ {
		for bx := 0; bx < cw; bx++ {
			var s [3]float64
			c := 0.0
			for y := by * f; y < min((by+1)*f, im.H); y++ {
				for x := bx * f; x < min((bx+1)*f, im.W); x++ {
					for ch2 := 0; ch2 < 3; ch2++ {
						s[ch2] += im.P[(y*im.W+x)*3+ch2]
					}
					c++
				}
			}
			for ch2 := 0; ch2 < 3; ch2++ {
				out.P[(by*cw+bx)*3+ch2] = math.Round(s[ch2] / c)
			}
		}
	}
	out.writePNG(path)
}
