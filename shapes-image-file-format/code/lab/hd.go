package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// This file runs the study at native resolution, where the earlier rounds ran on a 512x288 downscale.
// Two questions it answers that the small eval could not:
//
//  1. What does the shape coder cost at *lossless*, the one operating point where WebP is bit-exact and there is nothing to trade?
//  2. Given WebP-lossless's byte budget, what does the shape coder actually produce, seen at 1:1?
//
// Resolution is not a free variable here. Boundary length scales with the linear dimension and area with its square, so a 16x pixel count is only a 4x boundary cost — the isoperimetric term that sinks the shape coder at 512x288 is weaker at 4K. Whether that is enough to change the verdict is exactly what this measures.

// pairKey is one directed adjacency observation, accumulated by sorting rather than by a per-region map.
// At 4K the partition can have millions of regions, and a map per region costs over a gigabyte before any counting starts; sorting 2*crackLen fixed-size records is both smaller and faster.
type pairKey struct{ r, nb int32 }

// bestEarlierNeighbour returns, for every region, the lower-numbered adjacent region sharing the longest boundary with it, or -1 when the region touches no earlier one.
// Region ids come from labels(), which numbers them in raster first-appearance order, so "lower id" is
// exactly "already decoded" — the same predictor availability rule colorBytes uses.
func bestEarlierNeighbour(lab []int32, n, w, h int) []int32 {
	pairs := make([]pairKey, 0, 2*w*h/8)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				pairs = append(pairs, pairKey{lab[p], lab[p+1]}, pairKey{lab[p+1], lab[p]})
			}
			if y < h-1 && lab[p] != lab[p+w] {
				pairs = append(pairs, pairKey{lab[p], lab[p+w]}, pairKey{lab[p+w], lab[p]})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].r != pairs[j].r {
			return pairs[i].r < pairs[j].r
		}
		return pairs[i].nb < pairs[j].nb
	})
	best := make([]int32, n)
	bestLen := make([]int32, n)
	for i := range best {
		best[i] = -1
	}
	for i := 0; i < len(pairs); {
		j := i
		for j < len(pairs) && pairs[j] == pairs[i] {
			j++
		}
		if r, nb, ln := pairs[i].r, pairs[i].nb, int32(j-i); nb < r && ln > bestLen[r] {
			best[r], bestLen[r] = nb, ln
		}
		i = j
	}
	return best
}

// colorBytesLean is colorBytes with the adjacency step replaced by bestEarlierNeighbour.
// The coding model — residual against the predictor, adaptive per-channel symbol counts over a 9-bit alphabet — is byte-for-byte the same, so numbers from this function are comparable with every earlier round.
func colorBytesLean(lab []int32, cols [][3]float64, w, h int) float64 {
	n := len(cols)
	best := bestEarlierNeighbour(lab, n, w, h)
	counts := make([][]uint32, 3)
	totals := make([]uint32, 3)
	for c := range counts {
		counts[c] = make([]uint32, 512)
	}
	bits := 0.0
	for r := 0; r < n; r++ {
		pred := [3]float64{128, 128, 128}
		if best[r] >= 0 {
			pred = cols[best[r]]
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

// exactPartition labels every 4-connected run of identical pixels, which is the finest partition a region coder could ever have to transmit and the only one that reconstructs the image exactly.
func exactPartition(im *Img) ([]int32, [][3]float64, int) {
	npix := im.W * im.H
	raw := make([]int32, npix)
	seen := make(map[uint32]int32, 1<<20)
	for p := 0; p < npix; p++ {
		k := uint32(im.P[p*3])<<16 | uint32(im.P[p*3+1])<<8 | uint32(im.P[p*3+2])
		id, ok := seen[k]
		if !ok {
			id = int32(len(seen))
			seen[k] = id
		}
		raw[p] = id
	}
	ncol := len(seen)
	lab := relabelComponents(raw, im.W, im.H)
	n := 0
	for _, l := range lab {
		if int(l)+1 > n {
			n = int(l) + 1
		}
	}
	cols := make([][3]float64, n)
	done := make([]bool, n)
	for p := 0; p < npix; p++ {
		if l := lab[p]; !done[l] {
			done[l] = true
			cols[l] = [3]float64{im.P[p*3], im.P[p*3+1], im.P[p*3+2]}
		}
	}
	return lab, cols, ncol
}

// hd prices the shape coder at native resolution: first exactly (lossless), then across the merge scale-space, writing a render at every mark so the results can be inspected at 1:1.
func hd(path, outDir string) {
	must(os.MkdirAll(outDir, 0o755))
	im := load(path)
	npix := im.W * im.H
	t0 := time.Now()

	// ---- lossless: the exact partition, priced with this study's own coder -------------------
	lab, cols, ncol := exactPartition(im)
	n := len(cols)
	cl := crackLen(lab, im.W, im.H)
	wallB := caeBytes(lab, im.W, im.H)
	// colorBytes2, not colorBytesLean: the scale-space below prices every lossy rung with colorBytes2, and report 08 states colorBytes2 as the method for the whole table. Pricing this one row with the single-longest-neighbour predictor instead made the lossless figure 4.45% worse than the coder the rest of the table uses, which is the study comparing itself against a weaker version of itself.
	colB := colorBytes2(lab, cols, im.W, im.H)
	fmt.Printf("# lossless, exact region partition of %s (%dx%d, %d px)\n", path, im.W, im.H, npix)
	fmt.Printf("distinct_colours %d\n", ncol)
	fmt.Printf("regions %d\n", n)
	fmt.Printf("px_per_region %.3f\n", float64(npix)/float64(n))
	fmt.Printf("crack_edges %d\n", cl)
	fmt.Printf("lossless_walls_B %.0f\n", wallB)
	fmt.Printf("lossless_walls_bits_per_edge %.4f\n", wallB*8/float64(cl))
	fmt.Printf("lossless_colours_B %.0f\n", colB)
	fmt.Printf("lossless_total_B %.0f\n", wallB+colB)
	fmt.Printf("lossless_bpp %.4f\n", (wallB+colB)*8/float64(npix))
	fmt.Printf("# lossless stage took %s\n", time.Since(t0).Round(time.Second))
	lab, cols = nil, nil

	// ---- the scale-space, with a render at every mark ---------------------------------------
	// Pricing here is deliberately identical to frontier(): the rate-distortion merge, six sweeps of Ising wall relaxation, the cheaper of the two wall coders, and colorBytes2 for the colours.
	// Anything less measures a weaker coder than the one the earlier rounds published, and a resolution comparison against a moving coder would mean nothing.
	marks := hdMarks(npix)
	mi := 0
	fmt.Printf("#\n# scale-space: runRD + relax6, walls = min(CAE, contour), colours = colorBytes2\n")
	fmt.Printf("%-9s %10s %8s %10s %10s %10s %8s\n", "regions", "crack", "psnr", "wallB", "colB", "totalB", "bpp")

	t1 := time.Now()
	m := newMerger(im)
	fmt.Fprintf(os.Stderr, "merger built in %s\n", time.Since(t1).Round(time.Second))
	// The merge starts at one region per pixel and decrements by one, so marks at or above the pixel count can never be reached and would otherwise stall the index on the first entry forever.
	for mi < len(marks) && marks[mi] >= m.nreg {
		mi++
	}
	m.runRD(marks[len(marks)-1], func(mm *merger, lambda float64) {
		for mi < len(marks) && mm.nreg == marks[mi] {
			// Relax a copy: the merger has to carry on from the unrelaxed partition to reach the next mark.
			base := make([]int32, len(mm.parent))
			for i := range base {
				base[i] = mm.find(int32(i))
			}
			lab := relabelComponents(base, im.W, im.H)
			nreg := 0
			for _, l := range lab {
				if int(l)+1 > nreg {
					nreg = int(l) + 1
				}
			}
			lab = relax(im, lab, nreg, lambda*bitsPerEdge, 6)
			nr, ps, bb, bc, rec := priceSeg(im, lab)
			cl := crackLen(lab, im.W, im.H)
			ct, _, _, _ := contourBytes(lab, im.W, im.H)
			wall := math.Min(bb, ct)
			fmt.Printf("%-9d %10d %8.2f %10.0f %10.0f %10.0f %8.4f\n",
				nr, cl, ps, wall, bc, wall+bc, (wall+bc)*8/float64(npix))
			os.Stdout.Sync()
			rec.writePNG(fmt.Sprintf("%s/hd_%08d.png", outDir, nr))
			fmt.Fprintf(os.Stderr, "mark %d (relaxed to %d) done at %s\n", mm.nreg, nr, time.Since(t0).Round(time.Second))
			mi++
		}
	})
	fmt.Printf("# total %s\n", time.Since(t0).Round(time.Second))
}

// hdMarks is a geometric ladder of region counts, so one mark set covers every rung of the resolution ladder without hand-tuning per image.
// The exact values do not matter: the comparison interpolates both curves onto a common PSNR, so only coverage does.
func hdMarks(npix int) []int {
	var out []int
	last := -1
	for v := float64(npix) / 2.5; v >= 128; v /= 1.7 {
		if n := int(v); n != last {
			out = append(out, n)
			last = n
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

// hdcheck asserts colorBytesLean is a drop-in for colorBytes.
// The lean version exists only to survive millions of regions; if it ever disagrees with the original the
// 4K numbers stop being comparable with the 512x288 rounds, which is the whole point of running them.
func hdcheck(path string) {
	im := load(path)
	lab, cols, _ := exactPartition(im)
	a := colorBytes(lab, cols, im.W, im.H)
	b := colorBytesLean(lab, cols, im.W, im.H)
	fmt.Printf("exact partition: %d regions\n  colorBytes     %.4f B\n  colorBytesLean %.4f B\n", len(cols), a, b)
	if d := math.Abs(a - b); d > 1e-6 {
		fmt.Fprintf(os.Stderr, "MISMATCH: %.6f B apart\n", d)
		os.Exit(1)
	}
	// And again on a coarse partition, where regions have many neighbours and ties matter.
	q := quantize(path, 8)
	raw := make([]int32, len(q.Idx))
	for i, v := range q.Idx {
		raw[i] = int32(v)
	}
	l2 := relabelComponents(raw, q.W, q.H)
	n2 := 0
	for _, v := range l2 {
		if int(v)+1 > n2 {
			n2 = int(v) + 1
		}
	}
	c2 := make([][3]float64, n2)
	for i, v := range l2 {
		c2[v] = q.Pal[q.Idx[i]]
	}
	a2 := colorBytes(l2, c2, q.W, q.H)
	b2 := colorBytesLean(l2, c2, q.W, q.H)
	fmt.Printf("coarse partition: %d regions\n  colorBytes     %.4f B\n  colorBytesLean %.4f B\n", n2, a2, b2)
	if d := math.Abs(a2 - b2); d > 1e-6 {
		fmt.Fprintf(os.Stderr, "MISMATCH: %.6f B apart\n", d)
		os.Exit(1)
	}
	fmt.Println("OK: lean colour coder is identical to the original on both partitions")
}
