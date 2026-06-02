package main

import (
	"math"
	"testing"
)

// TestCIEDE2000Sharma validates the CIEDE2000 implementation against the
// reference pairs published by Sharma, Wu & Dalal (2005), "The CIEDE2000
// color-difference formula: Implementation notes, supplementary test data,
// and mathematical observations." If these pass, the perceptual numbers the
// harness reports are trustworthy.
func TestCIEDE2000Sharma(t *testing.T) {
	cases := []struct {
		l1, l2 Lab
		want   float64
	}{
		{Lab{50.0000, 2.6772, -79.7751}, Lab{50.0000, 0.0000, -82.7485}, 2.0425},
		{Lab{50.0000, 3.1571, -77.2803}, Lab{50.0000, 0.0000, -82.7485}, 2.8615},
		{Lab{50.0000, 2.8361, -74.0200}, Lab{50.0000, 0.0000, -82.7485}, 3.4412},
		{Lab{50.0000, -1.3802, -84.2814}, Lab{50.0000, 0.0000, -82.7485}, 1.0000},
		{Lab{50.0000, -1.1848, -84.8006}, Lab{50.0000, 0.0000, -82.7485}, 1.0000},
		{Lab{50.0000, -0.9009, -85.5211}, Lab{50.0000, 0.0000, -82.7485}, 1.0000},
		{Lab{50.0000, 0.0000, 0.0000}, Lab{50.0000, -1.0000, 2.0000}, 2.3669},
		{Lab{50.0000, -1.0000, 2.0000}, Lab{50.0000, 0.0000, 0.0000}, 2.3669},
		{Lab{50.0000, 2.4900, -0.0010}, Lab{50.0000, -2.4900, 0.0009}, 7.1792},
		{Lab{60.2574, -34.0099, 36.2677}, Lab{60.4626, -34.1751, 39.4387}, 1.2644},
		{Lab{63.0109, -31.0961, -5.8663}, Lab{62.8187, -29.7946, -4.0864}, 1.2630},
		{Lab{22.7233, 20.0904, -46.6940}, Lab{23.0331, 14.9730, -42.5619}, 2.0373},
	}
	for i, c := range cases {
		got := CIEDE2000(c.l1, c.l2)
		if math.Abs(got-c.want) > 1e-4 {
			t.Errorf("pair %d: got %.4f, want %.4f", i+1, got, c.want)
		}
	}
}
