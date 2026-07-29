#!/bin/bash
# WebP rate-distortion at each rung of the resolution ladder, on the study's own PSNR.
# Same content at every rung (all four are resamples of one 4K original), so resolution is the only variable.
set -u
L=../lab/labx
for W in 3840 1920 960 512; do
  case $W in 3840) SRC=src4k.png;; *) SRC=src$W.png;; esac
  PX=$(( W * W * 9 / 16 ))
  for q in 0 2 5 8 12 16 20 25 30 40 50 60 70 80 90 95 100; do
    cwebp -q $q -m 6 -quiet "$SRC" -o /tmp/lw.webp 2>/dev/null || continue
    dwebp -quiet /tmp/lw.webp -o /tmp/lw.png 2>/dev/null
    printf "%d webp q%d %d %s\n" "$W" "$q" "$(stat -f%z /tmp/lw.webp)" "$($L psnr "$SRC" /tmp/lw.png)"
  done
  cwebp -lossless -z 9 -quiet "$SRC" -o /tmp/lwl.webp 2>/dev/null
  printf "%d webp-ll - %d 99.00\n" "$W" "$(stat -f%z /tmp/lwl.webp)"
  for q in 20 30 40 50 60 70 80; do
    avifenc -q $q -s 6 "$SRC" /tmp/la.avif >/dev/null 2>&1 || continue
    avifdec /tmp/la.avif /tmp/la.png >/dev/null 2>&1
    printf "%d avif q%d %d %s\n" "$W" "$q" "$(stat -f%z /tmp/la.avif)" "$($L psnr "$SRC" /tmp/la.png)"
  done
done
