#!/usr/bin/env bash
# Report 35 — SUPERVISED classification, region graph vs pixel grid, with the steelman.
#
# The sample coordinates below are specific to the two images used in report 35. For any other
# image they must be re-picked: run with a probe set first and read the printed rgb values, which
# is how the second run was fixed after two "keep" points landed on grass.
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
need cwebp; need dwebp
build_lab
PHOTOS="${PHOTOS:?set PHOTOS=/path/to/images}"
IMG="${IMG:?set IMG=<basename without extension>}"
KEEP="${KEEP:?set KEEP=x,y;x,y}"
REMOVE="${REMOVE:?set REMOVE=x,y;x,y}"
REGIONS="${REGIONS:-1200}"
mkdir -p "$OUT/35/src"

SRCIN=$(ls "$PHOTOS/$IMG".* | head -1)
SRC="$OUT/35/src/$IMG.png"
to_png "$SRCIN" "$SRC"
"$LAB" hd "$SRC" "$OUT/35/$IMG" >/dev/null 2>&1
R=$(mark_nearest "$OUT/35/$IMG" "$REGIONS")
op=$("$LAB" psnr "$SRC" "$R")
echo "ours: $(basename "$R") at $op dB"
for q in 1 3 5 8 12 16 20 25 30 35 40 50 60 70 80 90; do
  cwebp -quiet -m 6 -q $q "$SRC" -o "$OUT/35/$IMG.webp" 2>/dev/null
  dwebp -quiet "$OUT/35/$IMG.webp" -o "$OUT/35/$IMG-webp.png" 2>/dev/null
  p=$("$LAB" psnr "$SRC" "$OUT/35/$IMG-webp.png")
  awk "BEGIN{exit !($p >= $op)}" && { echo "webp: q$q at $p dB, $(wc -c < "$OUT/35/$IMG.webp") B"; break; }
done

# Probe mode: pass PROBE=1 with KEEP set to a grid to read colours before choosing real examples.
"$LAB" bgclass "$SRC" "$R" "$OUT/35/$IMG-webp.png" "$OUT/35/$IMG-class.png" "keep=$KEEP" "remove=$REMOVE"
