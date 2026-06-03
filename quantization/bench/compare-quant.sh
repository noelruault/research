#!/usr/bin/env bash
# compare-quant.sh — the palette-quality shootout (research report 10).
#
# For each image and each color count, produce a quantized PNG with every tool
# from the SAME source pixels, then score them all with the same CIEDE2000
# scorer (go run . score). No dithering anywhere, so we measure palette
# selection, not dither. Tools:
#   - pngquant (libimagequant)        the quality bar
#   - ImageMagick -colors (octree)    the common baseline
#   - ours/pca                        deterministic default
#   - ours/refine                     PCA-divisive init + k-means refine
#
# Usage: ./compare-quant.sh "16 64" /path/to/images
set -euo pipefail

NS="${1:-16 64}"
SRC="${2:-../../../pixelize/docs/demo/inputs}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "# quant shootout — $(date '+%F %T')"
echo "# pngquant $(pngquant --version 2>&1 | head -1 | awk '{print $1}'), $(convert --version | head -1 | awk '{print $2,$3}'), go $(go version | awk '{print $3}')"
echo "# scorer: CIEDE2000 (Sharma-validated); no dithering; mean over corpus"

meande () { awk '{for(i=1;i<=NF;i++){split($i,a,"=");v[a[1]]=a[2]}; print v["meanDE"]}'; }

for N in $NS; do
  echo
  echo "== N = $N =="
  printf "%-12s %10s %10s %10s %10s\n" "tool" "pngquant" "imagemagick" "ours/pca" "ours/refine"
  # accumulators
  declare -A SUM=( [pngquant]=0 [im]=0 [pca]=0 [refine]=0 )
  COUNT=0
  for f in "$SRC"/*.jpg "$SRC"/*.png; do
    [ -e "$f" ] || continue
    base="$(basename "${f%.*}")"
    orig="$WORK/${base}_orig.png"
    convert "$f" "$orig"                                                   # common source pixels

    pngquant --force --nofs --output "$WORK/pq.png" "$N" -- "$orig" 2>/dev/null || cp "$orig" "$WORK/pq.png"
    convert "$orig" -dither None -colors "$N" "$WORK/im.png"
    go run . emit -in "$orig" -out "$WORK/pca.png"    -n "$N" -q pca
    go run . emit -in "$orig" -out "$WORK/refine.png" -n "$N" -q refine

    pq=$(go run . score -a "$orig" -b "$WORK/pq.png"     | meande)
    im=$(go run . score -a "$orig" -b "$WORK/im.png"     | meande)
    pc=$(go run . score -a "$orig" -b "$WORK/pca.png"    | meande)
    rf=$(go run . score -a "$orig" -b "$WORK/refine.png" | meande)

    printf "%-12s %10.3f %10.3f %10.3f %10.3f\n" "$base" "$pq" "$im" "$pc" "$rf"
    SUM[pngquant]=$(awk -v s="${SUM[pngquant]}" -v x="$pq" 'BEGIN{print s+x}')
    SUM[im]=$(awk -v s="${SUM[im]}" -v x="$im" 'BEGIN{print s+x}')
    SUM[pca]=$(awk -v s="${SUM[pca]}" -v x="$pc" 'BEGIN{print s+x}')
    SUM[refine]=$(awk -v s="${SUM[refine]}" -v x="$rf" 'BEGIN{print s+x}')
    COUNT=$((COUNT+1))
  done
  printf "%-12s %10.3f %10.3f %10.3f %10.3f   (mean dE2000 over %d)\n" "MEAN" \
    "$(awk -v s="${SUM[pngquant]}" -v c="$COUNT" 'BEGIN{print s/c}')" \
    "$(awk -v s="${SUM[im]}" -v c="$COUNT" 'BEGIN{print s/c}')" \
    "$(awk -v s="${SUM[pca]}" -v c="$COUNT" 'BEGIN{print s/c}')" \
    "$(awk -v s="${SUM[refine]}" -v c="$COUNT" 'BEGIN{print s/c}')" "$COUNT"
done
