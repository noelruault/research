#!/usr/bin/env bash
# Usage: decode.sh <algo-level> <input> [output]
set -euo pipefail
out="${3:-/dev/stdout}"
case "$1" in
  identity)              cat "$2" > "$out" ;;
  gzip-*)                gzip -d -c "$2" > "$out" ;;
  brotli-*|brotli-dict-*) if [[ "$1" == brotli-dict-* ]]; then
                           brotli -d -D "${DICT:?}" -c "$2" > "$out"
                         else
                           brotli -d -c "$2" > "$out"
                         fi ;;
  zstd-*|zstd-dict-*)    if [[ "$1" == zstd-dict-* ]]; then
                           zstd -d -q -D "${DICT:?}" -c "$2" > "$out"
                         else
                           zstd -d -q -c "$2" > "$out"
                         fi ;;
  *) echo "unknown algo: $1" >&2; exit 1 ;;
esac
