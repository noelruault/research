package main

import "math"

// This file ports WebP's shrink levers into our domain, one at a time, so the bake-off can watch the number fall as geometry is dropped and prediction is added. The eval is the same n=16 index grid; every method here is lossless.

// rleBytes is "shapes minus redundant geometry": each horizontal run is a 1-tall rect, but its x is implicit in scan order, so we store only (run length, color) per row instead of (x, y, w, h).
// It is the honest floor of the rect idea once the coordinates the raster already implies are removed.
func rleBytes(grid [][]int, w, h int) []byte {
	var out []byte
	putv := func(v int) { // little varint
		for v >= 0x80 {
			out = append(out, byte(v)|0x80)
			v >>= 7
		}
		out = append(out, byte(v))
	}
	for y := 0; y < h; y++ {
		x := 0
		for x < w {
			c := grid[x][y]
			run := 1
			for x+run < w && grid[x+run][y] == c {
				run++
			}
			putv(run)
			out = append(out, byte(c))
			x += run
		}
	}
	return out
}

// adaptiveEntropyBytes returns the exact size an adaptive arithmetic coder achieves on the index map (its cross-entropy under an add-one model), which is how such coders are analyzed and is within ~1 byte of a real bitstream.
// No geometry at all: it codes the index of each pixel given a neighbor context, exactly WebP's "predict then entropy-code" idea.
//
//	order 0: no context (raw symbol entropy)
//	order 1: context = pixel to the left order 2: context = (left, up) — 2D prediction, the WebP-like case
func adaptiveEntropyBytes(grid [][]int, w, h, order, ncol int) float64 {
	nctx := 1
	switch order {
	case 1:
		nctx = ncol
	case 2:
		nctx = ncol * ncol
	case 3:
		nctx = ncol * ncol * ncol
	}
	counts := make([][]int, nctx)
	totals := make([]int, nctx)
	for i := range counts {
		counts[i] = make([]int, ncol)
	}

	bits := 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			left, up := 0, 0
			if x > 0 {
				left = grid[x-1][y]
			}
			if y > 0 {
				up = grid[x][y-1]
			}
			upleft := 0
			if x > 0 && y > 0 {
				upleft = grid[x-1][y-1]
			}
			ctx := 0
			switch order {
			case 1:
				ctx = left
			case 2:
				ctx = left*ncol + up
			case 3:
				ctx = (left*ncol+up)*ncol + upleft
			}
			s := grid[x][y]
			// Add-one (Laplace) probability, the code length an adaptive coder pays.
			p := float64(counts[ctx][s]+1) / float64(totals[ctx]+ncol)
			bits += -math.Log2(p)
			counts[ctx][s]++
			totals[ctx]++
		}
	}
	return bits / 8
}
