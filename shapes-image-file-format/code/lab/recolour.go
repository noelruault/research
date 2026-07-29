package main

// recolour.go — B10: re-price the published colour column with the cross-channel transform.
//
// This measures, it does not propose a lever. Reports 11 and 12 established two facts about the region colour coder, and this file applies them to every published operating point at 3840x2160:
//
//  1. the coder never had a cross-channel transform, and the reversible one (G, R-G, B-G) is worth -28.0% of the modelled colour bill at lossless (report 11);
//  2. modelled bytes and real bytes rank differently, so the headline has to be a compressed stream and not a cross-entropy (report 12).
//
// Every mark therefore gets four numbers on the SAME labels: colorBytes2 (the published pricing function, unmodified), the RCT cost under the same order-0 adaptive model colorBytes2 uses, the size of the actual dumped RCT residual stream under brotli -q11, and that stream with report 12's 8-byte chroma coefficient applied. Columns 3 and 4 are the headline; columns 1 and 2 are there to show where the model and the compressor disagree.
//
// The partition is never re-derived. A lossy mark is the exact 4-connected partition of its own published render — the recovery trick reports 09 through 12 all used — and the lossless mark is the exact partition of the source, which is the same operation on a different image. colorBytes2 is re-measured on those recovered labels, and the published figure is printed beside it, so recovery drift is visible instead of hidden.
//
// Causality: the predictor reads `dec`, filled in region-id order, and region ids come from relabelComponents, which numbers in raster first-appearance order. The a-arm's coefficients are fitted encoder-side over the whole partition and transmitted, which is why they are paid for. `rcdec` rebuilds the image from the partition and the stream alone, so a tap that read undecoded data shows up there as a wrong sample.

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

// rcClamp8 keeps a prediction inside the byte range the true colour lives in; the decoder applies the same clamp, so it costs nothing. Identical to report 12's lxClamp8.
func rcClamp8(v float64) int {
	i := int(math.Round(v))
	if i < 0 {
		return 0
	}
	if i > 255 {
		return 255
	}
	return i
}

// rcWmean is colorBytes2's predictor: the boundary-weighted mean of the already-decoded neighbours.
func rcWmean(a *flAdj, dec [][3]float64, r int) [3]float64 {
	var acc [3]float64
	wsum := 0.0
	for i := a.off[r]; i < a.off[r+1]; i++ {
		nb, ln := a.nb[i], a.ln[i]
		if int(nb) >= r {
			continue
		}
		for c := 0; c < 3; c++ {
			acc[c] += float64(ln) * dec[nb][c]
		}
		wsum += float64(ln)
	}
	if wsum == 0 {
		return [3]float64{128, 128, 128}
	}
	for c := 0; c < 3; c++ {
		acc[c] = math.Round(acc[c] / wsum)
	}
	return acc
}

// rcFitA fits a_R and a_B in
//
//	pred_c = wmean_c + tG * ( 1 + a_c * (wmean_c/wmean_G - 1) )
//
// by least squares with the cross-channel gain k pinned at 1. That is report 12's surviving arm.
// k is deliberately not fitted: fitting it improves the order-0 model by 0.04% and costs brotli 0.60%, which is the whole point of reporting a compressed stream rather than a cross-entropy.
// Two float32 = 8 bytes are transmitted, and every table here charges for them.
// The fit reads cols directly, which is legal because every arm is lossless on the region colours, so a decoded neighbour IS its true colour; and it is encoder-side work in any case.
func rcFitA(a *flAdj, cols [][3]float64) [2]float32 {
	var s12, s22, sy2 [2]float64
	chOf := [2]int{0, 2} // R and B; G is coded first and unchanged
	for r := range cols {
		wm := rcWmean(a, cols, r)
		tG := cols[r][1] - wm[1]
		for j, c := range chOf {
			x2 := 0.0
			if wm[1] >= 1 {
				x2 = tG * (wm[c]/wm[1] - 1)
			}
			s12[j] += tG * x2
			s22[j] += x2 * x2
			sy2[j] += (cols[r][c] - wm[c]) * x2
		}
	}
	var out [2]float32
	for j := range out {
		if s22[j] > 0 {
			out[j] = float32((sy2[j] - s12[j]) / s22[j])
		}
	}
	return out
}

// rcPredA is the a-arm's prediction of R and B, given the weighted mean and the already-coded G step.
func rcPredA(wm [3]float64, tG int, fa [2]float32) (int, int) {
	var p [2]int
	for j, c := range [2]int{0, 2} {
		co := 1.0
		if wm[1] >= 1 {
			co += float64(fa[j]) * (wm[c]/wm[1] - 1)
		}
		p[j] = rcClamp8(wm[c] + float64(tG)*co)
	}
	return p[0], p[1]
}

// rcShape reports what report 12 found the compressor actually lives on: the exact-hit rate and the run structure of the byte stream, not the residual variance.
func rcShape(s []byte) (zeroPct, meanRun float64) {
	if len(s) == 0 {
		return 0, 0
	}
	zeros, runs := 0, 1
	for i, v := range s {
		if v == 0 {
			zeros++
		}
		if i > 0 && v != s[i-1] {
			runs++
		}
	}
	return 100 * float64(zeros) / float64(len(s)), float64(len(s)) / float64(runs)
}

// rcWriteCoef stores the two transmitted chroma coefficients as little-endian float32 = 8 B.
// This file IS the side information the a-column is charged for; its length is the charge.
func rcWriteCoef(path string, fa [2]float32) {
	var b [8]byte
	binary.LittleEndian.PutUint32(b[0:], math.Float32bits(fa[0]))
	binary.LittleEndian.PutUint32(b[4:], math.Float32bits(fa[1]))
	must(os.WriteFile(path, b[:], 0o644))
}

func rcReadCoef(path string) [2]float32 {
	b, err := os.ReadFile(path)
	must(err)
	if len(b) != 8 {
		fmt.Fprintf(os.Stderr, "coef file %s is %d B, expected 8\n", path, len(b))
		os.Exit(1)
	}
	return [2]float32{
		math.Float32frombits(binary.LittleEndian.Uint32(b[0:])),
		math.Float32frombits(binary.LittleEndian.Uint32(b[4:])),
	}
}

// rcRun prices one operating point. imgPath is the published render for a lossy mark, or the source for the lossless mark; in both cases the partition is the exact 4-connected partition of that image.
func rcRun(imgPath, tag string, expect int, dumpDir, srcPath string) {
	t0 := time.Now()
	im := load(imgPath)
	lab, cols, _ := exactPartition(im)
	n := len(cols)
	npix := im.W * im.H

	drift := 0.0
	if expect > 0 {
		drift = 100 * float64(n-expect) / float64(expect)
	}
	// PSNR against the source, on this study's own RGB definition, so a recovered partition can be checked against the published mark rather than assumed to be it.
	ps := math.Inf(1)
	if srcPath != "" && srcPath != imgPath {
		src := load(srcPath)
		ps = psnrSSE(sseBetween(src, im), npix)
	}

	ref := colorBytes2(lab, cols, im.W, im.H)
	a := flBuildAdj(lab, n, im.W, im.H)
	fa := rcFitA(a, cols)

	// One order-0 adaptive model per channel per arm, Laplace +1 — floor.go's mRCT exactly, so the rct column has to reproduce report 11's number and the two arms are directly comparable.
	mRCT := flNewModel(3, 1024)
	mA := flNewModel(3, 1024)
	var rctS, aS []byte
	if dumpDir != "" {
		rctS = make([]byte, 0, 3*n)
		aS = make([]byte, 0, 3*n)
	}
	dec := make([][3]float64, n)
	mismatch := 0
	for r := 0; r < n; r++ {
		wm := rcWmean(a, dec, r)
		var dW [3]int
		for c := 0; c < 3; c++ {
			dW[c] = int(cols[r][c] - wm[c])
		}
		tG, tR, tB := dW[1], dW[0]-dW[1], dW[2]-dW[1]
		mRCT.code(0, tG+512)
		mRCT.code(1, tR+512)
		mRCT.code(2, tB+512)

		pR, pB := rcPredA(wm, tG, fa)
		uR, uB := int(cols[r][0])-pR, int(cols[r][2])-pB
		mA.code(0, tG+512)
		mA.code(1, uR+512)
		mA.code(2, uB+512)

		if dumpDir != "" {
			rctS = append(rctS, byte(tG), byte(tR), byte(tB))
			aS = append(aS, byte(tG), byte(uR), byte(uB))
		}

		// Decode replay, in line: rebuild from the residual and nothing else.
		for c := 0; c < 3; c++ {
			dec[r][c] = wm[c] + float64(dW[c])
			if dec[r][c] != cols[r][c] {
				mismatch++
			}
		}
		if (pR+uR)&255 != int(cols[r][0]) || (pB+uB)&255 != int(cols[r][2]) {
			mismatch++
		}
	}

	zero, run := 0.0, 0.0
	if dumpDir != "" {
		rctP := flPlanar(rctS)
		zero, run = rcShape(rctP)
		flWriteStream(dumpDir+"/"+tag+"_rct_planar.bin", rctP)
		flWriteStream(dumpDir+"/"+tag+"_a_planar.bin", flPlanar(aS))
		rcWriteCoef(dumpDir+"/"+tag+"_a.coef", fa)
		dumpLabels(dumpDir, n, im.W, im.H, lab)
	}

	// One machine-readable line per mark; the driver collects these and joins the brotli sizes onto them.
	fmt.Printf("RC tag=%s regions=%d published=%d drift=%+.4f%% psnr=%.2f colorBytes2=%.2f rct_modelled=%.0f a_modelled=%.0f aR=%.5f aB=%.5f mismatch=%d zero=%.3f meanrun=%.4f px_per_region=%.3f\n",
		tag, n, expect, drift, ps, ref, mRCT.adaptiveBytes(), mA.adaptiveBytes()+8, fa[0], fa[1], mismatch, zero, run,
		float64(npix)/float64(n))
	fmt.Fprintf(os.Stderr, "[%s] %d regions in %s\n", tag, n, time.Since(t0).Round(time.Second))
}

// rcCmd: recolour <image.png> <tag> [publishedRegions] [srcForPSNR];  RCDUMP=<dir> writes the streams.
func rcCmd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: lab recolour <image.png> <tag> [publishedRegions] [src.png]")
		os.Exit(2)
	}
	expect := 0
	if len(args) > 2 {
		expect, _ = strconv.Atoi(args[2])
	}
	src := ""
	if len(args) > 3 {
		src = args[3]
	}
	rcRun(args[0], args[1], expect, os.Getenv("RCDUMP"), src)
}

// rcDec is the check that makes the brotli column a real colour bill and not bookkeeping: given ONLY the partition and one byte stream (plus, for the a arm, the 8 transmitted bytes), rebuild the image.
// usage: rcdec <labels.bin> <stream.bin> <rct|a> <ref.png> [coef.bin]
func rcDec(labPath, streamPath, mode, refPath, coefPath string) {
	lab, w, h := loadLabels(labPath)
	raw, err := os.ReadFile(streamPath)
	must(err)
	n := 0
	for _, l := range lab {
		if int(l)+1 > n {
			n = int(l) + 1
		}
	}
	if len(raw) != 3*n {
		fmt.Fprintf(os.Stderr, "stream is %d B, expected %d\n", len(raw), 3*n)
		os.Exit(1)
	}
	var fa [2]float32
	if mode == "a" {
		fa = rcReadCoef(coefPath)
		fmt.Printf("coefficients a=(%.5f,%.5f), 8 B of side information\n", fa[0], fa[1])
	}
	at := func(r, c int) int { return c*n + r } // every stream this file writes is planar
	a := flBuildAdj(lab, n, w, h)
	dec := make([][3]float64, n)
	for r := 0; r < n; r++ {
		pred := rcWmean(a, dec, r)
		g := int(raw[at(r, 0)])
		gr := (int(pred[1]) + g) & 255
		dec[r][1] = float64(gr)
		// The stream carries the G residual mod 256; G and its prediction are both bytes, so the signed step is recovered exactly with no extra bits.
		tg := gr - int(pred[1])
		switch mode {
		case "rct":
			dec[r][0] = float64((int(pred[0]) + g + int(raw[at(r, 1)])) & 255)
			dec[r][2] = float64((int(pred[2]) + g + int(raw[at(r, 2)])) & 255)
		case "a":
			pR, pB := rcPredA(pred, tg, fa)
			dec[r][0] = float64((pR + int(raw[at(r, 1)])) & 255)
			dec[r][2] = float64((pB + int(raw[at(r, 2)])) & 255)
		default:
			fmt.Fprintln(os.Stderr, "unknown mode")
			os.Exit(2)
		}
	}
	ref := load(refPath)
	bad, maxd := 0, 0.0
	for p, l := range lab {
		for c := 0; c < 3; c++ {
			if d := math.Abs(dec[l][c] - ref.P[p*3+c]); d > 0 {
				bad++
				if d > maxd {
					maxd = d
				}
			}
		}
	}
	fmt.Printf("decode %s (%s, planar): %d regions, %d wrong samples of %d, max |delta| %.0f\n",
		streamPath, mode, n, bad, 3*w*h, maxd)
	if bad != 0 {
		os.Exit(1)
	}
	fmt.Println("EXACT: the partition plus this byte stream reconstruct the image bit for bit")
}
