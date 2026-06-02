# SCOPE — Infra A v2: Static SPA

## Target
Static single-page application served from CDN. Hashed-filename assets behind long
`Cache-Control: max-age=31536000, immutable`. Build-time encode CPU is acceptable;
runtime encode is irrelevant (no dynamic responses).

## Stack (mocked)
- nginx serving `bench/corpus/assets/` directly
- HTTP/2, TLS 1.3 (assumed)
- Production Brotli + gzip; no zstd at edge

## target_kind
filesystem-only

## Asset classes in scope
- HTML pages (`index.html`, `about.html`)
- JS bundles (`app-*.js`, `vendor-*.js`)
- CSS (`main-*.css`)
- SVG (`logo.svg`)

## Metric

```yaml
metric:
  primary: wire_bytes_p95
  weights:
    encode_cpu_ms: 0.0       # build-time, free
    decode_cpu_ms: 0.5       # mobile decode matters
budget_seconds_per_candidate: 30
client_profile: mobile_4g_slow
target_kind: filesystem-only
```

## Exclusions
None. All assets are public, no secrets, no reflected user input.
BREACH not applicable.
exclusions: []

## Notes
- Versioned JS/CSS pairs are present (`app-3e4f5a6.js` and `app-3e4f5a7.js`,
  `main-7f8a3c2.css` and `main-7f8a3c3.css`, `vendor-a9b8c7d.js` and `vendor-a9b8c7e.js`).
  Strong candidates for shared compression dictionary (RFC 9842) experiments.
- HTML pages share common chrome (header/nav/footer). Possible static-dictionary candidate.
- Live HTTP server is NOT running. target_kind is filesystem-only. Skip Phase 7 (over-the-wire verification).
  Run file-level encode/decode/size benchmarks only.
- Goal: 10 experiments. Cover at minimum:
  1. baseline (identity)
  2. gzip-6
  3. gzip-9
  4. brotli-5 (dynamic baseline)
  5. brotli-11 (static)
  6. zstd-3
  7. zstd-19
  8. shared dictionary brotli (versioned bundle pair)
  9. shared dictionary zstd (versioned bundle pair)
  10. asset-class-specific tuning (e.g. SVG handling, HTML vs JS)
