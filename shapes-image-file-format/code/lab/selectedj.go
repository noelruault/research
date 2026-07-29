package main

// selJTaps is the junction-bitmap template produced by `labx csel renders4k/hd_00006417.png 20`:
// greedy forward selection on top of the ten taps of contour.go:100-102, at 3840x2160 and 6,417 regions.
//
// 6,417 is the densest partition at which the contour coder is still the chosen wall coder (contour 93,577 B against CAE 96,087 B), so it is the operating point with the most junctions -- the best case for widening, which is what makes a negative here worth something.
//
// Frozen so it can be priced at every other operating point and resolution without reselecting.
// The first ten entries are baseJTaps unchanged, so any prefix of length >= 10 is a strict superset of the published coder.
var selJTaps = []jtap{
	{-1, 0}, {-2, 0}, {0, -1}, {-1, -1}, {1, -1},
	{2, -1}, {0, -2}, {-1, -2}, {1, -2}, {-3, 0},
	{0, -3}, {0, -4}, {-4, 0}, {-2, -1}, {1, -3},
	{-6, 0}, {2, -3}, {3, -1}, {-3, -2}, {2, -2},
}
