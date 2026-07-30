#!/usr/bin/env bash
# A1 / A1b — does the colour-driven merge dissolve a sprite's silhouette?
# Reports: DESIGN-ALPHA.md, DESIGN-ALPHA-A1-data.txt, DESIGN-ALPHA-A1b-data.txt
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
build_lab
SP="$SPRITES/assets/props/extra"
[ -d "$SP" ] || { echo "no sprite corpus at $SP (set SPRITES=)" >&2; exit 1; }
mkdir -p "$OUT/a1"

echo "=== A3 pilot: how much sprite alpha is soft rim vs interior translucency ==="
"$LAB" alphahist "$SP"/*.png

echo
echo "=== A1/A1b: silhouette dissolution at every mark ==="
printf "%-28s %6s %8s %10s %10s %10s %s\n" render cross dissolv "dissolv%" invisible "invis%" verdict
for s in ak74 bow pickaxe; do
  "$LAB" hd "$SP/$s.png" "$OUT/a1/$s" >/dev/null 2>&1
  for f in "$OUT/a1/$s"/hd_*.png; do
    "$LAB" silhouette "$SP/$s.png" "$f" "$OUT/a1/vis-$s-$(basename "$f" .png).png" 8 | tail -1
  done
done

echo
echo "=== SHPC v2 round trip, alpha included ==="
for pair in "ak74 940" "bow 1368" "pickaxe 160"; do
  s=${pair% *}; n=${pair#* }
  R=$(mark_nearest "$OUT/a1/$s" "$n")
  "$LAB" p4enc "$R" "$OUT/a1/$s.shpc" | grep -E "alpha|FILE"
  # The reference is the ORIGINAL sprite, not the render: at the finest mark they are the same
  # image, and pointing p4dec at the source is what proved this is a LOSSLESS encode (report 32).
  "$LAB" p4dec "$OUT/a1/$s.shpc" "$OUT/a1/$s-rt.png" "$SP/$s.png" | tail -1
done
