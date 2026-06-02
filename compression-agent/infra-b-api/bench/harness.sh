#!/usr/bin/env bash
# bench/harness.sh — file-level harness adapted to no-live-server testing.
# Per Section 5.5 of compression-engineer.md.
#
# Usage:
#   ./bench/harness.sh <experiment-id> <algo-level> [include-tiny=0|1]
# Example:
#   ./bench/harness.sh 0002 gzip-6
#   DICT=bench/zstd-dict.bin ./bench/harness.sh 0009 zstd-dict-3
#
# Notes:
# - Default excludes corpus items below the 1024 B threshold (per SCOPE.md).
# - Pass include-tiny=1 to include them (used for threshold sweep).
# - We invoke encoders DIRECTLY through hyperfine (--shell=none) to avoid
#   bash wrapper overhead (~10 ms) polluting CPU numbers. encode.sh /
#   decode.sh remain the canonical dispatch shape used elsewhere.
set -euo pipefail

EXP_ID="${1:?experiment id required}"
ALGO="${2:?algo-level required}"
INCLUDE_TINY="${3:-0}"
WARMUP="${WARMUP:-3}"
RUNS="${RUNS:-20}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CORPUS="$ROOT/bench/corpus/http"
OUT="$ROOT/bench/results/$EXP_ID"
mkdir -p "$OUT/encoded"

# Resolve direct encoder/decoder commands (bypass wrapper for clean CPU numbers).
encode_cmd() {  # args: algo input output
  local algo="$1" in="$2" out="$3"
  case "$algo" in
    identity)        echo "cp $in $out" ;;
    gzip-6)          echo "sh -c 'gzip -6 -c \"$in\" > \"$out\"'" ;;
    gzip-9)          echo "sh -c 'gzip -9 -c \"$in\" > \"$out\"'" ;;
    brotli-1)        echo "sh -c 'brotli -q 1 -c \"$in\" > \"$out\"'" ;;
    brotli-4)        echo "sh -c 'brotli -q 4 -c \"$in\" > \"$out\"'" ;;
    brotli-5)        echo "sh -c 'brotli -q 5 -c \"$in\" > \"$out\"'" ;;
    brotli-11)       echo "sh -c 'brotli -q 11 -c \"$in\" > \"$out\"'" ;;
    zstd-1)          echo "sh -c 'zstd -q -1 -c \"$in\" > \"$out\"'" ;;
    zstd-3)          echo "sh -c 'zstd -q -3 -c \"$in\" > \"$out\"'" ;;
    zstd-9)          echo "sh -c 'zstd -q -9 -c \"$in\" > \"$out\"'" ;;
    zstd-19)         echo "sh -c 'zstd -q -19 -c \"$in\" > \"$out\"'" ;;
    zstd-dict-3)     echo "sh -c 'zstd -q -3 -D \"$DICT\" -c \"$in\" > \"$out\"'" ;;
    zstd-dict-9)     echo "sh -c 'zstd -q -9 -D \"$DICT\" -c \"$in\" > \"$out\"'" ;;
    zstd-dict-19)    echo "sh -c 'zstd -q -19 -D \"$DICT\" -c \"$in\" > \"$out\"'" ;;
    *) echo "unknown algo: $algo" >&2; return 1 ;;
  esac
}
decode_cmd() {  # args: algo input
  local algo="$1" in="$2"
  case "$algo" in
    identity)        echo "sh -c 'cat \"$in\" > /dev/null'" ;;
    gzip-6|gzip-9)   echo "sh -c 'gzip -d -c \"$in\" > /dev/null'" ;;
    brotli-1|brotli-4|brotli-5|brotli-11) echo "sh -c 'brotli -d -c \"$in\" > /dev/null'" ;;
    zstd-1|zstd-3|zstd-9|zstd-19) echo "sh -c 'zstd -q -d -c \"$in\" > /dev/null'" ;;
    zstd-dict-3|zstd-dict-9|zstd-dict-19) echo "sh -c 'zstd -q -d -D \"$DICT\" -c \"$in\" > /dev/null'" ;;
    *) echo "unknown algo: $algo" >&2; return 1 ;;
  esac
}

# Build corpus list (filter tiny < 1024 B per SCOPE.md unless overridden).
ITEMS=()
for f in "$CORPUS"/*.json; do
  size=$(wc -c < "$f")
  if [ "$INCLUDE_TINY" = "1" ] || [ "$size" -ge 1024 ]; then
    ITEMS+=("$(basename "$f")")
  fi
done

echo "[harness] exp=$EXP_ID algo=$ALGO items=${#ITEMS[@]} include_tiny=$INCLUDE_TINY"

# 1) Encode each item with hyperfine (per-item CPU timing).
for name in "${ITEMS[@]}"; do
  in="$CORPUS/$name"
  enc_out="$OUT/encoded/$name.enc"
  cmd=$(encode_cmd "$ALGO" "$in" "$enc_out")
  # Pre-create the file once so the bench just measures encode work.
  eval "$cmd"
  hyperfine \
    --warmup "$WARMUP" --runs "$RUNS" \
    --time-unit millisecond \
    --shell=none \
    --export-json "$OUT/encode-$name.json" \
    "$cmd" \
    > /dev/null 2>&1 || {
      # Some hyperfine builds need shell because of pipes; fall back.
      hyperfine \
        --warmup "$WARMUP" --runs "$RUNS" \
        --time-unit millisecond \
        --export-json "$OUT/encode-$name.json" \
        "$cmd" > /dev/null
    }
done

# 2) Decode each compressed item with hyperfine.
for name in "${ITEMS[@]}"; do
  enc_out="$OUT/encoded/$name.enc"
  cmd=$(decode_cmd "$ALGO" "$enc_out")
  hyperfine \
    --warmup "$WARMUP" --runs "$RUNS" \
    --time-unit millisecond \
    --shell=none \
    --export-json "$OUT/decode-$name.json" \
    "$cmd" \
    > /dev/null 2>&1 || {
      hyperfine \
        --warmup "$WARMUP" --runs "$RUNS" \
        --time-unit millisecond \
        --export-json "$OUT/decode-$name.json" \
        "$cmd" > /dev/null
    }
done

# 3) Aggregate into items.json keyed by filename.
python3 - "$OUT" "${ITEMS[@]}" <<'PYEOF'
import json, sys
from pathlib import Path

OUT = Path(sys.argv[1])
ITEMS = sys.argv[2:]

def p95(xs):
    if not xs: return 0.0
    xs = sorted(xs)
    idx = max(0, int(round(0.95 * (len(xs) - 1))))
    return xs[idx]

items = {}
for name in ITEMS:
    enc = json.loads((OUT / f"encode-{name}.json").read_text())["results"][0]
    dec = json.loads((OUT / f"decode-{name}.json").read_text())["results"][0]
    enc_times_ms = [t * 1000.0 for t in enc["times"]]
    dec_times_ms = [t * 1000.0 for t in dec["times"]]
    enc_path = OUT / "encoded" / f"{name}.enc"
    items[name] = {
        "wire_bytes_p95": enc_path.stat().st_size,
        "encode_cpu_ms_p95": p95(enc_times_ms),
        "decode_cpu_ms_p95": p95(dec_times_ms),
        "encode_cpu_ms_mean": enc["mean"] * 1000.0,
        "decode_cpu_ms_mean": dec["mean"] * 1000.0,
        "encode_cpu_ms_min": enc["min"] * 1000.0,
        "decode_cpu_ms_min": dec["min"] * 1000.0,
    }

(OUT / "items.json").write_text(json.dumps(items, indent=2))
print(f"[harness] wrote {OUT}/items.json with {len(items)} items")
PYEOF

# 4) If baseline exists and this is not the baseline run, score against it.
BASELINE="$ROOT/bench/results/baseline.json"
if [ -f "$BASELINE" ] && [ "$EXP_ID" != "0001" ]; then
  echo "[harness] scoring vs baseline..."
  python3 "$ROOT/bench/score.py" "$BASELINE" "$OUT/items.json" \
    | tee "$OUT/score.json"
fi
