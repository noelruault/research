# Shared paths and guards for the run scripts. Source it, do not execute it.
#
# Every script here is bash, not zsh: AUTORESEARCH.md's invariants say so, and the one time a loop
# was written for zsh its `set -- $pair` split wrong and produced a table of zeros.
set -euo pipefail

LAB_DIR="${LAB_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../lab" && pwd)}"
LAB="${LAB:-$LAB_DIR/lab}"
OUT="${OUT:-${TMPDIR:-/tmp}/shpc-runs}"
SPRITES="${SPRITES:-$HOME/go/src/github.com/noelruault/sprites}"

build_lab() {
  ( cd "$LAB_DIR" && go build -o "$LAB" . )
}

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing tool: $1" >&2; exit 127; }
}

# to_png normalises any input (JPEG, PNG) into a PNG all arms read identically.
# Every comparison in this study decodes the source ONCE and hands the same file to every arm.
to_png() {
  local src="$1" dst="$2"
  if [ "${src##*.}" = "png" ]; then cp "$src" "$dst"; else sips -s format png "$src" --out "$dst" >/dev/null; fi
}

# mark_nearest echoes the hd render whose region count is closest to $2.
mark_nearest() {
  local dir="$1" want="$2" best="" bd=999999999
  for f in "$dir"/hd_*.png; do
    local n=$(basename "$f" .png); n=${n#hd_}; n=$((10#$n))
    local d=$(( n > want ? n - want : want - n ))
    if [ $d -lt $bd ]; then bd=$d; best="$f"; fi
  done
  echo "$best"
}
