package main

import (
	"fmt"
	"math"
)

// On flat art the partition is not an approximation, it is the picture.
// This prices an exact, lossless region coder: the true connected components of the image, their walls coded as before, and their colours coded as palette indices predicted from the already-decoded neighbours.
// It is the one regime where explicit geometry has a structural argument, so it deserves an exact number rather than an extrapolation.

// paletteColorBytes codes each region's colour as a palette index predicted from the already-decoded neighbour with the longest shared wall.
// Adjacent regions differ by construction, so the model is conditioned on that neighbour's index, which is what makes it cheap on flat art.
func paletteColorBytes(lab []int32, cols [][3]float64, w, h int) (float64, int) {
	// Build the palette.
	key := func(c [3]float64) int { return int(c[0])<<16 | int(c[1])<<8 | int(c[2]) }
	idOf := map[int]int{}
	for _, c := range cols {
		if _, ok := idOf[key(c)]; !ok {
			idOf[key(c)] = len(idOf)
		}
	}
	np := len(idOf)
	if np > 512 {
		return math.Inf(1), np
	}
	pidx := make([]int, len(cols))
	for i, c := range cols {
		pidx[i] = idOf[key(c)]
	}
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
	counts := make([]uint32, (np+1)*np)
	totals := make([]uint32, np+1)
	bits := float64(np) * 24 // the palette itself is transmitted once
	for r := 0; r < n; r++ {
		best, bestLen := int32(-1), 0
		for nb, ln := range share[r] {
			if int(nb) < r && ln > bestLen {
				best, bestLen = nb, ln
			}
		}
		ctx := np // no predictor yet
		if best >= 0 {
			ctx = pidx[best]
		}
		s := pidx[r]
		bits += -math.Log2(float64(counts[ctx*np+s]+1) / float64(totals[ctx]+uint32(np)))
		counts[ctx*np+s]++
		totals[ctx]++
	}
	return bits / 8, np
}

// lossless prices an exact region coder against the lossless raster wall.
func lossless(path string) {
	im := load(path)
	npix := im.W * im.H
	// Exact connected components of equal-colour pixels.
	raw := make([]int32, npix)
	seen := map[int]int32{}
	for p := 0; p < npix; p++ {
		k := int(im.P[p*3])<<16 | int(im.P[p*3+1])<<8 | int(im.P[p*3+2])
		id, ok := seen[k]
		if !ok {
			id = int32(len(seen))
			seen[k] = id
		}
		raw[p] = id
	}
	lab := relabelComponents(raw, im.W, im.H)
	n, ps, bb, _, rec := priceSeg(im, lab)
	cl := crackLen(lab, im.W, im.H)
	ct, vb, tb, steps := contourBytes(lab, im.W, im.H)

	cols := make([][3]float64, n)
	cnt := make([]float64, n)
	for p, l := range lab {
		cnt[l]++
		for c := 0; c < 3; c++ {
			cols[l][c] += im.P[p*3+c]
		}
	}
	for k := range cols {
		for c := 0; c < 3; c++ {
			cols[k][c] = math.Round(cols[k][c] / cnt[k])
		}
	}
	bp, np := paletteColorBytes(lab, cols, im.W, im.H)

	fmt.Printf("%s: %dx%d, %d distinct colours, %d regions, %d crack edges\n", path, im.W, im.H, len(seen), n, cl)
	fmt.Printf("  reconstruction PSNR (must be lossless): %.2f dB\n", ps)
	fmt.Printf("  walls, CAE crack planes    : %8.0f B  (%.3f bits/edge)\n", bb, bb*8/float64(cl))
	fmt.Printf("  walls, contour turn coder  : %8.0f B  (junctions %.0f B + turns %.0f B over %d steps = %.3f bits/step)\n",
		ct, vb, tb, steps, tb*8/float64(steps))
	fmt.Printf("  colours, palette index     : %8.0f B  (%d palette entries, %.2f bits/region)\n",
		bp, np, bp*8/float64(n))
	best := math.Min(bb, ct) + bp
	fmt.Printf("  TOTAL exact region coder   : %8.0f B  (%.3f bpp)\n", best, best*8/float64(npix))
	_ = rec
	// The raster wall for the identical pixels: the repo's own champion, an adaptive order-k context coder over the index map.
	if len(seen) <= 256 {
		idx := make([]uint8, npix)
		for p := 0; p < npix; p++ {
			k := int(im.P[p*3])<<16 | int(im.P[p*3+1])<<8 | int(im.P[p*3+2])
			idx[p] = uint8(seen[k])
		}
		nc := len(seen)
		fmt.Printf("  raster wall, order-1 ctx   : %8.0f B\n", adaptiveBytes(idx, im.W, im.H, 1, nc))
		fmt.Printf("  raster wall, order-2 ctx   : %8.0f B   <-- the repo's own best encoder\n", adaptiveBytes(idx, im.W, im.H, 2, nc))
		fmt.Printf("  raster wall, order-3 ctx   : %8.0f B\n", adaptiveBytes(idx, im.W, im.H, 3, nc))
	}
}
