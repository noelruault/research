package main

import (
	"math"
	"math/rand"
	"testing"
)

// TestP4WallRoundTrip is the check that fails if the container's coder or its schedule breaks.
// It codes a synthetic partition's crack planes with the real interleaved coder, decodes them back with nothing but the chunk, and asserts three things:
// the planes come back identical, the rebuilt partition carries the same region ids the colour stream is written against, and the coded length is the cross-entropy plus a terminator rather than plus a percentage.
// No image, no brotli, no external files, so it runs in milliseconds.
func TestP4WallRoundTrip(t *testing.T) {
	const w, h = 137, 89 // deliberately not a round number, to catch edge handling at the last row and column
	rng := rand.New(rand.NewSource(1))
	lab := make([]int32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Blocky, with a few seams jittered, so the partition has real contours and junctions rather than a grid.
			lab[y*w+x] = int32((x+rng.Intn(3))/11*8 + (y+rng.Intn(3))/9)
		}
	}
	lab = relabelComponents(lab, w, h)
	n := 0
	for _, l := range lab {
		if int(l)+1 > n {
			n = int(l) + 1
		}
	}

	V, Hz := crackPlanes(lab, w, h)
	xeV, xeH := priceVariant(p4Variant(), V, Hz, w, h)
	chunk := p4EncodeWalls(V, Hz, w, h)

	dV, dHz := p4DecodeWalls(chunk, w, h)
	for i := range V {
		if V[i] != dV[i] || Hz[i] != dHz[i] {
			t.Fatalf("crack planes differ at %d: V %d/%d, Hz %d/%d", i, V[i], dV[i], Hz[i], dHz[i])
		}
	}

	dLab, dN := p4Label(dV, dHz, w, h)
	if dN != n {
		t.Fatalf("rebuilt %d regions, encoded %d", dN, n)
	}
	for i := range lab {
		if lab[i] != dLab[i] {
			t.Fatalf("region id at %d is %d, expected %d — the colour stream would be read against the wrong region", i, dLab[i], lab[i])
		}
	}

	// The whole point of P4 is that the coder costs a terminator, not a percentage.
	// 32 B is loose enough never to flake and tight enough to catch p4Split drifting away from binModel's probabilities.
	if over := float64(len(chunk)) - (xeV + xeH); over < 0 || over > 32 {
		t.Fatalf("coded %d B against a %.2f B cross-entropy: %+.2f B of coder overhead", len(chunk), xeV+xeH, over)
	}
}

// TestWallxRefRepricesCae pins the cross-check wallx and wallxExact rely on: one variant in this file must reproduce potts.go's caeBytes exactly.
// Report 20 moved which one, and the stale index left both verbs exiting 1 before they printed a row.
func TestWallxRefRepricesCae(t *testing.T) {
	const w, h = 61, 47
	lab := make([]int32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			lab[y*w+x] = int32(x/7*8 + y/5)
		}
	}
	lab = relabelComponents(lab, w, h)
	rows := wallxRow(lab, w, h)
	if got := variants()[wallxRef].name; got != "baseFix" {
		t.Fatalf("wallxRef points at %q; it must point at the variant caeBytes codes", got)
	}
	if d := math.Abs(rows[wallxRef].bV + rows[wallxRef].bH - caeBytes(lab, w, h)); d > 1e-6 {
		t.Fatalf("%s disagrees with caeBytes by %.6f B", variants()[wallxRef].name, d)
	}
}
