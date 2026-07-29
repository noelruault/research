#!/bin/bash
# Rate-distortion sweep of the shipped raster codecs at native 3840x2160.
# Every codec is decoded back to PNG and scored with the study's own RGB PSNR, so the shape coder and the raster codecs are on one metric. Bytes are the real encoded file including its container.
set -u
HD="$(cd "$(dirname "$0")" && pwd)"
LAB="$HD/../lab"
SRC="$HD/src4k.png"
T="$HD/tmp"; mkdir -p "$T"
psnr() { "$LAB/lab" psnr "$SRC" "$1"; }

printf "%-10s %-6s %12s %8s %9s\n" codec setting bytes psnr bpp
row() { printf "%-10s %-6s %12d %8.2f %9.4f\n" "$1" "$2" "$3" "$4" "$(echo "$3 * 8 / 8294400" | bc -l)"; }

for q in 0 5 10 20 30 40 50 60 70 80 85 90 95 100; do
  cwebp -q $q -m 6 -quiet "$SRC" -o "$T/w.webp" 2>/dev/null || continue
  dwebp -quiet "$T/w.webp" -o "$T/w.png" 2>/dev/null
  row webp "q$q" "$(stat -f%z "$T/w.webp")" "$(psnr "$T/w.png")"
done

for n in 80 60 40 20; do
  cwebp -near_lossless $n -q 100 -m 6 -quiet "$SRC" -o "$T/wn.webp" 2>/dev/null || continue
  dwebp -quiet "$T/wn.webp" -o "$T/wn.png" 2>/dev/null
  row webp-nl "n$n" "$(stat -f%z "$T/wn.webp")" "$(psnr "$T/wn.png")"
done

cwebp -lossless -z 9 -quiet "$SRC" -o "$T/wl.webp" 2>/dev/null
row webp-ll "-" "$(stat -f%z "$T/wl.webp")" 99

for q in 20 30 40 50 60 70 80 90; do
  avifenc -q $q -s 6 "$SRC" "$T/a.avif" >/dev/null 2>&1 || continue
  avifdec "$T/a.avif" "$T/a.png" >/dev/null 2>&1
  row avif "q$q" "$(stat -f%z "$T/a.avif")" "$(psnr "$T/a.png")"
done

avifenc --lossless -s 6 "$SRC" "$T/al.avif" >/dev/null 2>&1
row avif-ll "-" "$(stat -f%z "$T/al.avif")" 99

for d in 5 3 2 1 0.5; do
  cjxl -d $d -e 7 "$SRC" "$T/j.jxl" >/dev/null 2>&1 || continue
  djxl "$T/j.jxl" "$T/j.png" >/dev/null 2>&1
  row jxl "d$d" "$(stat -f%z "$T/j.jxl")" "$(psnr "$T/j.png")"
done

cjxl -d 0 -e 7 "$SRC" "$T/jl.jxl" >/dev/null 2>&1
row jxl-ll "-" "$(stat -f%z "$T/jl.jxl")" 99

for q in 30 50 70 85 95; do
  sips -s format jpeg -s formatOptions $q "$SRC" --out "$T/p.jpg" >/dev/null 2>&1
  sips -s format png "$T/p.jpg" --out "$T/p.png" >/dev/null 2>&1
  row jpeg "q$q" "$(stat -f%z "$T/p.jpg")" "$(psnr "$T/p.png")"
done

row png "-" "$(stat -f%z "$SRC")" 99
