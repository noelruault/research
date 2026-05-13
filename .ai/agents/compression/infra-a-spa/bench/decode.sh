#!/usr/bin/env bash
# Usage: decode.sh <algo-level> <input>
# Decompresses to /dev/null. Used by hyperfine to measure decode CPU.
# Dictionary candidates require DICT=<path>.
set -euo pipefail
case "$1" in
  identity)        cat "$2" > /dev/null ;;
  gzip-*)          gzip   -d -c "$2" > /dev/null ;;
  brotli-1|brotli-5|brotli-8|brotli-11|brotli-11-w24)
                   brotli -d -c "$2" > /dev/null ;;
  zstd-1|zstd-3|zstd-9|zstd-19|zstd-22)
                   zstd   -d -c "$2" 2>/dev/null > /dev/null ;;
  zstd-dict-19)    zstd   -d -D "${DICT:?need DICT=<path>}" -c "$2" 2>/dev/null > /dev/null ;;
  brotli-dict-11)  brotli -d -D "${DICT:?need DICT=<path>}" -c "$2" > /dev/null ;;
  *) echo "unknown algo: $1" >&2; exit 1 ;;
esac
