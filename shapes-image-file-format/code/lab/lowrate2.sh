#!/bin/bash
# Full-range rate-distortion sweep in a single run, so both ends of the curve are comparable.
set -e
cd "$(dirname "$0")"
SRC=sierra.png

echo "codec    q    bytes      bpp    PSNR"
for q in 0 1 2 4 6 8 10 14 18 22 26 30 34 38 42 46; do
  cwebp -q $q -quiet "$SRC" -o /tmp/lr.webp
  dwebp -quiet /tmp/lr.webp -o /tmp/lr.png
  b=$(stat -f%z /tmp/lr.webp)
  p=$(./lab psnr "$SRC" /tmp/lr.png | grep -oE '[0-9]+\.[0-9]+' | head -1)
  printf "webp   %3d  %7d  %7.4f  %6s\n" "$q" "$b" "$(echo "scale=5;$b*8/147456" | bc)" "$p"
done

for q in 6 10 14 18 22 26 30 34 38; do
  avifenc -q $q -s 4 "$SRC" /tmp/lr.avif >/dev/null 2>&1
  avifdec /tmp/lr.avif /tmp/lra.png >/dev/null 2>&1
  b=$(stat -f%z /tmp/lr.avif)
  p=$(./lab psnr "$SRC" /tmp/lra.png | grep -oE '[0-9]+\.[0-9]+' | head -1)
  printf "avif   %3d  %7d  %7.4f  %6s\n" "$q" "$b" "$(echo "scale=5;$b*8/147456" | bc)" "$p"
done
