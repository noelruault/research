# SCOPE — Infra A v3: Static SPA (smoke test)

## Target
Static SPA. Hashed-filename assets, long cache. Build-time encode CPU is free.

## Stack (mocked)
- nginx serving `bench/corpus/assets/`
- HTTP/2, TLS 1.3 (assumed)
- No live server in this test

## Asset classes in scope
HTML, JS, CSS, SVG.

## Metric

```yaml
metric:
  primary: wire_bytes_p95
  weights:
    encode_cpu_ms: 0.0
    decode_cpu_ms: 0.5
budget_seconds_per_candidate: 30
client_profile: mobile_4g_slow
```

## Exclusions
None.

## Notes
- This is a v3 SMOKE TEST. Run 6 representative experiments to confirm v3 works
  end-to-end. The full 10-experiment battery from v1/v2 is not required here.
- Live HTTP server is NOT running. Skip Phase 7.
- Required experiments:
  1. baseline (identity)
  2. gzip-6 (production-floor candidate)
  3. brotli-5 (dynamic-friendly brotli)
  4. brotli-11 (static champion expected)
  5. zstd-19 (static-strong zstd)
  6. zstd-dict-19 on the versioned-bundle subset (older as dict, newer as input,
     `dcz` framing per RFC 9842)
- Apply v3 §4.5 timing rule: use `bench/measure.py` for sub-ms timings.
