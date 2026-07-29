package main

// turnprice is the side-by-side bill: for every mark of the published scale-space it prints the contour coder's total under the published turn coder and under each candidate, on identical labels, with the CAE coder alongside so the min() that hd() actually publishes can be recomputed.

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"
)

// dumpLabels writes one relaxed partition so later pricing runs can reuse the exact labels instead of re-walking the merge, which takes minutes at 4K.
func dumpLabels(dir string, nr, w, h int, lab []int32) {
	must(os.MkdirAll(dir, 0o755))
	f, err := os.Create(fmt.Sprintf("%s/lab_%08d.bin", dir, nr))
	must(err)
	defer f.Close()
	must(binary.Write(f, binary.LittleEndian, int32(w)))
	must(binary.Write(f, binary.LittleEndian, int32(h)))
	must(binary.Write(f, binary.LittleEndian, lab))
}

// loadLabels reads a partition written by dumpLabels.
func loadLabels(path string) ([]int32, int, int) {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	var w, h int32
	must(binary.Read(f, binary.LittleEndian, &w))
	must(binary.Read(f, binary.LittleEndian, &h))
	lab := make([]int32, int(w)*int(h))
	must(binary.Read(f, binary.LittleEndian, lab))
	return lab, int(w), int(h)
}

func turnprice(path string, maxReg int) {
	im := load(path)
	npix := im.W * im.H
	t0 := time.Now()
	cs := pickedCtxs()
	hs := pickedHiers()

	fmt.Printf("# %s %dx%d — contour turn-coder variants on identical partitions (hd() scale-space)\n", path, im.W, im.H)
	fmt.Printf("# vertB is the junction map plus direction bits, identical in every column. totals are vertB + that column's turn bits.\n")
	fmt.Printf("%-9s %8s %10s %10s %10s", "regions", "psnr", "crack", "caeB", "vertB")
	for _, c := range cs {
		fmt.Printf(" %14s", c.name)
	}
	for _, h := range hs {
		fmt.Printf(" %30s", h.name)
	}
	fmt.Println()

	marks := hdMarks(npix)
	mi := 0
	m := newMerger(im)
	for mi < len(marks) && marks[mi] >= m.nreg {
		mi++
	}
	m.runRD(marks[len(marks)-1], func(mm *merger, lambda float64) {
		for mi < len(marks) && mm.nreg == marks[mi] {
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
			// Marks far above the pricing window cost minutes of relaxation for a row that is thrown away; the merger state they come from is untouched either way, so the priced partitions are bit-identical.
			if nreg > 2*maxReg {
				fmt.Fprintf(os.Stderr, "mark %d skipped (above pricing window) at %s\n", mm.nreg, time.Since(t0).Round(time.Second))
				mi++
				continue
			}
			lab = relax(im, lab, nreg, lambda*bitsPerEdge, 6)
			nr, ps, bb, _, _ := priceSeg(im, lab)
			if nr <= maxReg {
				cl := crackLen(lab, im.W, im.H)
				ref, refV, refT, _ := contourBytes(lab, im.W, im.H)
				vb, tb, evs, _, ok := contourReplay(lab, im.W, im.H)
				if math.Abs((vb+tb)/8-ref) > 1e-6 || !ok {
					fmt.Fprintf(os.Stderr, "fatal: replay disagrees with contourBytes at %d regions (delta %.6f, reconstruct %v)\n", nr, (vb+tb)/8-ref, ok)
					os.Exit(1)
				}
				_ = refV
				_ = refT
				fmt.Printf("%-9d %8.2f %10d %10.0f %10.0f", nr, ps, cl, bb, vb/8)
				for _, c := range cs {
					fmt.Printf(" %14.0f", (vb+adaptiveTurnBits(evs, c))/8)
				}
				for _, h := range hs {
					fmt.Printf(" %30.0f", (vb+h.price(evs))/8)
				}
				fmt.Println()
				os.Stdout.Sync()
				if d := os.Getenv("LABDUMP"); d != "" {
					dumpLabels(d, nr, im.W, im.H, lab)
				}
			}
			fmt.Fprintf(os.Stderr, "mark %d (relaxed to %d) done at %s\n", mm.nreg, nr, time.Since(t0).Round(time.Second))
			mi++
		}
	})
	fmt.Printf("# total %s\n", time.Since(t0).Round(time.Second))
}
