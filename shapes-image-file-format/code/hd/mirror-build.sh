#!/bin/bash
# Assets for the 1:1 mirror.
#
# Every panel is a native-resolution window of a real decoded file.
# Nothing is resampled, so one screen pixel is one image pixel and the artefacts on show are the encoder's own.
# Display copies are lossless WebP for exactly that reason: a lossy display copy would add its own artefacts on top of the ones being compared, which is the failure this whole study exists to avoid.
set -eu
HD="$(cd "$(dirname "$0")" && pwd)"
L="$HD/../lab/labx"
SRC="$HD/src4k.png"
OUT="$HD/mirror"; rm -rf "$OUT"; mkdir -p "$OUT"
T="$HD/tmp"; mkdir -p "$T"
S=448

# The two shape-coder operating points on show, both taken straight from the scale-space run.
SHAPES_HI="$HD/renders4k/hd_03380956.png"   # 8,055,367 B  53.37 dB
SHAPES_LO="$HD/renders4k/hd_00019338.png"   #   203,511 B  29.44 dB

# Two WebP operating points, because "which one is better" has two honest answers.
# q14 matches SHAPES_LO's file size, so the comparison is iso-byte and you judge the picture.
# q8 matches SHAPES_LO's PSNR, so the comparison is iso-quality and you judge the byte count.
cwebp -q 14 -m 6 -quiet "$SRC" -o "$T/wq14.webp"
dwebp -quiet "$T/wq14.webp" -o "$T/wq14.png"
cwebp -q 8 -m 6 -quiet "$SRC" -o "$T/wq8.webp"
dwebp -quiet "$T/wq8.webp" -o "$T/wq8.png"
echo "iso-byte  WebP q14: $(stat -f%z "$T/wq14.webp") B"
echo "iso-look  WebP q8 : $(stat -f%z "$T/wq8.webp") B"

# WebP lossless decodes to the source bit for bit; assert that rather than assume it.
cwebp -lossless -z 9 -quiet "$SRC" -o "$T/wll.webp"
dwebp -quiet "$T/wll.webp" -o "$T/wll.png"
if ! cmp -s <("$L" stat "$SRC" "$T/wll.png") <(printf 'psnr 99.00  mae 0.0000  max 0  pct>1 0.00  pct>2 0.00  pct>4 0.00  pct>8 0.00\n'); then
  echo "WebP lossless did not round-trip exactly:"; "$L" stat "$SRC" "$T/wll.png"; exit 1
fi
echo "verified: cwebp -lossless round-trips $SRC bit for bit"

emit() { # name srcpng x y
  "$L" crop "$3" "$T/crop.png" "$4" "$5" $S $S
  cwebp -lossless -z 9 -quiet "$T/crop.png" -o "$OUT/$1_$2.webp"
}

# name x y — the three windows, chosen for the three ways a region coder can fail.
CROPS=("sky 288 230" "ridge 1344 1000" "snow 2400 1350")

for spec in "${CROPS[@]}"; do
  set -- $spec
  name=$1; x=$2; y=$3
  emit "$name" orig      "$SRC"        "$x" "$y"
  emit "$name" shapeshi  "$SHAPES_HI"  "$x" "$y"
  emit "$name" shapeslo  "$SHAPES_LO"  "$x" "$y"
  emit "$name" webplo    "$T/wq14.png" "$x" "$y"
  emit "$name" webpiso   "$T/wq8.png"  "$x" "$y"

  # Difference against the exact original, amplified so sub-threshold error is legible.
  "$L" crop "$SRC" "$T/a.png" "$x" "$y" $S $S
  "$L" crop "$SHAPES_HI" "$T/b.png" "$x" "$y" $S $S
  "$L" diff "$T/a.png" "$T/b.png" "$T/d.png" 32
  cwebp -lossless -z 9 -quiet "$T/d.png" -o "$OUT/${name}_diffhi.webp"

  # Per-window fidelity, because a whole-image PSNR says nothing about which content type broke.
  printf "%-6s shapes-hi " "$name"; "$L" stat "$T/a.png" "$T/b.png"
  "$L" crop "$SHAPES_LO" "$T/b.png" "$x" "$y" $S $S
  printf "%-6s shapes-lo " "$name"; "$L" stat "$T/a.png" "$T/b.png"
  "$L" crop "$T/wq14.png" "$T/b.png" "$x" "$y" $S $S
  printf "%-6s webp-lo   " "$name"; "$L" stat "$T/a.png" "$T/b.png"
  "$L" crop "$T/wq8.png" "$T/b.png" "$x" "$y" $S $S
  printf "%-6s webp-iso  " "$name"; "$L" stat "$T/a.png" "$T/b.png"
done

echo "--- asset weight ---"
du -sh "$OUT"; ls -la "$OUT"
