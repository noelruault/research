#!/usr/bin/env bash
# Usage: decode.sh <algo-level> <input>
# Output goes to stdout; consumers redirect to /dev/null for CPU timing.
set -euo pipefail
ALGO="$1"
IN="$2"

case "$ALGO" in
  identity)        cat "$IN" ;;
  gzip-6|gzip-9)   gzip -d -c "$IN" ;;
  brotli-1|brotli-4|brotli-5|brotli-11) brotli -d -c "$IN" ;;
  zstd-1|zstd-3|zstd-9|zstd-19) zstd -q -d -c "$IN" ;;
  zstd-dict-3|zstd-dict-9|zstd-dict-19) zstd -q -d -D "${DICT:?DICT env var required}" -c "$IN" ;;
  *) echo "unknown algo: $ALGO" >&2; exit 1 ;;
esac
