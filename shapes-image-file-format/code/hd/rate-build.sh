#!/bin/bash
# Assets for the rate slider: the same two coders at six file sizes across three decades.
#
# Each step is BYTE-MATCHED where that is possible -- cwebp's quality is searched until its file lands near the shape coder's, so the only thing left to compare is the picture.
#
# Two steps are NOT matchable, and that is a result rather than a problem: at 3840x2160 cwebp bottoms out at q0 = 85,102 B, so the two smallest steps ask for files WebP cannot produce at all. Those are pinned to the floor and flagged, because pretending the axis is symmetric would hide the one place the region coder genuinely goes where WebP does not.
#
# The top step is WebP LOSSLESS: past ~8 MB WebP stops being lossy on this image, so that is the honest end of the axis rather than a quality setting.
#
# Windows are 384px here rather than 448: six steps x two coders x three windows is a lot of pixels to embed, and 384 is still native 1:1 on any normal screen.
set -eu
HD="$(cd "$(dirname "$0")" && pwd)"
L="$HD/../lab/labx"
SRC="$HD/src4k.png"
OUT="$HD/rate"
rm -rf "$OUT"
mkdir -p "$OUT"
T="$HD/tmp"
mkdir -p "$T"
S=384

# regions:bytes:psnr -- a log-spaced walk through the scale-space measured in ladder_3840.txt
STEPS=(
  "227:19819:21.99"
  "1383:50016:24.99"
  "11121:153190:28.51"
  "96359:533107:32.74"
  "710144:2413389:40.42"
  "3380956:8055367:53.37"
)
CROPS=("sky 288 230" "ridge 1344 1000" "snow 2400 1350")

cwebp -q 0 -m 6 -quiet "$SRC" -o "$T/floor.webp"
FLOOR=$(stat -f%z "$T/floor.webp")
echo "cwebp floor at $(sips -g pixelWidth "$SRC" | tail -1 | tr -d ' '): q0 = $FLOOR B"

# Bisect cwebp -q for the file size closest to a target; bytes are monotone in q.
match_webp() {
  local target=$1 out=$2 lo=0 hi=100 mid best=-1 bestd=999999999 b d
  for _ in $(seq 1 9); do
    mid=$(((lo + hi) / 2))
    cwebp -q $mid -m 6 -quiet "$SRC" -o "$T/probe.webp"
    b=$(stat -f%z "$T/probe.webp")
    d=$((b > target ? b - target : target - b))
    if [ $d -lt $bestd ]; then
      bestd=$d
      best=$mid
      cp "$T/probe.webp" "$out"
    fi
    if [ $b -lt $target ]; then lo=$mid; else hi=$mid; fi
    [ $((hi - lo)) -le 1 ] && break
  done
  echo "$best"
}

: >"$OUT/steps.tsv"
i=0
for spec in "${STEPS[@]}"; do
  i=$((i + 1))
  regions=${spec%%:*}
  rest=${spec#*:}
  bytes=${rest%%:*}
  psnr=${rest#*:}
  shp=$(printf "%s/renders4k/hd_%08d.png" "$HD" "$regions")
  floored=0

  if [ "$i" -eq "${#STEPS[@]}" ]; then
    cwebp -lossless -z 9 -quiet "$SRC" -o "$T/w.webp"
    wq="lossless"
  elif [ "$bytes" -lt "$FLOOR" ]; then
    cp "$T/floor.webp" "$T/w.webp"
    wq="q0 floor"
    floored=1
  else
    wq="q$(match_webp "$bytes" "$T/w.webp")"
  fi
  dwebp -quiet "$T/w.webp" -o "$T/w.png"
  wbytes=$(stat -f%z "$T/w.webp")
  wpsnr=$("$L" psnr "$SRC" "$T/w.png")

  printf "step %d  shapes %9d B %6s dB (%s regions)   webp %-9s %9d B %6s dB%s\n" \
    "$i" "$bytes" "$psnr" "$regions" "$wq" "$wbytes" "$wpsnr" \
    "$([ $floored = 1 ] && echo '   <- WebP cannot go this small')"
  printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
    "$i" "$regions" "$bytes" "$psnr" "$wq" "$wbytes" "$wpsnr" "$floored" >>"$OUT/steps.tsv"

  for spec2 in "${CROPS[@]}"; do
    set -- $spec2
    name=$1
    x=$2
    y=$3
    "$L" crop "$SRC" "$T/o.png" "$x" "$y" $S $S
    "$L" crop "$shp" "$T/c.png" "$x" "$y" $S $S
    cwebp -lossless -z 9 -quiet "$T/c.png" -o "$OUT/${name}_s${i}_shapes.webp"
    printf "  %-6s shapes " "$name"
    "$L" stat "$T/o.png" "$T/c.png"
    "$L" crop "$T/w.png" "$T/c.png" "$x" "$y" $S $S
    cwebp -lossless -z 9 -quiet "$T/c.png" -o "$OUT/${name}_s${i}_webp.webp"
    printf "  %-6s webp   " "$name"
    "$L" stat "$T/o.png" "$T/c.png"
  done
done

# The reference panel, one per window: the exact original.
for spec2 in "${CROPS[@]}"; do
  set -- $spec2
  "$L" crop "$SRC" "$T/o.png" "$2" "$3" $S $S
  cwebp -lossless -z 9 -quiet "$T/o.png" -o "$OUT/$1_orig.webp"
done

echo "--- weight ---"
du -sh "$OUT"
