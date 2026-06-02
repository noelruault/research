# SCOPE — Infra B v2: JSON API origin

## Target
Backend JSON API behind a load balancer. Responses generated dynamically per request.
Encode CPU is paid per request and must be bounded. Pre-compression is not viable for
most endpoints (responses vary per user / query).

## Stack (mocked)
- Go origin with HTTP/2 (assumed)
- Currently gzip only at origin
- No CDN compression layer

## target_kind
filesystem-only

## Asset classes in scope
- JSON responses of varying size:
  - tiny (`error-404.json`, `user-profile.json`)
  - medium (`search-results*.json`, `order-history.json`, `notifications.json`)
  - large (`products-list*.json`, `catalog-full.json`)

## Metric

```yaml
metric:
  primary: score
  weights:
    encode_cpu_ms: 0.5       # encoded per request
    decode_cpu_ms: 0.3       # client decode also runtime-paid
budget_seconds_per_candidate: 30
client_profile: cable
target_kind: filesystem-only
score_formula: wire_bytes_p95 + 0.5*encode_cpu_ms_p95 + 0.3*decode_cpu_ms_p95
```

## Exclusions
- Endpoints below `min_compress_size` (1024 B): `error-404.json`, `user-profile.json`.
  Compression overhead exceeds savings.
exclusions:
  - "*/error-404.json"
  - "*/user-profile.json"

## Security constraints
- Assume some endpoints reflect user input (search query, order filters). Do not enable
  brotli/zstd on endpoints that could combine reflected input + secrets (CSRF token,
  session, PII) without an explicit BREACH-mitigation plan.
- For benchmark purposes here, treat all corpus items as compressible. In a real
  deployment, audit per endpoint.

## Notes
- Repetitive JSON structure (same keys across product/order objects) strongly favors
  shared dictionaries for the JSON family. Test `zstd --train` candidate.
- Versioned pairs available (`products-list.json` vs `products-list-v2.json`,
  `search-results.json` vs `search-results-v2.json`) for dictionary testing.
- Live HTTP server is NOT running. target_kind is filesystem-only. Skip Phase 7. File-level only.
- Goal: 10 experiments. Cover at minimum:
  1. baseline (identity)
  2. gzip-6 (current state)
  3. gzip-9
  4. brotli-1 (dynamic ultra-fast)
  5. brotli-5
  6. zstd-1
  7. zstd-3
  8. zstd-9
  9. zstd with trained dictionary (`zstd --train` on JSON corpus)
  10. min-size threshold sweep (compare strategies excluding tiny endpoints)
