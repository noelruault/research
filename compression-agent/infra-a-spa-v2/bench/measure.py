#!/usr/bin/env python3
"""Canonical sub-millisecond CPU timer for compression experiments. v2.

Times encode and decode of one corpus item N times via subprocess.run.
Emits per-item p50/p95/p99 latency in microseconds and resulting wire bytes.
"""
import json, sys, subprocess, time, statistics, os, hashlib
from pathlib import Path


def percentile(xs, p):
    xs = sorted(xs)
    if not xs:
        return 0.0
    k = (len(xs) - 1) * p / 100.0
    f = int(k)
    c = min(f + 1, len(xs) - 1)
    return xs[f] + (xs[c] - xs[f]) * (k - f)


def time_one(cmd, stdin_bytes=None, runs=30, warmup=3):
    timings = []
    out_size = 0
    last_stdout = b""
    for i in range(warmup + runs):
        t0 = time.perf_counter_ns()
        r = subprocess.run(cmd, input=stdin_bytes, capture_output=True, check=True)
        t1 = time.perf_counter_ns()
        if i >= warmup:
            timings.append((t1 - t0) / 1000.0)  # microseconds
            out_size = len(r.stdout)
            last_stdout = r.stdout
    return timings, out_size, last_stdout


def measure_item(item_path, encode_cmd, decode_cmd, runs=30):
    raw = Path(item_path).read_bytes()
    enc_us, wire_bytes, encoded = time_one(encode_cmd, stdin_bytes=raw, runs=runs)
    dec_us, _, _ = time_one(decode_cmd, stdin_bytes=encoded, runs=runs)
    return {
        "raw_bytes": len(raw),
        "wire_bytes": wire_bytes,
        "wire_bytes_p95": wire_bytes,  # static deterministic encode
        "encode_cpu_ms_p50": percentile(enc_us, 50) / 1000.0,
        "encode_cpu_ms_p95": percentile(enc_us, 95) / 1000.0,
        "encode_cpu_ms_p99": percentile(enc_us, 99) / 1000.0,
        "decode_cpu_ms_p50": percentile(dec_us, 50) / 1000.0,
        "decode_cpu_ms_p95": percentile(dec_us, 95) / 1000.0,
        "decode_cpu_ms_p99": percentile(dec_us, 99) / 1000.0,
        "raw_sha256": hashlib.sha256(raw).hexdigest(),
        "n_runs": runs,
    }


RECIPES = {
    "identity":  (["cat"],                              ["cat"]),
    "gzip-6":    (["gzip", "-6", "-c"],                 ["gzip", "-d", "-c"]),
    "gzip-9":    (["gzip", "-9", "-c"],                 ["gzip", "-d", "-c"]),
    "brotli-1":  (["brotli", "-q", "1", "-c"],          ["brotli", "-d", "-c"]),
    "brotli-4":  (["brotli", "-q", "4", "-c"],          ["brotli", "-d", "-c"]),
    "brotli-5":  (["brotli", "-q", "5", "-c"],          ["brotli", "-d", "-c"]),
    "brotli-8":  (["brotli", "-q", "8", "-c"],          ["brotli", "-d", "-c"]),
    "brotli-11": (["brotli", "-q", "11", "-c"],         ["brotli", "-d", "-c"]),
    "zstd-1":    (["zstd", "-1", "-c", "-q"],           ["zstd", "-d", "-c", "-q"]),
    "zstd-3":    (["zstd", "-3", "-c", "-q"],           ["zstd", "-d", "-c", "-q"]),
    "zstd-9":    (["zstd", "-9", "-c", "-q"],           ["zstd", "-d", "-c", "-q"]),
    "zstd-19":   (["zstd", "-19", "-c", "-q"],          ["zstd", "-d", "-c", "-q"]),
    "zstd-22":   (["zstd", "--ultra", "-22", "-c", "-q"], ["zstd", "-d", "-c", "-q"]),
}


def main():
    """Usage: measure.py <algo> <corpus_dir> <out_items.json>
    Env: DICT=<dict_path> (for dict variants), RUNS=<n>, ITEMS=<comma_sep_filter>
    """
    algo, corpus_dir, out_path = sys.argv[1], sys.argv[2], sys.argv[3]
    dict_path = os.environ.get("DICT")
    runs = int(os.environ.get("RUNS", "30"))
    item_filter = os.environ.get("ITEMS")
    filter_set = set(item_filter.split(",")) if item_filter else None

    if algo.startswith("zstd-dict-") or algo.startswith("brotli-dict-"):
        if not dict_path:
            print("error: DICT env var required for dict variants", file=sys.stderr)
            sys.exit(2)
        if algo.startswith("zstd-dict-"):
            level = algo.rsplit("-", 1)[1]
            enc = ["zstd", f"-{level}", "-D", dict_path, "-c", "-q"]
            dec = ["zstd", "-d", "-D", dict_path, "-c", "-q"]
        elif algo.startswith("brotli-dict-"):
            level = algo.rsplit("-", 1)[1]
            enc = ["brotli", "-q", level, "-D", dict_path, "-c"]
            dec = ["brotli", "-d", "-D", dict_path, "-c"]
    else:
        enc, dec = RECIPES[algo]

    items = {}
    for p in sorted(Path(corpus_dir).iterdir()):
        if not p.is_file():
            continue
        if filter_set and p.name not in filter_set:
            continue
        items[p.name] = measure_item(p, enc, dec, runs=runs)
    Path(out_path).write_text(json.dumps(items, indent=2))
    print(json.dumps({"algo": algo, "n_items": len(items), "out": out_path}))


if __name__ == "__main__":
    main()
