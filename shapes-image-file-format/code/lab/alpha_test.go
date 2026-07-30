package main

import "testing"

// TestAlphaOpaqueIsInert is the guarantee every published number in this study rests on:
// an image with no alpha must take exactly the path it took before alpha existed.
// The merge carries alpha as a fourth SSE channel, and a CONSTANT fourth channel contributes a zero term to every dSSE — so the partition cannot move. This asserts that algebraically rather than trusting it: same geometry, one arm with no alpha and one arm fully opaque, identical dSSE.
func TestAlphaOpaqueIsInert(t *testing.T) {
	const w, h = 23, 17
	mk := func(withAlpha bool) *Img {
		im := &Img{W: w, H: h, P: make([]float64, w*h*3)}
		if withAlpha {
			im.A = make([]float64, w*h)
		}
		for p := 0; p < w*h; p++ {
			x, y := p%w, p/w
			im.P[p*3] = float64((x * 11) % 256)
			im.P[p*3+1] = float64((y * 7) % 256)
			im.P[p*3+2] = float64((x*y)%256) / 2
			if withAlpha {
				im.A[p] = 255 // fully opaque: constant, so it must be inert
			}
		}
		return im
	}
	bare, opaque := newMerger(mk(false)), newMerger(mk(true))
	for a := int32(0); a < w*h; a++ {
		for b := range bare.adj[a] {
			if x, y := bare.dSSE(a, b), opaque.dSSE(a, b); x != y {
				t.Fatalf("dSSE(%d,%d) moved when a constant alpha was added: %v vs %v", a, b, x, y)
			}
		}
	}
}

// TestAlphaSeparatesIdenticalColours is report A1's failure, reduced to its smallest form.
// Two halves the SAME colour, one transparent and one opaque. Before alpha reached the merge this was a single region and the silhouette was gone; the merge could not see any difference because there was none left to see. Now the transparency IS the difference.
func TestAlphaSeparatesIdenticalColours(t *testing.T) {
	const w, h = 8, 4
	im := &Img{W: w, H: h, P: make([]float64, w*h*3), A: make([]float64, w*h)}
	for p := 0; p < w*h; p++ {
		im.P[p*3], im.P[p*3+1], im.P[p*3+2] = 17, 17, 17 // identical colour everywhere
		if p%w < w/2 {
			im.A[p] = 0 // left half transparent
		} else {
			im.A[p] = 255 // right half opaque
		}
	}
	// The exact partition must not put a transparent pixel and an opaque one in one region.
	lab, _, _ := exactPartition(im)
	for p := 0; p < w*h; p++ {
		if p%w == w/2-1 && lab[p] == lab[p+1] {
			t.Fatalf("pixel %d (alpha 0) and %d (alpha 255) share region %d despite identical colour", p, p+1, lab[p])
		}
	}
	// And the merge must price joining them as a real cost, not free.
	m := newMerger(im)
	across := m.dSSE(int32(w/2-1), int32(w/2))
	if across <= 0 {
		t.Fatalf("merging across a transparency edge costs %v; it must be positive", across)
	}
	// Same colour, same alpha: genuinely free, so the alpha term is not just a constant penalty.
	along := m.dSSE(int32(w/2), int32(w/2+1))
	if along != 0 {
		t.Fatalf("merging two identical opaque pixels costs %v; it must be 0", along)
	}
}

// TestRegionAlphasNilWithoutAlpha pins the convention the container's mode field depends on:
// no alpha in the source means no alpha plane, which means SHPC v2 mode 0 and no alpha chunk.
func TestRegionAlphasNilWithoutAlpha(t *testing.T) {
	im := &Img{W: 4, H: 4, P: make([]float64, 4*4*3)}
	if got := regionAlphas(im, make([]int32, 16), 1); got != nil {
		t.Fatalf("regionAlphas on an image with no alpha returned %v, want nil", got)
	}
}
