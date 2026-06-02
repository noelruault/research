#!/usr/bin/env bash
# Usage: decode.sh <algo-level> <input> <output>
# DICT env var required for dict variants.
set -euo pipefail
ALGO="$1"; IN="$2"; OUT="${3:-/dev/null}"
case "$ALGO" in
  identity)                                        cat "$IN" > "$OUT" ;;
  gzip-*)                                          gzip   -d -c "$IN" > "$OUT" ;;
  brotli-dict-*)                                   brotli -d -D "${DICT:?}" -c "$IN" > "$OUT" ;;
  brotli-*)                                        brotli -d -c "$IN" > "$OUT" ;;
  zstd-dict-*)                                     zstd   -d -D "${DICT:?}" -c -q "$IN" > "$OUT" ;;
  zstd-*)                                          zstd   -d -c -q "$IN" > "$OUT" ;;
  *) echo "unknown algo: $ALGO" >&2; exit 1 ;;
esac
