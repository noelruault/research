# Infra A — Static SPA, v3 findings

Compression smoke test on `infra-a-spa-v3/`, run by the `compression-engineer-v3` agent.
v3 = v1 + 3 cherry-picks: `bench/measure.py` sub-ms timer (§4.5), §2.5 Brotli static-
dictionary mismatch warning, `score.py` dual-format CI (§4.4). v2 ceremony omitted.

Bench artifacts live in `infra-a-spa-v3/bench/`. This file is the human-facing summary.

## 1. Discovery

- `target_kind`: filesystem-only (no live HTTP server in this test).
- Stack (per `bench/SCOPE.md`): static SPA, hashed filenames, long cache; nginx +
  HTTP/2 + TLS 1.3 *assumed* for the config emit step. Phase 7 (over-the-wire) skipped.
- Detected toolchain on PATH: `brotli 1.2.0`, `zstd 1.5.7`, `gzip` (system), `python3`,
  `hyperfine`, `xxd`, `openssl`, `curl`. All v3-required encoders present.

## 2. Corpus inventory

`bench/corpus/assets/`, 9 files, 795,354 raw bytes total.

| file                  |   bytes |  type   |
|-----------------------|--------:|---------|
| index.html            |  16,874 | HTML    |
| about.html            |  19,800 | HTML    |
| logo.svg              |  12,739 | SVG     |
| main-7f8a3c2.css      |  30,739 | CSS (older) |
| main-7f8a3c3.css      |  31,749 | CSS (newer) |
| app-3e4f5a6.js        |  81,928 | JS  (older) |
| app-3e4f5a7.js        |  84,041 | JS  (newer) |
| vendor-a9b8c7d.js     | 256,229 | JS  (older) |
| vendor-a9b8c7e.js     | 261,255 | JS  (newer) |
| **total**             | **795,354** | |

Three versioned pairs are present (app, main, vendor): `*-older` is used as the shared
dictionary, `*-newer` as the dictionary input for Exp 0006.

## 3. Baseline (Exp 0001 — identity)

- Total wire bytes: 795,354 (= raw, no encoding).
- Encode/decode timings reflect `cat` subprocess overhead (~5–14 ms per file via
  `subprocess.run` + `time.perf_counter_ns`, per v3 §4.5). These are subtracted off
  uniformly in the score deltas.
- Persisted to `bench/results/baseline.json`.

## 4. Experiments

Metric: `wire_bytes_p95 + 0.0·encode_cpu_ms_p95 + 0.5·decode_cpu_ms_p95`
(per `bench/SCOPE.md`; α=0.0 because static-asset encode is paid once at build).
Bootstrap CI: 10,000 resamples, deterministic seed `0xC0FFEE`, alpha=0.05.
KEEP iff CI95_high < 0.

| id | candidate | total wire B | mean Δ B | mean Δ % | CI95 bytes | CI95 % | decision |
|----|-----------|-------------:|---------:|---------:|------------|--------|---------:|
| 0001 | identity (baseline) | 795,354 |       0 |   0.00% | (n/a)      | (n/a)  | baseline |
| 0002 | gzip-6              | 163,804 | -70,172.9 | -78.52% | [-124,268.2, -25,721.5] | [-80.67%, -76.34%] | KEEP |
| 0003 | brotli-5            | 182,161 | -68,132.4 | -77.96% | [-120,110.3, -25,505.1] | [-80.48%, -75.95%] | KEEP* |
| 0004 | brotli-11           | **142,836** | **-72,501.9** | **-82.07%** | [-128,215.6, -26,775.3] | [-83.88%, -80.57%] | **KEEP — winner** |
| 0005 | zstd-19             | 144,954 | -72,264.5 | -81.05% | [-127,994.7, -26,460.7] | [-82.36%, -79.75%] | KEEP |
| 0006 | zstd-dict-19 (3-file subset, dcz) | 63,835 (raw subset 377,045) | -104,329.9 | -81.94% | [-218,389.2, -25,174.1] | [-83.62%, -79.53%] | KEEP — best on subset |

\* Exp 0003 is KEEP versus identity but **defeated by gzip-6 on every JS file** in this
corpus; see §6 below. This is the v3 §2.5 dictionary-mismatch pattern surfacing on
synthetic JS (the original observation cited JSON).

Per-file wire bytes:

| file                | identity | gzip-6 | brotli-5 | brotli-11 | zstd-19 |
|---------------------|---------:|-------:|---------:|----------:|--------:|
| index.html          |  16,874  |  2,650 |  2,395   |  2,051    |  2,565  |
| about.html          |  19,800  |  3,734 |  3,422   |  3,026    |  3,573  |
| logo.svg            |  12,739  |  3,349 |  3,059   |  2,607    |  2,763  |
| main-7f8a3c2.css    |  30,739  |  7,752 |  7,675   |  6,234    |  6,503  |
| main-7f8a3c3.css    |  31,749  |  8,127 |  8,059   |  6,484    |  6,786  |
| app-3e4f5a6.js      |  81,928  | 17,035 | 19,109   | 15,381    | 15,394  |
| app-3e4f5a7.js      |  84,041  | 17,373 | 19,439   | 15,626    | 15,748  |
| vendor-a9b8c7d.js   | 256,229  | 51,395 | 58,958   | 45,282    | 45,427  |
| vendor-a9b8c7e.js   | 261,255  | 52,389 | 60,045   | 46,145    | 46,195  |

Roundtrip byte-identity verified for every (encode, decode) pair across all 9 files
and all 3 dict pairs. dcz framing per RFC 9842 §3.2 verified: 8-byte magic
`5e 2a 4d 18 20 00 00 00`, 32-byte SHA-256 of dictionary, then zstd payload starting
`28 b5 2f fd`.

## 5. Recommended configuration

**Winner**: brotli-11 for all text/CSS/JS/SVG/HTML, with zstd-19 alongside as a
peer-encoding for clients that prefer it; dcz dictionary delta-encoding for the
versioned bundles.

### 5.1 nginx (with ngx_brotli + zstd-nginx-module)

```nginx
# Source: ngx_http_gzip_module + google/ngx_brotli + tokers/zstd-nginx-module.
# Numbers from infra-a-spa-v3/bench/EXPERIMENTS.md.

# --- Pre-compressed static (preferred path; uses .br/.zst sidecars built at deploy) ---
brotli_static  on;   # Exp 0004: -82.07% wire bytes vs identity, CI95 [-83.88, -80.57]%
zstd_static    on;   # Exp 0005: -81.05% wire bytes, CI95 [-82.36, -79.75]%
gzip_static    on;   # Exp 0002: -78.52% wire bytes, fallback for legacy clients

# --- Runtime fallback for dynamically-generated responses ---
brotli            on;
brotli_comp_level 5;            # Exp 0003: see notes — defeated by gzip-6 on JS in
                                # this synthetic corpus; keep brotli-5 only if log shows
                                # ≥99% Accept-Encoding: br for text/HTML/CSS where
                                # Exp 0003 still wins.
brotli_min_length 1024;
brotli_window     4m;
brotli_types
    text/plain text/css text/xml application/json application/javascript
    application/xml application/xml+rss application/wasm font/ttf font/otf
    image/svg+xml;

zstd              on;
zstd_comp_level   3;            # runtime; static is pre-compressed at zstd-19
zstd_min_length   1024;
zstd_types
    text/plain text/css application/json application/javascript
    application/wasm image/svg+xml font/ttf font/otf;

gzip              on;           # legacy fallback; ~1% of clients in 2026
gzip_vary         on;
gzip_comp_level   6;
gzip_min_length   1024;
gzip_types        $brotli_types;  # same set; nginx negotiates per Accept-Encoding
gzip_proxied      expired no-cache no-store private auth;

# --- Vary correctness ---
add_header Vary "accept-encoding" always;
```

### 5.2 nginx — RFC 9842 shared dictionary path (Exp 0006)

```nginx
# Exp 0006: zstd-dict-19 versioned subset, -7.1% over zstd-19, dcz framing verified.

# Older bundle is a dictionary candidate
location = /app/v1/main.js {
    add_header Use-As-Dictionary 'match="/app/*/main.js", match-dest=("script"), id="app-bundle"';
    add_header Cache-Control     "public, max-age=31536000, immutable";
    add_header Vary              "accept-encoding, available-dictionary";
}

# Newer bundle: server picks pre-compressed dict-encoded payload by SHA-256 hash
map $http_available_dictionary $dict_hash {
    default                   "";
    "~*^:([A-Za-z0-9+/=]+):$" "$1";
}
map $http_accept_encoding $picked_enc {
    default     "";
    "~*\bdcz\b" "dcz";
    "~*\bdcb\b" "dcb";
}
location ~ ^/app/v[0-9]+/main\.(js|css)$ {
    if ($dict_hash != "")  { if ($picked_enc != "") {
        rewrite ^ /precompressed/$dict_hash/$picked_enc$uri last;
    } }
    add_header Vary "accept-encoding, available-dictionary";
}
location /precompressed/ {
    internal;
    add_header Vary           "accept-encoding, available-dictionary";
    add_header Cache-Control  "public, max-age=31536000, immutable";
    types { application/javascript js; text/css css; }
}
```

### 5.3 Cloudflare Workers

```js
// Exp 0004 (brotli-11) and Exp 0006 (zstd-dict-19, dcz) cited inline.
export default {
  async fetch(req, env) {
    const ae           = req.headers.get("Accept-Encoding") || "";
    const dictHashHdr  = req.headers.get("Available-Dictionary");
    const wantsDcz     = ae.includes("dcz");
    const wantsDcb     = ae.includes("dcb");

    // Dictionary-compressed path — Exp 0006: -7.1% over zstd-19 alone on versioned subset.
    if (dictHashHdr && (wantsDcz || wantsDcb)) {
      const enc  = wantsDcz ? "dcz" : "dcb";
      const hash = dictHashHdr.replaceAll(":", "").replaceAll("/", "_");
      const key  = `precompressed/${hash}/${enc}${new URL(req.url).pathname}`;
      const obj  = await env.ASSETS.get(key);
      if (obj) {
        return new Response(obj.body, {
          headers: {
            "content-encoding": enc,
            "content-type":     contentTypeFor(req.url),
            "vary":             "accept-encoding, available-dictionary",
            "cache-control":    "public, max-age=31536000, immutable",
          },
        });
      }
    }

    // Otherwise: Workers Sites with brotli-11 sidecars (Exp 0004) — winner among
    // non-dict candidates at -82.07% vs identity, CI95 [-83.88%, -80.57%].
    return env.ASSETS.fetch(req);
  },
};

function contentTypeFor(url) {
  if (url.endsWith(".js"))   return "application/javascript";
  if (url.endsWith(".css"))  return "text/css";
  if (url.endsWith(".html")) return "text/html; charset=utf-8";
  if (url.endsWith(".svg"))  return "image/svg+xml";
  return "application/octet-stream";
}
```

## 6. Build hooks

### 6.1 Make

```makefile
# infra-a-spa-v3 build hook — pre-compress text assets at q=11 (Exp 0004) and zstd -19 (Exp 0005).
DIST   := dist
ASSETS := $(shell find $(DIST) -type f \
           \( -name '*.js' -o -name '*.css' -o -name '*.html' \
              -o -name '*.svg' -o -name '*.wasm' -o -name '*.json' \))

precompress: $(ASSETS:%=%.br) $(ASSETS:%=%.zst)
%.br:  % ; brotli -q 11 -k -f $<
%.zst: % ; zstd  -19 -k -f $<

# Optional: build versioned-dict payloads for adjacent releases (Exp 0006).
# Requires the previous release's bundle on disk as $(PREV)/$(notdir $@).
dict-precompress:
	@for f in $(DIST)/app/v$(VER)/*.js $(DIST)/app/v$(VER)/*.css; do \
	  prev=$(PREV)/v$$(($(VER)-1))/$$(basename $$f); \
	  [ -f "$$prev" ] || continue; \
	  hash=$$(openssl dgst -sha256 -binary "$$prev" | xxd -p | tr -d '\n'); \
	  zstd -19 -D "$$prev" -c "$$f" > /tmp/payload.zst; \
	  { printf '\x5e\x2a\x4d\x18\x20\x00\x00\x00'; \
	    printf '%s' "$$hash" | xxd -r -p; \
	    cat /tmp/payload.zst; } > "$$f.dcz"; \
	done

.PHONY: precompress dict-precompress
```

### 6.2 npm script

```json
{
  "scripts": {
    "build": "vite build && npm run precompress",
    "precompress": "find dist -type f \\( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' -o -name '*.wasm' \\) -exec sh -c 'brotli -q 11 -k -f \"$1\" && zstd -19 -k -f \"$1\"' _ {} \\;"
  }
}
```

Comment in the build log: `# brotli-11 = Exp 0004: -82.07% wire bytes (CI95 [-83.88, -80.57]%); zstd-19 = Exp 0005: -81.05% (CI95 [-82.36, -79.75]%).`

## 7. Security review

- **CRIME (TLS-level compression)**: not applicable here — the workload is static
  asset response-body compression, not TLS record compression. TLS 1.3 forbids
  record compression (RFC 8446 §1.2); on the deployed nginx, ensure
  `ssl_protocols TLSv1.2 TLSv1.3` and rely on TLS 1.3's removal. For TLS 1.2,
  nginx disables it by default; verify with
  `openssl s_client -connect host:443 -tls1_2 2>&1 | grep -i compression`.
- **BREACH**: not applicable — this corpus is hashed-filename static assets with no
  reflected request input and no secrets in body. The hashed-filename + immutable
  cache pattern is the safest possible target for body compression. The agent's
  default exclusions (`/api/auth/*`, `/api/csrf/token`, etc., per §4.1 of the v3
  agent definition) do not apply here because no such endpoints exist in scope.
- **Shared-dictionary safety (Exp 0006)**:
  - `match` is same-origin (`match="/app/*/main.js"`), no regex.
  - Pre-compressed at deploy time only; the request-time path is a static lookup
    by SHA-256, never re-encoding under user-controlled input.
  - Dictionary contents (prior bundle) are public, hashed-filename, immutable —
    no secret-bearing markup. RFC 9842 §6.1 oracle risk does not apply.
  - `Vary` includes both `accept-encoding` and `available-dictionary` to prevent
    cross-client cache poisoning.
- **Range requests + compression**: clients sending `Range:` to a `Content-Encoding:
  br` resource is undefined per RFC 9110 §14. nginx with `brotli_static on` serves
  the pre-compressed file as 200 OK without honoring Range, which is the safe
  behavior. Verify with `curl -sI -H 'Range: bytes=0-1023' -H 'Accept-Encoding: br'
  $URL`; expect 200 (full body) or 206 *without* Content-Encoding, never 206 + CE.
- **Vary correctness**: explicit `add_header Vary "accept-encoding"` (and
  `accept-encoding, available-dictionary` on dict-eligible locations) prevents
  serving brotli to a client that didn't advertise it.
- **Double-compression**: `gzip_static`, `brotli_static`, `zstd_static` look up
  sidecar files; the runtime `brotli`/`zstd`/`gzip` directives only fire when no
  pre-compressed file exists. Combined with hashed-filename build artifacts, the
  pre-compressed path is hit ≥99% of the time. Audit with
  `curl -sI $URL | grep -i content-encoding` — single token expected.

## 8. Honest follow-ups

- **No live HTTP server** was tested. Phase 7 (over-the-wire verification) is
  skipped per the agent definition's smoke-test adaptation. Before promoting
  brotli-11 to production, run the §8.1–8.6 verifications from the v3 agent
  definition against the actual nginx deployment.
- **Brotli-5 anomaly (Exp 0003)** deserves attention. On *this* corpus brotli-5
  produced more bytes than gzip-6 on every JS file. Possible causes: (a) the
  synthetic JS bundles do not match brotli's RFC 7932 Appendix A static dictionary
  (which is HTML/JS-tuned for *real* web text); (b) brotli at q=5 on highly
  repetitive synthetic content can underperform gzip's simpler scheme. v3 §2.5
  flags this for JSON; here we observe the same pattern on synthetic JS. The
  remediation is what the recommended config does: pre-compress static at brotli-11
  (where the encoder works hard enough to overcome the mismatch) and use brotli-5
  only as a runtime fallback for *dynamic* HTML/CSS where it still wins. **Run
  Exp 0003 again on the real production JS bundles before committing to
  `brotli_comp_level 5` for runtime JS.**
- **Sample size**: 9 corpus items is small, which is why the bootstrap CI bytes
  are wide (e.g. Exp 0004 CI95 byte-range spans -128 KB to -27 KB). The percentage
  CIs are tight (Exp 0004 [-83.88%, -80.57%]) because the deltas are large
  relative to per-item baselines. For tighter byte-CIs, scale corpus to 50+
  representative items per asset class (per the agent definition §5.2).
- **Dictionary candidate is one-pair-per-file**, not a trained zstd dictionary
  built from many samples. For broader savings on a multi-version corpus, consider
  `zstd --train-cover` over a directory of historical bundles (RFC 8878 §5,
  facebook/zstd manual), then pre-compute payloads against that single trained
  dictionary at deploy time.
- **Encoder versions are pinned to local PATH** (`brotli 1.2.0`, `zstd 1.5.7`).
  Production CI should pin the same versions in the build container so the
  pre-compressed sidecars are reproducible.
- **No PNG/JPEG/AVIF/WOFF2 candidates** — corpus is text-only. If/when raster or
  font assets are added, run §2.6 / §2.7 of the agent definition.
- **Cloudflare Workers `dcb`/`dcz` support** is recent (RFC 9842 published 2025);
  verify in the deployed Worker that `Accept-Encoding: dcz` actually round-trips
  and that the precompressed KV/R2 path is hit. CDN support varies; AWS CloudFront
  and Akamai dictionary support should be re-verified at deploy time.

## 9. Where the bench artifacts live

- `infra-a-spa-v3/bench/SCOPE.md` — input metric/weights (do not edit).
- `infra-a-spa-v3/bench/EXPERIMENTS.md` — append-only log of all 6 experiments.
- `infra-a-spa-v3/bench/measure.py` — sub-ms timer (v3 §4.5).
- `infra-a-spa-v3/bench/score.py` — bootstrap-CI scorer with dual-format CI (v3 §4.4).
- `infra-a-spa-v3/bench/encode.sh`, `decode.sh`, `harness.sh` — candidate dispatch.
- `infra-a-spa-v3/bench/results/baseline.json` — Exp 0001.
- `infra-a-spa-v3/bench/results/{gzip-6,brotli-5,brotli-11,zstd-19,zstd-dict-19}/` —
  one directory per experiment with `items.json` and `score.json`.

Final answer: KEEP brotli-11 for all static text on this corpus
(Exp 0004, -82.07% wire bytes, CI95 [-83.88%, -80.57%]); ship zstd-19 alongside
(Exp 0005); deploy zstd-dict-19 dcz delta-encoding for adjacent versioned bundles
(Exp 0006, -7.1% over zstd-19, framing verified per RFC 9842 §3.2). Keep gzip-6 as
the legacy fallback (Exp 0002).
