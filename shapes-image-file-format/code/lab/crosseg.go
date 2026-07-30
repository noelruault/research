package main

import (
	"fmt"
	"os"
	"strconv"
)

// crossegCmd takes the PARTITION from one image and the COLOURS from another.
//
// The idea, and it is native to this format rather than a trick: geometry and colour are separate channels here, so nothing requires them to come from the same pixels.
// Flatten an image to make its classes separable, segment THAT, then fill the resulting regions from the ORIGINAL.
// The delivered picture keeps its real colours and inherits a partition computed where the boundaries were easier to find.
//
// A raster codec cannot do this at all: its "segmentation" is the block grid, and its colours are the same coefficients.
// Here they are two chunks in the file.
//
// The same trick answers the background-removal case: derive the mask on the flattened image, apply it to the original.
//
// usage: crosseg <seg-source.png> <colour-source.png> <regions> <out.png>
func crossegCmd(args []string) {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: lab crosseg <seg-source.png> <colour-source.png> <regions> <out.png>")
		os.Exit(2)
	}
	segIm, colIm := load(args[0]), load(args[1])
	if segIm.W != colIm.W || segIm.H != colIm.H {
		fmt.Fprintf(os.Stderr, "fatal: %dx%d vs %dx%d — the two sources must be the same size\n",
			segIm.W, segIm.H, colIm.W, colIm.H)
		os.Exit(1)
	}
	target, err := strconv.Atoi(args[2])
	must(err)

	// Merge on the segmentation source, capturing the final merge key as the relaxation lambda — which is what hd does, and what keeps the wall straightening at the same operating point.
	m := newMerger(segIm)
	lambda := 0.0
	m.runRD(target, func(_ *merger, k float64) { lambda = k })
	lab, _ := m.labels()
	n := 0
	for _, l := range lab {
		if int(l)+1 > n {
			n = int(l) + 1
		}
	}
	// Relax on the segmentation source too: the walls belong to the partition, not to the colours.
	lab = relax(segIm, lab, n, lambda*bitsPerEdge, 6)

	// Price and paint from the COLOUR source. This is the whole point of the verb.
	nrX, psX, wallX, colX, recX := priceSeg(colIm, lab)
	recX.writePNG(args[3])

	// Baseline: the same target segmented on the colour source directly, so the comparison is like-for-like at matched region count rather than against a different operating point.
	m2 := newMerger(colIm)
	lambda2 := 0.0
	m2.runRD(target, func(_ *merger, k float64) { lambda2 = k })
	lab2, _ := m2.labels()
	n2 := 0
	for _, l := range lab2 {
		if int(l)+1 > n2 {
			n2 = int(l) + 1
		}
	}
	lab2 = relax(colIm, lab2, n2, lambda2*bitsPerEdge, 6)
	nrB, psB, wallB, colB, _ := priceSeg(colIm, lab2)

	fmt.Printf("%-34s %9s %8s %10s %10s %10s\n", "partition from", "regions", "PSNR", "wallB", "colB", "totalB")
	fmt.Printf("%-34s %9d %8.2f %10.0f %10.0f %10.0f\n",
		shortName(args[0])+" (cross)", nrX, psX, wallX, colX, wallX+colX)
	fmt.Printf("%-34s %9d %8.2f %10.0f %10.0f %10.0f\n",
		shortName(args[1])+" (baseline)", nrB, psB, wallB, colB, wallB+colB)
	fmt.Printf("PSNR is measured against %s in BOTH rows, so the two are comparable.\n", shortName(args[1]))
	dB := psX - psB
	dBytes := 100 * ((wallX + colX) - (wallB + colB)) / (wallB + colB)
	fmt.Printf("cross vs baseline: %+.2f dB, %+.2f%% bytes\n", dB, dBytes)
}
