#!/bin/bash
set -e
cd "$(dirname "$0")"
NAMES=(athens liberty monet napoleon pearl starry sugarhigh who-king)

run() {
  local ext=$1 dir=$2 solo=0 dict=0 n m s d
  for n in "${NAMES[@]}"; do
    brotli -f -q 11 -c "$dir/$n.$ext" > /tmp/solo.br
    s=$(stat -f%z /tmp/solo.br)
    : > /tmp/dict.bin
    for m in "${NAMES[@]}"; do [ "$m" = "$n" ] || cat "$dir/$m.$ext" >> /tmp/dict.bin; done
    brotli -f -q 11 -D /tmp/dict.bin -c "$dir/$n.$ext" > /tmp/dict.br
    d=$(stat -f%z /tmp/dict.br)
    solo=$((solo+s)); dict=$((dict+d))
    printf "  %-11s solo %7d   +dict %7d   %.2fx\n" "$n" "$s" "$d" "$(echo "scale=3;$s/$d" | bc)"
  done
  printf "  %-11s SUM  %7d   SUM   %7d   %.2fx\n\n" TOTAL "$solo" "$dict" "$(echo "scale=3;$solo/$dict" | bc)"
}

echo "=== webp-lossless (already entropy-coded) ==="; run webp quant
echo "=== .shapes JSON ==="; run shapes shp
echo "=== .svg ==="; run svg shp
