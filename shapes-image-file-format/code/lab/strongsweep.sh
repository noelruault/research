#!/bin/bash
# The report 05 sweep, re-run at each codec's strong settings rather than its defaults.
# Report 05 used `cwebp -q N` (method 4) and `avifenc -s 4`; neither is what a publisher shipping
# these files would use, and both leave bytes on the table that the shape coder was then credited with.
set -e
cd "$(dirname "$0")"
SRC=sierra.png
echo "codec    q    bytes      bpp    PSNR"
for q in 0 1 2 4 6 8 10 14 18 22 26 30 34 38 42 46; do
  cwebp -q $q -m 6 -quiet "$SRC" -o /tmp/sw.webp
  dwebp -quiet /tmp/sw.webp -o /tmp/sw.png
  b=$(stat -f%z /tmp/sw.webp)
  p=$(./lab psnr "$SRC" /tmp/sw.png | grep -oE '[0-9]+\.[0-9]+' | head -1)
  printf "webp   %3d  %7d  %7.4f  %6s\n" "$q" "$b" "$(echo "scale=5;$b*8/147456" | bc)" "$p"
done
for q in 6 10 14 18 22 26 30 34 38; do
  avifenc -q $q -s 0 "$SRC" /tmp/sw.avif >/dev/null 2>&1
  avifdec /tmp/sw.avif /tmp/swa.png >/dev/null 2>&1
  b=$(stat -f%z /tmp/sw.avif)
  p=$(./lab psnr "$SRC" /tmp/swa.png | grep -oE '[0-9]+\.[0-9]+' | head -1)
  printf "avif   %3d  %7d  %7.4f  %6s\n" "$q" "$b" "$(echo "scale=5;$b*8/147456" | bc)" "$p"
done
