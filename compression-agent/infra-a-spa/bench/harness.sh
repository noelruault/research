#!/usr/bin/env bash
# bench/harness.sh — file-level compression bench.
#
# Usage: harness.sh <candidate-id> [<algo-for-encode>]
#   harness.sh brotli-5
#   harness.sh brotli-dict-11-versioned brotli-dict-11   # alias
#
# Adapted from agent definition Section 5.5 for file-level operation
# (no live HTTP server in this test environment, per SCOPE.md).
#
# For each file in bench/corpus/assets/:
#   1. encode N times (hyperfine), record p95 encode_cpu_ms
#   2. measure encoded size (wire_bytes_p95 == encoded byte count; deterministic)
#   3. decode the encoded artifact N times, record p95 decode_cpu_ms
#
# Emits:
#   bench/results/<candidate>/encode-<basename>.json
#   bench/results/<candidate>/decode-<basename>.json
#   bench/results/<candidate>/<basename>.enc
#   bench/results/<candidate>/items.json   (keyed by basename)
#
# Dictionary mode: if DICT_MAP is set (newline-separated INPUT=DICT pairs),
# each input uses its mapped dictionary. Required for shared-dictionary
# (RFC 9842) experiments.

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CAND="$1"
ALGO="${2:-$CAND}"            # candidate label may differ from encode.sh algo
RUNS="${RUNS:-10}"
WARMUP="${WARMUP:-2}"

OUT="bench/results/$CAND"
mkdir -p "$OUT"

# Optional: path-keyed dict mapping. Format: "<basename>=<dict-path>" per line.
declare -A DICTS=()
if [[ -n "${DICT_MAP:-}" ]]; then
  while IFS='=' read -r k v; do
    [[ -z "$k" ]] && continue
    DICTS["$k"]="$v"
  done <<< "$DICT_MAP"
fi

CORPUS_DIR="bench/corpus/assets"
ITEMS_JSON="$OUT/items.json"
echo "{" > "$ITEMS_JSON"
first=1

for src in "$CORPUS_DIR"/*; do
  base="$(basename "$src")"
  enc="$OUT/$base.enc"

  # Pick dictionary if mapped
  if [[ -n "${DICTS[$base]:-}" ]]; then
    DICT_FOR_FILE="${DICTS[$base]}"
    export DICT="$DICT_FOR_FILE"
  else
    unset DICT || true
  fi

  # 1. Encode + measure encode CPU
  enc_json="$OUT/encode-$base.json"
  if [[ -n "${DICT:-}" ]]; then
    hyperfine --warmup "$WARMUP" --runs "$RUNS" --time-unit millisecond \
      --export-json "$enc_json" \
      "DICT='$DICT' bench/encode.sh $ALGO '$src' '$enc'" \
      >/dev/null
  else
    hyperfine --warmup "$WARMUP" --runs "$RUNS" --time-unit millisecond \
      --export-json "$enc_json" \
      "bench/encode.sh $ALGO '$src' '$enc'" \
      >/dev/null
  fi

  # 2. Encoded size (deterministic; this is the wire byte cost)
  size=$(stat -f%z "$enc" 2>/dev/null || stat -c%s "$enc")

  # 3. Decode CPU
  dec_json="$OUT/decode-$base.json"
  if [[ -n "${DICT:-}" ]]; then
    hyperfine --warmup "$WARMUP" --runs "$RUNS" --time-unit millisecond \
      --export-json "$dec_json" \
      "DICT='$DICT' bench/decode.sh $ALGO '$enc'" \
      >/dev/null
  else
    hyperfine --warmup "$WARMUP" --runs "$RUNS" --time-unit millisecond \
      --export-json "$dec_json" \
      "bench/decode.sh $ALGO '$enc'" \
      >/dev/null
  fi

  # Aggregate via python (compute p95 from samples in ms)
  py_out=$(python3 - "$enc_json" "$dec_json" <<'PY'
import json, sys, statistics
def p95(xs):
    if not xs: return 0.0
    xs = sorted(xs)
    # nearest-rank p95
    k = max(0, int(round(0.95 * (len(xs) - 1))))
    return xs[k]
e = json.loads(open(sys.argv[1]).read())["results"][0]["times"]
d = json.loads(open(sys.argv[2]).read())["results"][0]["times"]
# hyperfine times are seconds; convert to ms
e_ms = [t*1000.0 for t in e]
d_ms = [t*1000.0 for t in d]
print(json.dumps({"encode_p95_ms": p95(e_ms), "decode_p95_ms": p95(d_ms)}))
PY
)

  enc_p95=$(echo "$py_out" | python3 -c "import json,sys;print(json.load(sys.stdin)['encode_p95_ms'])")
  dec_p95=$(echo "$py_out" | python3 -c "import json,sys;print(json.load(sys.stdin)['decode_p95_ms'])")

  if [[ $first -eq 1 ]]; then first=0; else echo "," >> "$ITEMS_JSON"; fi
  printf '  "%s": {"wire_bytes_p95": %d, "encode_cpu_ms_p95": %s, "decode_cpu_ms_p95": %s}' \
    "$base" "$size" "$enc_p95" "$dec_p95" >> "$ITEMS_JSON"
done
echo "" >> "$ITEMS_JSON"
echo "}" >> "$ITEMS_JSON"

echo "Wrote $ITEMS_JSON"
