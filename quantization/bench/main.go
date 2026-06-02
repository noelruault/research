// Command quantbench measures color-quantization pieces on a corpus,
// reporting RGB MSE/PSNR and perceptual CIEDE2000 (mean + p95). It is the
// research harness for pixelize's quantize package — self-contained, importing
// nothing from pixelize. See ../00-methodology.md.
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	corpus := flag.String("corpus", "../../../pixelize/docs/demo/inputs", "directory of source images")
	nlist := flag.String("n", "16", "comma-separated palette sizes")
	flag.Parse()

	imgs, err := loadCorpus(*corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "corpus:", err)
		os.Exit(1)
	}
	if len(imgs) == 0 {
		fmt.Fprintln(os.Stderr, "no images found in", *corpus)
		os.Exit(1)
	}

	var ns []int
	for _, s := range strings.Split(*nlist, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			ns = append(ns, v)
		}
	}

	quantizers := []Quantizer{Popularity{}, MedianCut{}}

	fmt.Printf("# quantbench — %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("# go %s, %d cores, %d images\n", runtime.Version(), runtime.NumCPU(), len(imgs))
	fmt.Printf("# metric scale: MSE lower=better, PSNR higher=better, dE2000 lower=better\n\n")

	for _, n := range ns {
		fmt.Printf("== N = %d ==\n", n)
		fmt.Printf("%-14s %10s %8s %9s %9s   %s\n", "quantizer", "MSE", "PSNR", "meanDE", "p95DE", "image")
		for _, q := range quantizers {
			var aggMSE, aggMean, aggP95, aggPSNR float64
			t0 := time.Now()
			for _, im := range imgs {
				pal := q.Quantize(im.img, n)
				s := scoreAgainst(im.img, pal)
				aggMSE += s.MSE
				aggPSNR += s.PSNR
				aggMean += s.MeanDE
				aggP95 += s.P95DE
				fmt.Printf("%-14s %10.2f %8.2f %9.3f %9.3f   %s\n",
					q.Name(), s.MSE, s.PSNR, s.MeanDE, s.P95DE, im.name)
			}
			k := float64(len(imgs))
			fmt.Printf("%-14s %10.2f %8.2f %9.3f %9.3f   [MEAN over %d, %dms]\n\n",
				q.Name()+"*", aggMSE/k, aggPSNR/k, aggMean/k, aggP95/k, len(imgs),
				time.Since(t0).Milliseconds())
		}
	}
}

type namedImage struct {
	name string
	img  image.Image
}

func loadCorpus(dir string) ([]namedImage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []namedImage
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out = append(out, namedImage{name: e.Name(), img: img})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}
