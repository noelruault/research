# SCOPE — Infra B v3: JSON API (smoke test)

## Target
Backend JSON API. Responses generated dynamically. Encode CPU paid per request.

## Stack (mocked)
- Go origin, HTTP/2 (assumed)
- gzip currently at origin
- No live server in this test

## Asset classes in scope
JSON of varying size.

## Metric

```yaml
metric:
  primary: score
  weights:
    encode_cpu_ms: 0.5     # encoded per request
    decode_cpu_ms: 0.3     # client decode runtime-paid
budget_seconds_per_candidate: 30
client_profile: cable
score_formula: wire_bytes_p95 + 0.5*encode_cpu_ms_p95 + 0.3*decode_cpu_ms_p95
```

## Exclusions
- Endpoints below `min_compress_size` (1024 B): `error-404.json`, `user-profile.json`.

## Security constraints
- Some endpoints reflect user input. Real BREACH audit is outside this smoke test.
  Treat all corpus items as compressible for the purpose of measurement.

## Notes
- This is a v3 SMOKE TEST. 6 representative experiments to verify v3 works end-to-end.
  The full 10-experiment battery from v1/v2 is not required here.
- Live HTTP server is NOT running. Skip Phase 7.
- Required experiments:
  1. baseline (identity)
  2. gzip-6 (current production)
  3. brotli-1 (must be benched to confirm v3 §2.5 brotli-static-dict-mismatch warning
     reproduces: brotli-1 should produce more wire bytes than gzip-6 on JSON)
  4. zstd-3 (zstd default)
  5. zstd-9 (zstd higher level)
  6. zstd-dict-9 with a dictionary trained on this JSON family
     (`zstd --train bench/corpus/http/*.json -o bench/zstd-dict.bin`)
- Apply v3 §4.5 timing rule: use `bench/measure.py` for sub-ms timings (this is the
  v1 bug fix; do NOT use `hyperfine -- sh -c` for these ~5-15 ms encodes).
