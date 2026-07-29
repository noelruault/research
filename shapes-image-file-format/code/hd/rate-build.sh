#!/bin/bash
# Assets for the rate slider: the same two coders at seven file sizes across three decades, ending at exact.
#
# Every step is BYTE-MATCHED -- cwebp's quality is searched until its file lands near the shape coder's, so the only thing left to compare is the picture.
#
# The two smallest steps need a second search. At 3840x2160 cwebp bottoms out at q0 = 85,102 B, so a 20 KB file cannot be made by turning quality down at native resolution. It can be made the way anyone actually ships a small image: encode at a lower resolution and let the client upscale. That pipeline still outputs 3840x2160 pixels, so it is in the same contest, and for those steps the script searches BOTH resolution and quality and keeps the best-scoring candidate at the target size. An earlier version of this page pinned those steps to the native q0 floor and reported that WebP "cannot go this small"; that claim did not survive being steelmanned -- see 08-rate-floor-steelman-data.txt and report 06 #10.
#
# The last two steps both pair against WebP LOSSLESS: past ~8 MB WebP stops being lossy on this image, so the axis ends at exact-versus-exact rather than at a quality setting.
#
# The final step is the shape coder's own lossless point -- the exact region partition, whose cost is read from ladder_3840.txt rather than restated here so it can never drift from report 08 result 1. Its render is src4k.png itself: an exact partition reconstructs the source bit for bit, which is the whole meaning of the step.
#
# Windows are 384px here rather than 448: seven steps x two coders x three windows is a lot of pixels to embed, and 384 is still native 1:1 on any normal screen.
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
# The lossless rung, read from the `lab hd` output so this script never restates a measured number.
LL_REGIONS=$(awk '$1=="regions"{print $2; exit}' "$HD/ladder_3840.txt")
LL_BYTES=$(awk '$1=="lossless_total_B"{print $2; exit}' "$HD/ladder_3840.txt")
STEPS+=("$LL_REGIONS:$LL_BYTES:99.00")

CROPS=("sky 288 230" "ridge 1344 1000" "snow 2400 1350")
# Encode rungs tried when the target is below the native floor. 3840 is included so the search can still choose "no resampling" if that ever wins.
RUNGS=("3840 2160" "1920 1080" "1280 720" "960 540" "640 360" "480 270")

cwebp -q 0 -m 6 -quiet "$SRC" -o "$T/floor.webp"
FLOOR=$(stat -f%z "$T/floor.webp")
echo "cwebp native floor at 3840x2160: q0 = $FLOOR B"

# Bisect cwebp -q at a given encode resolution for the file closest to a target; bytes are monotone in q.
# Writes the winning file to $2 and echoes its quality.
bisect_q() { # target out W H
  local target=$1 out=$2 W=$3 H=$4 lo=0 hi=100 mid best=-1 bestd=999999999 b d
  for _ in $(seq 1 9); do
    mid=$(((lo + hi) / 2))
    cwebp -q $mid -m 6 -resize $W $H -quiet "$SRC" -o "$T/probe.webp"
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

# Decode $1 to a full-resolution PNG at $2, upscaling if the file is smaller than the source.
to_4k() {
  dwebp -quiet "$1" -o "$T/dec.png"
  if [ "$(sips -g pixelWidth "$T/dec.png" | tail -1 | tr -d ' \t' | cut -d: -f2)" = "3840" ]; then
    cp "$T/dec.png" "$2"
  else
    sips -z 2160 3840 "$T/dec.png" --out "$2" >/dev/null 2>&1
  fi
}

# Search resolution AND quality for the best-scoring WebP at a byte target. Used below the native floor.
# Echoes "<label> <bytes> <psnr>"; leaves the winner at $T/w.webp.
match_resampled() { # target
  local target=$1 bestp="-1" bestlabel="" bestbytes=0 W H q b p
  for rung in "${RUNGS[@]}"; do
    set -- $rung
    W=$1
    H=$2
    q=$(bisect_q "$target" "$T/cand.webp" "$W" "$H")
    b=$(stat -f%z "$T/cand.webp")
    # Reject candidates that miss the target high by more than 5%: this is a byte-matched ladder.
    [ "$b" -gt $((target * 105 / 100)) ] && continue
    to_4k "$T/cand.webp" "$T/cand4k.png"
    p=$("$L" psnr "$SRC" "$T/cand4k.png")
    printf "    try %4sx%-4s q%-3s %8d B  %s dB\n" "$W" "$H" "$q" "$b" "$p" >&2
    if [ "$(echo "$p > $bestp" | bc)" = "1" ]; then
      bestp=$p
      bestbytes=$b
      bestlabel="q$q@${W}x${H}"
      [ "$W" = "3840" ] && bestlabel="q$q"
      cp "$T/cand.webp" "$T/w.webp"
    fi
  done
  echo "$bestlabel $bestbytes $bestp"
}

: >"$OUT/steps.tsv"
# Per-window fidelity, keyed by step and window rather than by line order: the page reads this file directly, and a positional format silently mis-attributes every number the moment a window is added or a step is inserted. That is exactly how the floor-era numbers survived a rebuild.
: >"$OUT/window-stats.tsv"
i=0
for spec in "${STEPS[@]}"; do
  i=$((i + 1))
  regions=${spec%%:*}
  rest=${spec#*:}
  bytes=${rest%%:*}
  psnr=${rest#*:}
  resampled=0

  # The exact partition renders to the source itself; every other step has a scale-space render.
  if [ "$psnr" = "99.00" ]; then
    shp="$SRC"
  else
    shp=$(printf "%s/renders4k/hd_%08d.png" "$HD" "$regions")
  fi

  if [ "$i" -ge $((${#STEPS[@]} - 1)) ]; then
    # The top two steps both compare against a bit-exact WebP.
    cwebp -lossless -z 9 -quiet "$SRC" -o "$T/w.webp"
    wq="lossless"
  elif [ "$bytes" -lt "$FLOOR" ]; then
    read -r wq _ _ < <(match_resampled "$bytes")
    resampled=1
  else
    wq="q$(bisect_q "$bytes" "$T/w.webp" 3840 2160)"
  fi
  to_4k "$T/w.webp" "$T/w.png"
  wbytes=$(stat -f%z "$T/w.webp")
  wpsnr=$("$L" psnr "$SRC" "$T/w.png")

  printf "step %d  shapes %9d B %6s dB (%s regions)   webp %-14s %9d B %6s dB%s\n" \
    "$i" "$bytes" "$psnr" "$regions" "$wq" "$wbytes" "$wpsnr" \
    "$([ $resampled = 1 ] && echo '   <- below the native floor: encoded small, upscaled')"
  printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
    "$i" "$regions" "$bytes" "$psnr" "$wq" "$wbytes" "$wpsnr" "$resampled" >>"$OUT/steps.tsv"

  for spec2 in "${CROPS[@]}"; do
    set -- $spec2
    name=$1
    x=$2
    y=$3
    "$L" crop "$SRC" "$T/o.png" "$x" "$y" $S $S
    "$L" crop "$shp" "$T/c.png" "$x" "$y" $S $S
    cwebp -lossless -z 9 -quiet "$T/c.png" -o "$OUT/${name}_s${i}_shapes.webp"
    st=$("$L" stat "$T/o.png" "$T/c.png")
    printf "  %-6s shapes %s\n" "$name" "$st"
    printf "%s\t%s\tshapes\t%s\n" "$i" "$name" "$(echo "$st" | awk '{print $2}')" >>"$OUT/window-stats.tsv"
    "$L" crop "$T/w.png" "$T/c.png" "$x" "$y" $S $S
    cwebp -lossless -z 9 -quiet "$T/c.png" -o "$OUT/${name}_s${i}_webp.webp"
    st=$("$L" stat "$T/o.png" "$T/c.png")
    printf "  %-6s webp   %s\n" "$name" "$st"
    printf "%s\t%s\twebp\t%s\n" "$i" "$name" "$(echo "$st" | awk '{print $2}')" >>"$OUT/window-stats.tsv"
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
