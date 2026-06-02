#!/usr/bin/env bash
# Usage: decode.sh <algo-level> <input> <output>
set -euo pipefail
case "$1" in
  identity)      cat "$2" > "$3" ;;
  gzip-*)        gzip   -d -c "$2" > "$3" ;;
  brotli-*)      brotli -d -c "$2" > "$3" ;;
  zstd-dict-*)   zstd -d -D "${DICT:?}" -c "$2" > "$3" ;;
  zstd-*)        zstd   -d -c "$2" > "$3" ;;
  *) echo "unknown algo: $1" >&2; exit 1 ;;
esac
