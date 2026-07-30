#!/usr/bin/env bash
# Report 33 — matched-fidelity bytes on photographs, and the UNSUPERVISED flood (which failed).
# PHOTOS must point at a directory of images; the corpus used in report 33 is not committed.
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
need cwebp; need dwebp
build_lab
PHOTOS="${PHOTOS:?set PHOTOS=/path/to/images}"
mkdir -p "$OUT/33/src"
for f in "$PHOTOS"/*; do
  case "$f" in *.png|*.jpg|*.jpeg|*.PNG|*.JPG|*.JPEG) to_png "$f" "$OUT/33/src/$(basename "${f%.*}").png";; esac
done

echo "=== matched-fidelity bytes, capability band (~1,200 regions) ==="
printf "%-12s %8s %8s %11s %10s %5s %10s %9s\n" image regions PSNR "OURS .shpc" "WebP m6" q "WebP PSNR" delta
for f in "$OUT/33/src"/*.png; do
  b=$(basename "$f" .png)
  "$LAB" hd "$f" "$OUT/33/$b" >/dev/null 2>&1
  R=$(mark_nearest "$OUT/33/$b" 1200)
  "$LAB" p4enc "$R" "$OUT/33/$b.shpc" >/dev/null
  ours=$(wc -c < "$OUT/33/$b.shpc"); op=$("$LAB" psnr "$f" "$R")
  # Smallest -m 6 quality reaching our fidelity. -m 6 because the invariants say any knob given
  # to one side is given to the other, and this side gets max effort.
  for q in 1 3 5 8 12 16 20 25 30 35 40 45 50 55 60 70 80 90; do
    cwebp -quiet -m 6 -q $q "$f" -o "$OUT/33/$b.webp" 2>/dev/null
    dwebp -quiet "$OUT/33/$b.webp" -o "$OUT/33/$b-webp.png" 2>/dev/null
    p=$("$LAB" psnr "$f" "$OUT/33/$b-webp.png")
    if awk "BEGIN{exit !($p >= $op)}"; then
      wb=$(wc -c < "$OUT/33/$b.webp")
      printf "%-12s %8s %8s %11d %10d %5s %10s %8s%%\n" "$b" \
        "$(basename "$R" .png | sed 's/hd_0*//')" "$op" "$ours" "$wb" "$q" "$p" \
        "$(awk "BEGIN{printf \"%+.1f\", 100*($ours-$wb)/$wb}")"
      break
    fi
  done
done

echo
echo "=== the unsupervised flood, region graph vs pixel grid (report 33's negative result) ==="
for f in "$OUT/33/src"/*.png; do
  b=$(basename "$f" .png); R=$(mark_nearest "$OUT/33/$b" 1200)
  [ -f "$OUT/33/$b-webp.png" ] || continue
  echo "-- $b"
  for tol in 28 55 90; do
    "$LAB" bgcut "$f" "$R" "$OUT/33/$b-webp.png" "$OUT/33/$b-cut$tol.png" $tol | sed -n '2,4p'
  done
done
