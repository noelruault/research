#!/usr/bin/env bash
# Report 32 — SHPC v2 against PNG / WebP / AVIF on game sprites.
# At the finest mark the merge has nothing to merge, so every arm here is LOSSLESS on the same
# pixels; p4dec verifies that against the original sprite rather than assuming it.
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
need cwebp; need avifenc
build_lab
SP="$SPRITES/assets/props/extra"
[ -d "$SP" ] || { echo "no sprite corpus at $SP (set SPRITES=)" >&2; exit 1; }
mkdir -p "$OUT/32"

printf "%-9s %7s %11s %10s %10s %10s %14s\n" sprite regions "OURS .shpc" "PNG(Go)" "WebP-ll" "AVIF-ll" "as-authored"
for pair in "ak74 940" "bow 1368" "pickaxe 160"; do
  s=${pair% *}; n=${pair#* }
  "$LAB" hd "$SP/$s.png" "$OUT/32/$s" >/dev/null 2>&1
  R=$(mark_nearest "$OUT/32/$s" "$n")
  "$LAB" p4enc "$R" "$OUT/32/$s.shpc" >/dev/null
  "$LAB" p4dec "$OUT/32/$s.shpc" /dev/null "$SP/$s.png" >/dev/null || echo "  !! $s does not round-trip to the SOURCE"
  # -exact matters: without it cwebp is free to rewrite RGB under fully transparent pixels,
  # which would make it a different image and the comparison meaningless.
  cwebp -quiet -lossless -z 9 -exact "$R" -o "$OUT/32/$s.webp"
  avifenc -s 0 --lossless "$R" "$OUT/32/$s.avif" >/dev/null 2>&1
  printf "%-9s %7s %11d %10d %10d %10d %14d\n" "$s" "$n" \
    "$(wc -c < "$OUT/32/$s.shpc")" "$(wc -c < "$R")" \
    "$(wc -c < "$OUT/32/$s.webp")" "$(wc -c < "$OUT/32/$s.avif")" "$(wc -c < "$SP/$s.png")"
done
echo
echo "as-authored is NOT a codec comparison: those PNGs were never run through an optimiser."
