package main

import "fmt"

// hdnd is the experiment that exposed correction #7 in report 06.
// It calls colorBytes repeatedly on ONE fixed partition inside ONE process.
// Any spread between the printed values is Go's randomised map iteration leaking into a published number, which is exactly what it used to do: six calls returned six different answers spanning 224 B.
// It is kept runnable because a determinism claim nobody can re-check is just a determinism hope.
func hdnd(path string) {
	im := load(path)
	lab, cols, _ := exactPartition(im)
	fmt.Printf("%d regions; colorBytes over 6 calls:\n", len(cols))
	for i := 0; i < 6; i++ {
		fmt.Printf("  %.4f B\n", colorBytes(lab, cols, im.W, im.H))
	}
	fmt.Printf("colorBytesLean over 3 calls:\n")
	for i := 0; i < 3; i++ {
		fmt.Printf("  %.4f B\n", colorBytesLean(lab, cols, im.W, im.H))
	}
}
