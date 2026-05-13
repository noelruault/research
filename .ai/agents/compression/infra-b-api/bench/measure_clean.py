#!/usr/bin/env python3
"""Clean CPU measurement bypassing hyperfine's shell wrapper overhead.

Uses subprocess.run with low-level fork/exec only; no `sh -c` interposed.
Reports per-item encode_p95 / decode_p95 in ms over N runs.

Usage:
    python3 bench/measure_clean.py <algo> <input> [runs=30] [warmup=3]
Output: JSON to stdout: {"encode_ms_p95":..., "decode_ms_p95":...,
                        "encoded_size":..., "encode_times":[...], "decode_times":[...]}
"""
import os
import sys
import subprocess
import time
import tempfile
import json
import statistics
from pathlib import Path


def encode_argv(algo, in_path, out_path):
    if algo == "identity":
        return None  # signal: just copy bytes
    if algo == "gzip-6":
        return ["gzip", "-6", "-c", in_path]
    if algo == "gzip-9":
        return ["gzip", "-9", "-c", in_path]
    if algo == "brotli-1":
        return ["brotli", "-q", "1", "-c", in_path]
    if algo == "brotli-4":
        return ["brotli", "-q", "4", "-c", in_path]
    if algo == "brotli-5":
        return ["brotli", "-q", "5", "-c", in_path]
    if algo == "brotli-11":
        return ["brotli", "-q", "11", "-c", in_path]
    if algo == "zstd-1":
        return ["zstd", "-q", "-1", "-c", in_path]
    if algo == "zstd-3":
        return ["zstd", "-q", "-3", "-c", in_path]
    if algo == "zstd-9":
        return ["zstd", "-q", "-9", "-c", in_path]
    if algo == "zstd-19":
        return ["zstd", "-q", "-19", "-c", in_path]
    if algo == "zstd-dict-3":
        return ["zstd", "-q", "-3", "-D", os.environ["DICT"], "-c", in_path]
    if algo == "zstd-dict-9":
        return ["zstd", "-q", "-9", "-D", os.environ["DICT"], "-c", in_path]
    if algo == "zstd-dict-19":
        return ["zstd", "-q", "-19", "-D", os.environ["DICT"], "-c", in_path]
    raise ValueError(f"unknown algo: {algo}")


def decode_argv(algo, in_path):
    if algo == "identity":
        return None
    if algo.startswith("gzip"):
        return ["gzip", "-d", "-c", in_path]
    if algo.startswith("brotli"):
        return ["brotli", "-d", "-c", in_path]
    if algo == "zstd-dict-3" or algo == "zstd-dict-9" or algo == "zstd-dict-19":
        return ["zstd", "-q", "-d", "-D", os.environ["DICT"], "-c", in_path]
    if algo.startswith("zstd"):
        return ["zstd", "-q", "-d", "-c", in_path]
    raise ValueError(f"unknown algo: {algo}")


def time_proc(argv, in_bytes=None):
    """Run a subprocess, capture stdout to /dev/null (write to a tempfile FD),
    and time wall-clock CPU. Use perf_counter_ns for precision."""
    devnull = subprocess.DEVNULL
    t0 = time.perf_counter_ns()
    p = subprocess.run(argv, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=True)
    t1 = time.perf_counter_ns()
    return (t1 - t0) / 1e6, p.stdout  # ms, encoded bytes


def p95(xs):
    xs = sorted(xs)
    if not xs:
        return 0.0
    return xs[int(round(0.95 * (len(xs) - 1)))]


def main():
    algo = sys.argv[1]
    in_path = sys.argv[2]
    runs = int(sys.argv[3]) if len(sys.argv) > 3 else 30
    warmup = int(sys.argv[4]) if len(sys.argv) > 4 else 3

    enc_argv = encode_argv(algo, in_path, None)
    if enc_argv is None:
        # identity: timing is just the cat-equivalent; skip subprocess and time read
        in_bytes = Path(in_path).read_bytes()
        enc_size = len(in_bytes)
        # fake non-zero
        enc_times = []
        dec_times = []
        for _ in range(warmup + runs):
            t0 = time.perf_counter_ns()
            _ = Path(in_path).read_bytes()
            t1 = time.perf_counter_ns()
            enc_times.append((t1 - t0) / 1e6)
        enc_times = enc_times[warmup:]
        dec_times = enc_times[:]  # identity decode == identity encode
        out = {
            "algo": algo,
            "encoded_size": enc_size,
            "encode_ms_p95": p95(enc_times),
            "decode_ms_p95": p95(dec_times),
            "encode_ms_mean": statistics.mean(enc_times),
            "decode_ms_mean": statistics.mean(dec_times),
            "encode_ms_min": min(enc_times),
            "decode_ms_min": min(dec_times),
            "encode_times": enc_times,
            "decode_times": dec_times,
        }
        print(json.dumps(out))
        return

    # Encode N+warmup times. Capture last encoded payload to use for decode bench.
    enc_times = []
    encoded = None
    for i in range(warmup + runs):
        ms, encoded = time_proc(enc_argv)
        if i >= warmup:
            enc_times.append(ms)

    enc_size = len(encoded)
    # Write encoded to a tempfile for decode bench
    with tempfile.NamedTemporaryFile(delete=False, suffix=".enc") as tf:
        tf.write(encoded)
        enc_path = tf.name

    try:
        dec_argv = decode_argv(algo, enc_path)
        dec_times = []
        for i in range(warmup + runs):
            ms, _ = time_proc(dec_argv)
            if i >= warmup:
                dec_times.append(ms)
    finally:
        os.unlink(enc_path)

    out = {
        "algo": algo,
        "encoded_size": enc_size,
        "encode_ms_p95": p95(enc_times),
        "decode_ms_p95": p95(dec_times),
        "encode_ms_mean": statistics.mean(enc_times),
        "decode_ms_mean": statistics.mean(dec_times),
        "encode_ms_min": min(enc_times),
        "decode_ms_min": min(dec_times),
        "encode_times": enc_times,
        "decode_times": dec_times,
    }
    print(json.dumps(out))


if __name__ == "__main__":
    main()
