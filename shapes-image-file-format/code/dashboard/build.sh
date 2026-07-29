#!/bin/bash
# Dashboard assets. Codecs run at their STRONG settings (cwebp -m 6, avifenc -s 0), not CLI defaults -- see report 06 #9.
# Dashboard assets: for each encoder a real encoded file (its byte count) plus a decoded copy for display. Display copies are LOSSLESS WebP, so the artefacts shown are the encoder's own and not a re-encode of them.
set -e
S=/private/tmp/claude-501/-Users-noelruault-go-src-github-com-noelruault/23e7d4b1-17da-4b78-8110-6154f88a8011/scratchpad
LAB=$S/lab
OUT=$S/dash/assets
rm -rf "$OUT"; mkdir -p "$OUT"
cd "$LAB"

M=$S/dash/manifest.tsv
: > "$M"

psnr() { ./lab psnr sierra.png "$1" | grep -oE '[0-9]+\.[0-9]+' | head -1; }
emit() { # id family label bytes psnr srcpng
  cwebp -lossless -quiet "$6" -o "$OUT/$1.webp"
  printf "%s\t%s\t%s\t%s\t%s\n" "$1" "$2" "$3" "$4" "$5" >> "$M"
}

emit source source "original 512x288" "$(stat -f%z sierra.png)" 99 sierra.png

# --- JPEG: find the two operating points ---
for q in 30 44; do
  sips -s format jpeg -s formatOptions $q sierra.png --out /tmp/d.jpg >/dev/null 2>&1
  sips -s format png /tmp/d.jpg --out /tmp/dj.png >/dev/null 2>&1
  echo "jpeg q$q -> $(stat -f%z /tmp/d.jpg) B  $(psnr /tmp/dj.png) dB" >&2
done

# ---- Row A: the eval fidelity, ~28.66 dB ----
cwebp -q 36 -m 6 -quiet sierra.png -o /tmp/a.webp; dwebp -quiet /tmp/a.webp -o /tmp/a.png
emit a_webp webp "WebP q36 -m 6" "$(stat -f%z /tmp/a.webp)" "$(psnr /tmp/a.png)" /tmp/a.png

avifenc -q 28 -s 0 sierra.png /tmp/a.avif >/dev/null 2>&1; avifdec /tmp/a.avif /tmp/a.png >/dev/null 2>&1
emit a_avif avif "AVIF q28 -s 0" "$(stat -f%z /tmp/a.avif)" "$(psnr /tmp/a.png)" /tmp/a.png

emit a_shapes shapes "shape coder, 1,685 regions" 12202 28.66 render_1685_28.66.png

sips -s format jpeg -s formatOptions 36 sierra.png --out /tmp/a.jpg >/dev/null 2>&1
sips -s format png /tmp/a.jpg --out /tmp/aj.png >/dev/null 2>&1
emit a_jpeg jpeg "JPEG" "$(stat -f%z /tmp/a.jpg)" "$(psnr /tmp/aj.png)" /tmp/aj.png

emit a_png png "indexed PNG, n=16 grid" "$(stat -f%z $S/wall/quant.png)" "$(psnr $S/wall/quant.png)" "$S/wall/quant.png"
emit a_shipped shipped "SVG rect cover (shipped), 32,924 rects" 113562 "$(psnr $S/wall/quant.png)" "$S/wall/quant.png"

# ---- Row B: the low-rate band, ~26.4 dB, where the shape coder wins ----
cwebp -q 14 -m 6 -quiet sierra.png -o /tmp/b.webp; dwebp -quiet /tmp/b.webp -o /tmp/b.png
emit b_webp webp "WebP q14 -m 6" "$(stat -f%z /tmp/b.webp)" "$(psnr /tmp/b.png)" /tmp/b.png

avifenc -q 18 -s 0 sierra.png /tmp/b.avif >/dev/null 2>&1; avifdec /tmp/b.avif /tmp/b.png >/dev/null 2>&1
emit b_avif avif "AVIF q18 -s 0" "$(stat -f%z /tmp/b.avif)" "$(psnr /tmp/b.png)" /tmp/b.png

emit b_shapes shapes "shape coder, 615 regions" 6488 26.44 render_0615_26.44.png

# ---- Row C: the floor, 153 regions ----
emit c_shapes shapes "shape coder, 153 regions" 2986 24.03 render_0153_24.03.png
cwebp -q 0 -m 6 -quiet sierra.png -o /tmp/c.webp; dwebp -quiet /tmp/c.webp -o /tmp/c.png
emit c_webp webp "WebP q0 -m 6 (its floor)" "$(stat -f%z /tmp/c.webp)" "$(psnr /tmp/c.png)" /tmp/c.png

echo "--- manifest ---"; cat "$M"
echo "--- total embedded weight ---"; du -sh "$OUT"
