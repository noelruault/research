#!/usr/bin/env bash
# Usage: encode.sh <algo-level> <input> <output>
# DICT env var required for dict variants.
set -euo pipefail
ALGO="$1"; IN="$2"; OUT="$3"
case "$ALGO" in
  identity)        cat "$IN" > "$OUT" ;;
  gzip-6)          gzip -6  -c "$IN" > "$OUT" ;;
  gzip-9)          gzip -9  -c "$IN" > "$OUT" ;;
  brotli-1)        brotli -q 1  -c "$IN" > "$OUT" ;;
  brotli-4)        brotli -q 4  -c "$IN" > "$OUT" ;;
  brotli-5)        brotli -q 5  -c "$IN" > "$OUT" ;;
  brotli-8)        brotli -q 8  -c "$IN" > "$OUT" ;;
  brotli-11)       brotli -q 11 -c "$IN" > "$OUT" ;;
  zstd-1)          zstd -1  -c -q "$IN" > "$OUT" ;;
  zstd-3)          zstd -3  -c -q "$IN" > "$OUT" ;;
  zstd-9)          zstd -9  -c -q "$IN" > "$OUT" ;;
  zstd-19)         zstd -19 -c -q "$IN" > "$OUT" ;;
  zstd-dict-1)     zstd -1  -D "${DICT:?}" -c -q "$IN" > "$OUT" ;;
  zstd-dict-3)     zstd -3  -D "${DICT:?}" -c -q "$IN" > "$OUT" ;;
  zstd-dict-9)     zstd -9  -D "${DICT:?}" -c -q "$IN" > "$OUT" ;;
  zstd-dict-19)    zstd -19 -D "${DICT:?}" -c -q "$IN" > "$OUT" ;;
  *) echo "unknown algo: $ALGO" >&2; exit 1 ;;
esac
