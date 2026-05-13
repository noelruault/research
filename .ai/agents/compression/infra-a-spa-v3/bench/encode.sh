#!/usr/bin/env bash
# Usage: encode.sh <algo-level> <input> <output>
# v3 §4.4 — candidate dispatch (algo+level → encoded bytes)
set -euo pipefail
case "$1" in
  identity)      cat "$2" > "$3" ;;
  gzip-6)        gzip -6  -c "$2" > "$3" ;;
  gzip-9)        gzip -9  -c "$2" > "$3" ;;
  brotli-1)      brotli -q 1  -c "$2" > "$3" ;;
  brotli-5)      brotli -q 5  -c "$2" > "$3" ;;
  brotli-11)     brotli -q 11 -c "$2" > "$3" ;;
  zstd-1)        zstd -1  -c "$2" > "$3" ;;
  zstd-3)        zstd -3  -c "$2" > "$3" ;;
  zstd-19)       zstd -19 -c "$2" > "$3" ;;
  zstd-22)       zstd --ultra -22 -c "$2" > "$3" ;;
  zstd-dict-19)  zstd -19 -D "${DICT:?}" -c "$2" > "$3" ;;
  zstd-dict-3)   zstd -3  -D "${DICT:?}" -c "$2" > "$3" ;;
  zstd-dict-9)   zstd -9  -D "${DICT:?}" -c "$2" > "$3" ;;
  brotli-dict-11) brotli -q 11 -D "${DICT:?}" -c "$2" > "$3" ;;
  *) echo "unknown algo: $1" >&2; exit 1 ;;
esac
