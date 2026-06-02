#!/usr/bin/env bash
# Usage: encode.sh <algo-level> <input> <output>
# Per Section 4.4 of compression-engineer.md.
set -euo pipefail
ALGO="$1"
IN="$2"
OUT="$3"

case "$ALGO" in
  identity)        cat "$IN" > "$OUT" ;;
  gzip-6)          gzip -6  -c "$IN" > "$OUT" ;;
  gzip-9)          gzip -9  -c "$IN" > "$OUT" ;;
  brotli-1)        brotli -q 1  -c "$IN" > "$OUT" ;;
  brotli-4)        brotli -q 4  -c "$IN" > "$OUT" ;;
  brotli-5)        brotli -q 5  -c "$IN" > "$OUT" ;;
  brotli-11)       brotli -q 11 -c "$IN" > "$OUT" ;;
  zstd-1)          zstd -q -1  -c "$IN" > "$OUT" ;;
  zstd-3)          zstd -q -3  -c "$IN" > "$OUT" ;;
  zstd-9)          zstd -q -9  -c "$IN" > "$OUT" ;;
  zstd-19)         zstd -q -19 -c "$IN" > "$OUT" ;;
  zstd-dict-3)     zstd -q -3  -D "${DICT:?DICT env var required}" -c "$IN" > "$OUT" ;;
  zstd-dict-9)     zstd -q -9  -D "${DICT:?DICT env var required}" -c "$IN" > "$OUT" ;;
  zstd-dict-19)    zstd -q -19 -D "${DICT:?DICT env var required}" -c "$IN" > "$OUT" ;;
  *) echo "unknown algo: $ALGO" >&2; exit 1 ;;
esac
