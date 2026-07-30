#!/usr/bin/env bash
# Report 34 — where our bytes go, the affine re-run (P-08), and the wall-variant sweep.
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
build_lab
SRC="${SRC:?set SRC=/path/to/image.png}"
REGIONS="${REGIONS:-1200}"
mkdir -p "$OUT/34"
"$LAB" hd "$SRC" "$OUT/34/ss" 2>&1 | tee "$OUT/34/scalespace.txt" | grep -E "^[0-9]" || true
R=$(mark_nearest "$OUT/34/ss" "$REGIONS")

echo; echo "=== 1. component split of a real file ==="
"$LAB" p4enc "$R" "$OUT/34/x.shpc"

echo; echo "=== 2. P-08 affine colour, the parked lever whose trigger fired when RCT landed ==="
"$LAB" affine "$SRC"

echo; echo "=== 3. every wall-coder variant, priced on this exact partition ==="
"$LAB" wallxexact "$R"
