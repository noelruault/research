package main

import (
	"fmt"
	"math"
	"sort"
)

// scaling reports the statistical mechanics of the partition at the operating point: how region sizes are distributed, how the wall length is shared between big and small regions, and how compact the cells are relative to the isoperimetric ideal.
// Percolation and fracture both produce power-law cluster sizes, and if natural-image regions do too, the small-region tail sets an irreducible floor on the wall length any shape coder must pay for.
func scaling(im *Img, lab []int32, n, cl int, bBound, bCol float64) {
	npix := im.W * im.H
	size := make([]int, n)
	per := make([]int, n) // perimeter of each region in crack edges, image frame excluded
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			p := y*im.W + x
			size[lab[p]]++
			if x < im.W-1 && lab[p] != lab[p+1] {
				per[lab[p]]++
				per[lab[p+1]]++
			}
			if y < im.H-1 && lab[p] != lab[p+im.W] {
				per[lab[p]]++
				per[lab[p+im.W]]++
			}
		}
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return size[idx[a]] > size[idx[b]] })

	fmt.Printf("\n--- scaling at the operating point (%d regions, %d crack edges) ---\n", n, cl)
	// Maximum-likelihood Pareto exponent over the tail (Clauset-Shalizi-Newman).
	for _, xmin := range []float64{2, 4, 8} {
		var s float64
		k := 0
		for _, sz := range size {
			if float64(sz) >= xmin {
				s += math.Log(float64(sz) / xmin)
				k++
			}
		}
		if k > 10 {
			fmt.Printf("Pareto alpha (xmin=%.0f px, n=%d): %.3f\n", xmin, k, 1+float64(k)/s)
		}
	}
	// Where the area and where the wall length live.
	cumA, cumP := 0, 0
	for i, r := range idx {
		cumA += size[r]
		cumP += per[r]
		if i+1 == n/10 || i+1 == n/2 {
			fmt.Printf("top %3d%% of regions by size: %5.1f%% of area, %5.1f%% of total perimeter\n",
				(i+1)*100/n, float64(cumA)*100/float64(npix), float64(cumP)*50/float64(cl))
		}
	}
	small, smallPer := 0, 0
	for r := 0; r < n; r++ {
		if size[r] <= 16 {
			small++
			smallPer += per[r]
		}
	}
	fmt.Printf("regions <=16 px: %d of %d (%.0f%%), holding %.1f%% of the wall length\n",
		small, n, float64(small)*100/float64(n), float64(smallPer)*50/float64(cl))
	// Isoperimetric compactness: how much longer the walls are than circular cells of the same areas would be.
	ideal := 0.0
	for r := 0; r < n; r++ {
		ideal += 4 * math.Sqrt(float64(size[r])) // a square of that area, the lattice-optimal shape
	}
	fmt.Printf("wall length vs lattice-optimal compact cells: %.0f vs %.0f  (excess factor %.2f)\n",
		float64(cl)*2, ideal, float64(cl)*2/ideal)
	// What the bits would be if either half of the description were free.
	fmt.Printf("if walls were FREE : %.0f B ; if colours were FREE : %.0f B ; actual %.0f B\n",
		bCol, bBound, bBound+bCol)
}
