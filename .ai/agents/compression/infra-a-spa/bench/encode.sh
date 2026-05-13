#!/usr/bin/env bash
# Usage: encode.sh <algo-level> <input> <output>
# Dispatches a single encoder invocation for the named (algo, level) candidate.
# Dictionary candidates require DICT=<path> to be set in the environment.
# See agent definition Section 4.4 for the canonical encode dispatch table.
set -euo pipefail
case "$1" in
  identity)        cp "$2" "$3" ;;
  gzip-6)          gzip -6  -c "$2" > "$3" ;;
  gzip-9)          gzip -9  -c "$2" > "$3" ;;
  brotli-1)        brotli -q 1  -c "$2" > "$3" ;;
  brotli-5)        brotli -q 5  -c "$2" > "$3" ;;
  brotli-8)        brotli -q 8  -c "$2" > "$3" ;;
  brotli-11)       brotli -q 11 -c "$2" > "$3" ;;
  brotli-11-w24)   brotli -q 11 --lgwin=24 -c "$2" > "$3" ;;
  zstd-1)          zstd -1  -c "$2" 2>/dev/null > "$3" ;;
  zstd-3)          zstd -3  -c "$2" 2>/dev/null > "$3" ;;
  zstd-9)          zstd -9  -c "$2" 2>/dev/null > "$3" ;;
  zstd-19)         zstd -19 -c "$2" 2>/dev/null > "$3" ;;
  zstd-22)         zstd --ultra -22 -c "$2" 2>/dev/null > "$3" ;;
  zstd-dict-19)    zstd -19 -D "${DICT:?need DICT=<path>}" -c "$2" 2>/dev/null > "$3" ;;
  brotli-dict-11)
      # Raw shared-dictionary encode. Wrap in dcb framing separately.
      brotli -q 11 -D "${DICT:?need DICT=<path>}" -c "$2" > "$3" ;;
  *) echo "unknown algo: $1" >&2; exit 1 ;;
esac
