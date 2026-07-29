#!/bin/bash
# Steelman for report 08 result 4's claim that "WebP cannot go this small".
#
# The claim was measured with cwebp held at 3840x2160, where it bottoms out at q0 = 85,102 B. Nobody serving a 20 KB rendering of a 4K photograph holds the encode at 4K: they encode small and let the client upscale. That pipeline still outputs 3840x2160 pixels and is scored against the same 4K original on the same metric, so it is in the same contest and had to be tried before the claim could stand. It does not stand.
#
# Part 1 searches resolution and quality jointly for the best WebP at each of the shape coder's two sub-floor byte targets. Part 2 controls for the upsampler, because the win must not be an artefact of sips' Lanczos: it is repeated with ffmpeg's nearest-neighbour, bilinear, bicubic and Lanczos.
#
# command : bash code/hd/steelman.sh
set -eu
HD="$(cd "$(dirname "$0")" && pwd)"
L="$HD/../lab/labx"
SRC="$HD/src4k.png"
T="$HD/tmp-steelman"
rm -rf "$T"
mkdir -p "$T"

# The two shape-coder operating points below cwebp's native floor, from 08-rate-ladder-data.txt.
TARGETS=("19819 21.99" "50016 24.99")
RUNGS=("1920 1080" "1280 720" "960 540" "640 360" "480 270" "384 216")

bisect_q() { # target W H -> echoes q, leaves file at $T/best.webp
  local target=$1 W=$2 H=$3 lo=0 hi=100 mid best=-1 bestd=999999999 b d
  for _ in $(seq 1 9); do
    mid=$(((lo + hi) / 2))
    cwebp -q $mid -m 6 -resize $W $H -quiet "$SRC" -o "$T/probe.webp"
    b=$(stat -f%z "$T/probe.webp")
    d=$((b > target ? b - target : target - b))
    if [ $d -lt $bestd ]; then
      bestd=$d
      best=$mid
      cp "$T/probe.webp" "$T/best.webp"
    fi
    if [ $b -lt $target ]; then lo=$mid; else hi=$mid; fi
    [ $((hi - lo)) -le 1 ] && break
  done
  echo "$best"
}

cwebp -q 0 -m 6 -quiet "$SRC" -o "$T/floor.webp"
cat <<EOF
# Can WebP reach the shape coder's two smallest operating points after all?
#
# cwebp at native 3840x2160 bottoms out at q0 = $(stat -f%z "$T/floor.webp") B. Below that the format has no quality setting left -- but it does have a resolution setting, and the delivered pixel count is a choice the shape coder is also making. Both sides here output 3840x2160 and are scored against the same original with this study's RGB PSNR.
#
# Part 1 -- joint resolution/quality search at each sub-floor byte target. Upscale is sips (Lanczos).
# A rung whose bytes miss the target high has not matched it and cannot be counted as a win at that size; 1920x1080 is already at q1 by 31,922 B and simply cannot reach 19,819. Read size_vs_target first.
# columns : target_B  encode_w  encode_h  q  bytes  size_vs_target  psnr_at_4k  shape_coder_psnr  delta_dB
#
EOF
for spec in "${TARGETS[@]}"; do
  set -- $spec
  target=$1
  sp=$2
  for rung in "${RUNGS[@]}"; do
    set -- $rung
    q=$(bisect_q "$target" "$1" "$2")
    b=$(stat -f%z "$T/best.webp")
    dwebp -quiet "$T/best.webp" -o "$T/dec.png"
    sips -z 2160 3840 "$T/dec.png" --out "$T/up.png" >/dev/null 2>&1
    p=$("$L" psnr "$SRC" "$T/up.png")
    printf "%d\t%s\t%s\t%s\t%d\t%+.1f%%\t%s\t%s\t%+.2f\n" \
      "$target" "$1" "$2" "$q" "$b" \
      "$(echo "scale=4; 100 * ($b - $target) / $target" | bc)" "$p" "$sp" "$(echo "$p - $sp" | bc)"
  done
done

cat <<'EOF'
#
# Part 2 -- upsampler control at the 19,819 B target. If the win were an artefact of a good resampler it would disappear under nearest-neighbour. It does not: every filter at every rung beats 21.99 dB.
# columns : encode_w  encode_h  q  bytes  filter  psnr_at_4k
#
EOF
for rung in "1280 720 4" "960 540 18" "640 360 48"; do
  set -- $rung
  cwebp -q $3 -m 6 -resize $1 $2 -quiet "$SRC" -o "$T/c.webp"
  dwebp -quiet "$T/c.webp" -o "$T/c.png"
  b=$(stat -f%z "$T/c.webp")
  for flt in neighbor bilinear bicubic lanczos; do
    ffmpeg -loglevel error -y -i "$T/c.png" -vf "scale=3840:2160:flags=$flt" "$T/u.png"
    printf "%s\t%s\t%s\t%d\t%s\t%s\n" "$1" "$2" "$3" "$b" "$flt" "$("$L" psnr "$SRC" "$T/u.png")"
  done
done
