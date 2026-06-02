#!/usr/bin/env bash
# Usage: encode.sh <algo-level> <input> <output>
set -euo pipefail
case "$1" in
  identity)      cat        "$2" > "$3" ;;
  gzip-6)        gzip -6  -c "$2" > "$3" ;;
  gzip-9)        gzip -9  -c "$2" > "$3" ;;
  brotli-1)      brotli -q 1  -c "$2" > "$3" ;;
  brotli-4)      brotli -q 4  -c "$2" > "$3" ;;
  brotli-5)      brotli -q 5  -c "$2" > "$3" ;;
  brotli-8)      brotli -q 8  -c "$2" > "$3" ;;
  brotli-11)     brotli -q 11 -c "$2" > "$3" ;;
  zstd-1)        zstd -1  -q -c "$2" > "$3" ;;
  zstd-3)        zstd -3  -q -c "$2" > "$3" ;;
  zstd-9)        zstd -9  -q -c "$2" > "$3" ;;
  zstd-19)       zstd -19 -q -c "$2" > "$3" ;;
  zstd-22)       zstd --ultra -22 -q -c "$2" > "$3" ;;
  zstd-dict-3)   zstd -3  -q -D "${DICT:?}" -c "$2" > "$3" ;;
  zstd-dict-19)  zstd -19 -q -D "${DICT:?}" -c "$2" > "$3" ;;
  brotli-dict-5)  brotli -q 5  -D "${DICT:?}" -c "$2" > "$3" ;;
  brotli-dict-11) brotli -q 11 -D "${DICT:?}" -c "$2" > "$3" ;;
  *) echo "unknown algo: $1" >&2; exit 1 ;;
esac
