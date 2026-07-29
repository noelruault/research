package main

// turnload re-prices partitions dumped by turnprice, so a variant can be checked in seconds instead of re-walking the merge.

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
)

func turnload(dir string) {
	paths, err := filepath.Glob(dir + "/lab_*.bin")
	must(err)
	sort.Strings(paths)
	fmt.Printf("%-9s %8s %8s %10s %10s %8s %9s %9s\n",
		"regions", "loops", "loopB", "vertB", "turnB", "total", "turns", "b/turn")
	for _, p := range paths {
		lab, w, h := loadLabels(p)
		vb, tb, evs, _, ok := contourReplay(lab, w, h)
		nreg := 0
		for _, l := range lab {
			if int(l)+1 > nreg {
				nreg = int(l) + 1
			}
		}
		// What the loop starts already cost, and what a decoder would still need: the number of loops.
		loopB := float64(lastLoops) * (math.Log2(float64((w+1)*(h+1))) + 2) / 8
		fmt.Printf("%-9d %8d %8.0f %10.0f %10.0f %8.0f %9d %9.4f  reconstruct=%v\n",
			nreg, lastLoops, loopB, vb/8, tb/8, (vb+tb)/8, len(evs), tb/float64(len(evs)), ok)
	}
}
