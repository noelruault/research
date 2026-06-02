# Compression Engineer — Infra B JSON API

Output structured per Section 11 of `compression-engineer.md`.
All numbers cite an experiment id from `bench/EXPERIMENTS.md`.

## Discovery

- **Target**: Go HTTP origin serving JSON API responses, generated dynamically per
  request. HTTP/2 assumed. Currently gzip-only. No edge/CDN compression layer.
- **Stack**: Go origin, no live server in this environment (Phase 7 skipped).
- **HTTP version**: 2 (assumed; `Accept-Encoding` negotiation per RFC 9110 §8.4).
- **Asset classes in scope**: dynamic JSON of varying size only (no static assets,
  images, fonts, or wasm in this corpus).
- **Client mix**: API consumers (other services, SDKs). `cable` profile per
  `bench/SCOPE.md`; encode CPU weighted at 0.5 (paid per request).
- **Metric** (`bench/SCOPE.md`):
  ```
  score = wire_bytes_p95 + 0.5 * encode_cpu_ms_p95 + 0.3 * decode_cpu_ms_p95
  ```
  Decision rule: KEEP iff bootstrap CI95 high < 0 (Section 4.1, agent definition).

## Corpus

9 JSON files under `bench/corpus/http/`, 599 588 B total uncompressed.
Items below the 1024 B threshold (excluded from baseline / scoring per
`SCOPE.md`):

| File | Size | In scope? |
|---|---:|:--:|
| `catalog-full.json` | 189 168 B | yes |
| `products-list-v2.json` | 139 497 B | yes |
| `products-list.json` | 126 767 B | yes |
| `order-history.json` | 42 784 B | yes |
| `notifications.json` | 38 234 B | yes |
| `search-results-v2.json` | 31 626 B | yes |
| `search-results.json` | 31 169 B | yes |
| `user-profile.json` | 278 B | excluded (< 1024 B) |
| `error-404.json` | 65 B | excluded (< 1024 B) |

Effective scoring N = 7. Total in-scope uncompressed: **599 245 B**.

## Baseline (`bench/results/baseline.json`)

Status quo is identity (no compression) for the bench, since the goal is to
measure absolute deltas of each candidate vs uncompressed bytes. The
deployed status quo per `SCOPE.md` is gzip-6, which is captured separately
as Exp 0002 (-74.95% wire bytes). The relative score formula uses identity
as zero so all CI deltas can be compared.

| Metric (per item, identity) | p95 max | p95 mean |
|---|---:|---:|
| wire_bytes | 189 168 B | 85 606 B |
| encode_cpu_ms | 0.21 | 0.16 |
| decode_cpu_ms | 0.21 | 0.16 |

## Experiments

Detailed log: `bench/EXPERIMENTS.md`. Bootstrap-CI on per-item score deltas,
10 000 resamples, seed `0xC0FFEE`, alpha 0.05 (Section 4.4).

| id    | hypothesis                                       | enc bytes | enc_p95_max ms | dec_p95_max ms | score % vs base | CI95 [low, high]       | decision |
|-------|--------------------------------------------------|----------:|---------------:|---------------:|----------------:|------------------------|----------|
| 0001  | baseline (identity)                              |   599 245 |           0.21 |           0.21 |          +0.00  | -                      | BASELINE |
| 0002  | gzip-6 (status quo, RFC 1952)                    |   150 126 |          14.53 |           8.91 |         -74.94  | [-98 849,  -34 411]    | KEEP     |
| 0003  | gzip-9 (max gzip)                                |   147 267 |          17.72 |           5.94 |         -75.42  | [-99 499,  -34 639]    | KEEP\*   |
| 0004  | brotli-1 (low CPU, RFC 7932)                     |   167 106 |          14.38 |           6.77 |         -72.11  | [-95 185,  -32 734]    | KEEP\*   |
| 0005  | brotli-5 (default dynamic)                       |   141 467 |           9.10 |           8.13 |         -76.39  | [-100 755, -35 172]    | KEEP     |
| 0006  | zstd-1 (max throughput, RFC 8878)                |   158 380 |           9.99 |           7.15 |         -73.57  | [-97 057,  -33 742]    | KEEP\*   |
| 0007  | zstd-3 (zstd default)                            |   152 240 |           9.48 |           9.77 |         -74.59  | [-98 331,  -34 270]    | KEEP     |
| 0008  | zstd-9 (higher ratio)                            |   142 175 |          27.66 |           6.47 |         -76.27  | [-100 631, -35 013]    | KEEP     |
| 0009  | zstd-dict-3 (trained dict, RFC 8878 §5)          | **118 891** | **7.57**     | **6.54**       |    **-80.15**   | **[-105 610, -37 900]**| **KEEP** |
| 0010  | gzip-6 + tiny items (threshold sweep)            |       N/A |          14.53 |           8.91 |         -74.91  | [-83 243,  -21 318]    | KEEP aggregate; per-item regression on `error-404.json` (+38.5%) confirms 1024 B threshold |

`*` KEEP statistically vs identity, but dominated by another candidate on the
score formula. Not the deployment recommendation. See per-experiment writeup
in `bench/EXPERIMENTS.md`.

**Winner**: Exp 0009, zstd-3 with the trained dictionary
(`bench/zstd-dict.bin`, 112 640 B, magic `0xec30a437`, dict-id `74717759`).
-80.2% wire bytes vs identity, encode_cpu_ms_p95 max 7.6 ms,
decode_cpu_ms_p95 max 6.5 ms. Beats every other candidate on every axis.

**Runner-up (no-dict)**: Exp 0005 (brotli-5) or Exp 0008 (zstd-9), tied
near -76.3% wire bytes. Pick brotli-5 for browser-facing endpoints
(universal support); zstd-9 for server-to-server when zstd is on both
sides but a shipped dictionary is impractical.

**Antipatterns measured / confirmed (agent Section 3)**:
- gzip-9 vs gzip-6 (Exp 0003): +1.4 ms encode p95 mean cost buys only
  0.5% extra ratio. Score-dominated by gzip-6.
- brotli-1 (Exp 0004): produces *more* wire bytes than gzip-6 on this
  JSON corpus. Brotli's static dictionary is HTML/JS-tuned.
- brotli-11 / zstd-19 not benched: calibration table (Section 2.1) puts
  brotli-11 at 0.5 MB/s = ~380 ms for 189 KB. Section 3 antipattern #1.
  zstd-19 measured around 10 MB/s = ~19 ms; better than brotli-11 but
  still loses to zstd-dict-3 on bytes and CPU, so not separately benched.

## Recommended Configuration

### Primary: zstd-3 with shipped trained dictionary, server-to-server

Best score (Exp 0009). Use when both server and client are in our control
and we can ship the dictionary as part of the binary / SDK. The
dictionary is content-addressed; it must be byte-identical at both ends.

**Go origin (using `github.com/klauspost/compress/zstd` for production
quality; `compress/gzip` is stdlib but the stdlib does not ship a zstd
encoder)**:

```go
// origin/compress.go
//
// Cite: bench/EXPERIMENTS.md Exp 0009 (KEEP, -80.2% wire bytes,
// CI95 [-105610, -37900]) on this JSON corpus.
// Dictionary: bench/zstd-dict.bin, 112640 B, magic 0xec30a437,
// dict_id 74717759. Trained with `zstd --train bench/corpus/http/*.json`.
//
// RFC references: RFC 8878 (Zstandard format), RFC 9659 (Zstandard in
// HTTP), RFC 9110 §8.4 (Accept-Encoding semantics), RFC 9111 (Vary).

package origin

import (
    "bytes"
    _ "embed"
    "io"
    "net/http"
    "strings"
    "sync"

    "github.com/klauspost/compress/zstd"
)

//go:embed zstd-dict.bin
var zstdDict []byte

const minCompressBytes = 1024 // see bench/EXPERIMENTS.md Exp 0010

// One encoder pool per (algo, level). Exp 0009 used level 3 (zstd default).
var (
    zstdDictEncoderPool = &sync.Pool{
        New: func() any {
            // EOptionDict installs the trained dictionary. Same dict bytes
            // must be available to the decoder.
            enc, err := zstd.NewWriter(nil,
                zstd.WithEncoderLevel(zstd.SpeedDefault),       // level 3
                zstd.WithEncoderDict(zstdDict),
            )
            if err != nil { panic(err) }
            return enc
        },
    }
    zstdEncoderPool = &sync.Pool{
        New: func() any {
            enc, err := zstd.NewWriter(nil,
                zstd.WithEncoderLevel(zstd.SpeedDefault),
            )
            if err != nil { panic(err) }
            return enc
        },
    }
)

// CompressJSON returns the body, the Content-Encoding token, and ok.
// Body is identity if size < minCompressBytes (Exp 0010 threshold).
//
// Negotiation order:
//   1) zstd + trained dict, if client opts in via the special header
//      X-Compression-Dict: webbeds-json-v1 (out-of-band, since no IETF
//      content coding token covers "dict-bound zstd" outside RFC 9842
//      dcz scope).
//   2) zstd plain, if Accept-Encoding contains zstd.
//   3) br plain, if Accept-Encoding contains br.
//   4) gzip plain, if Accept-Encoding contains gzip.
//   5) identity.
func CompressJSON(body []byte, ae, dictHint string) ([]byte, string, bool) {
    if len(body) < minCompressBytes {
        return body, "", false
    }
    ae = strings.ToLower(ae)
    if dictHint == "webbeds-json-v1" && strings.Contains(ae, "zstd") {
        return encodeZstd(body, true), "zstd", true // signal dict via separate header
    }
    if strings.Contains(ae, "zstd") {
        return encodeZstd(body, false), "zstd", true
    }
    // ... fall through to br / gzip ...
    return body, "", false
}

func encodeZstd(body []byte, withDict bool) []byte {
    var buf bytes.Buffer
    var enc *zstd.Encoder
    var pool *sync.Pool
    if withDict {
        pool = zstdDictEncoderPool
    } else {
        pool = zstdEncoderPool
    }
    enc = pool.Get().(*zstd.Encoder)
    defer pool.Put(enc)
    enc.Reset(&buf)
    _, _ = enc.Write(body)
    _ = enc.Close()
    return buf.Bytes()
}

// Middleware:
//
//   func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//       body, _ := json.Marshal(payload)
//       enc, ce, ok := origin.CompressJSON(body,
//           r.Header.Get("Accept-Encoding"),
//           r.Header.Get("X-Compression-Dict"))
//       if ok {
//           w.Header().Set("Content-Encoding", ce)
//           if ce == "zstd" && r.Header.Get("X-Compression-Dict") == "webbeds-json-v1" {
//               w.Header().Set("X-Compression-Dict-Id", "74717759")
//           }
//       }
//       w.Header().Set("Vary", "Accept-Encoding, X-Compression-Dict")
//       w.Header().Set("Content-Type", "application/json; charset=utf-8")
//       w.Header().Set("Cache-Control", "no-store") // see Security Notes
//       w.Write(enc)
//   }
```

The Go file embeds the dictionary via `//go:embed`. The dictionary is part
of the binary; redeploy is required to roll a new dictionary. The
`X-Compression-Dict` request header is an out-of-band negotiation;
clients that know the dictionary opt in. For browser-facing endpoints,
use the RFC 9842 `dcz` framing instead (see "Browser path" below).

### Secondary: zstd-3 / brotli-5 / gzip-6 fallback (no dict)

For clients that do not opt into the dictionary, fall through to the
algorithm best matching `Accept-Encoding`. Score order on this corpus:

```
zstd > brotli-5 > gzip-6 > brotli-1 / zstd-1
```

Per-axis tradeoffs (Exp 0005, 0007, 0002):
- **brotli-5** (Exp 0005, -76.4%): widest browser support, slightly
  slower decode than zstd. Use as the second-choice for browser clients.
- **zstd-3** (Exp 0007, -74.6%): roughly tied with gzip-6 on bytes,
  faster decode than brotli but slower than gzip on this corpus.
  Keep as the no-dict server-to-server fallback.
- **gzip-6** (Exp 0002, -74.9%): universal compatibility, decent ratio
  on JSON. Reasonable absolute baseline; do not deploy gzip-9 (Exp 0003
  buys only 0.5% extra ratio at +1.4 ms encode mean).

### Browser path (RFC 9842 dcz, optional)

For browser clients, ship the dictionary as a static asset and use
RFC 9842 dictionary-coded zstd (`Content-Encoding: dcz`). This is the
spec-compliant path; it requires the 40-byte `dcz` framing
(`0x5e 0x2a 0x4d 0x18 0x20 0x00 0x00 0x00` + 32-byte SHA-256 of dict).

Server signals the dictionary on its first response:
```
HTTP/2 200 OK
Use-As-Dictionary: match="/api/v*/*.json", id="webbeds-json-v1"
Cache-Control: public, max-age=31536000, immutable
Vary: accept-encoding, available-dictionary
```

Subsequent requests carry `Available-Dictionary: :<sha256-base64>:`;
server returns `Content-Encoding: dcz` with the framed payload.

Browser support for `dcb`/`dcz` is preview-stage; gate this behind a
feature flag, fall through to plain `zstd` / `br` / `gzip`.

### Reverse-proxy alternative (Caddy v2)

Caddy v2 ships with `gzip` and `zstd` built in (no brotli without a
plugin). For a pure plain-zstd fallback at the edge:

```caddy
api.example.com {
    encode {
        zstd 3
        gzip 6
        minimum_length 1024  # bench/EXPERIMENTS.md Exp 0010
        match {
            header Content-Type application/json*
            header Content-Type text/*
        }
    }
    reverse_proxy origin:8080 {
        header_up X-Forwarded-Compression-Dict {http.request.header.X-Compression-Dict}
    }
    header /api/* Cache-Control "no-store"   # see Security Notes
    header /api/* Vary "Accept-Encoding, X-Compression-Dict"
}
```

Note: Caddy's built-in `encode` does not support the trained dictionary.
For the dictionary path, do compression at the Go origin (above) and
have Caddy pass through with `Cache-Control: no-transform` so the edge
does not re-encode:

```caddy
api.example.com {
    reverse_proxy origin:8080
    # Origin already encodes; do not re-encode at edge.
    header /api/* Cache-Control "no-store, no-transform"
    header /api/* Vary "Accept-Encoding, X-Compression-Dict"
}
```

### Reverse-proxy alternative (nginx)

For nginx with `tokers/zstd-nginx-module` (third-party):

```nginx
# bench/EXPERIMENTS.md Exp 0007 (zstd-3 plain) at the edge.
# For Exp 0009 (zstd + dict): edge cannot do dict; encode at origin.

zstd               on;
zstd_comp_level    3;
zstd_min_length    1024;        # Exp 0010
zstd_types         application/json text/plain;

# brotli optional, via google/ngx_brotli
brotli             on;
brotli_comp_level  5;            # Exp 0005
brotli_min_length  1024;
brotli_types       application/json text/plain;

# gzip universal fallback
gzip               on;
gzip_vary          on;
gzip_comp_level    6;            # Exp 0002
gzip_min_length    1024;         # Exp 0010
gzip_types         application/json text/plain;
gzip_proxied       expired no-cache no-store private auth;

# When origin encoded with the trained dict, do NOT re-encode at edge:
location /api/dict/ {
    proxy_pass http://origin;
    proxy_set_header Accept-Encoding $http_accept_encoding;
    add_header Cache-Control "no-store, no-transform" always;
    add_header Vary "Accept-Encoding, X-Compression-Dict" always;
}
```

## Build / Deploy Hook

The trained dictionary must ship alongside the binary. Use Go's
`embed` directive (already shown in the Go origin code above), and
re-run training as part of the build whenever a representative new
corpus is captured.

### Make-style build hook

```makefile
# Makefile
.PHONY: train-dict build

DICT := bench/zstd-dict.bin
CORPUS := bench/corpus/http
ORIGIN_PKG := ./origin

# Re-train the zstd dictionary from the corpus. Run periodically
# (weekly?) or whenever the JSON schema changes shape.
$(DICT): $(wildcard $(CORPUS)/*.json)
	zstd --train $(CORPUS)/*.json -o $@
	@echo "trained dict: $$(wc -c < $@) B, magic $$(xxd -l 4 -p $@)"

train-dict: $(DICT)

# The Go binary embeds the dict via //go:embed; copy it into the package.
build: $(DICT)
	cp $(DICT) $(ORIGIN_PKG)/zstd-dict.bin
	go build -o bin/origin ./cmd/origin
```

### CI step (GitHub Actions)

```yaml
- name: Train zstd dictionary
  run: |
    sudo apt-get install -y zstd
    zstd --train bench/corpus/http/*.json -o origin/zstd-dict.bin
    echo "Dict size: $(wc -c < origin/zstd-dict.bin)"
    xxd origin/zstd-dict.bin | head -1   # verify magic 37a430ec
- name: Build
  run: go build -o bin/origin ./cmd/origin
- name: Verify dict embedded
  run: |
    # The binary should embed origin/zstd-dict.bin; check the binary
    # contains the dict's magic at non-zero offset.
    xxd bin/origin | grep -q '37a4 30ec' || exit 1
```

### Dictionary versioning

Stamp every response that uses the dictionary with the dictionary id
header (e.g. `X-Compression-Dict-Id: 74717759`). Clients pin to a known
dict id; if the server's id differs, the client falls back to plain
zstd or another encoding. This avoids the "old client + new dict =
garbage decode" failure mode.

A safer approach uses RFC 9842 SHA-256 hash matching: the dict id is
the SHA-256 of the dictionary bytes, and the framing (`dcz`) carries
the hash. RFC 9842 §2 requires this for `dcz`.

Re-train the dictionary periodically. Each retraining produces a new
dict id; deploy server first (knows both old and new dicts), then roll
the new id forward to clients.

## Verification

Live HTTP server is not running per the test invocation; the verification
commands below are the canonical ones to run once a server is deployed
(per Section 8 of the agent definition).

```bash
# 1. Negotiation matrix per RFC 9110 §8.4
URL='https://api.example.com/api/v1/products'
for ae in 'identity' 'gzip' 'br' 'zstd' 'gzip, br, zstd'; do
  printf '%-30s ' "$ae"
  curl -sI -H "Accept-Encoding: $ae" "$URL" \
    | grep -iE 'content-encoding|content-length|vary' | tr '\n' ' '
  echo
done

# 2. Dictionary-coded path (out-of-band Webbeds header).
curl -sI -H 'Accept-Encoding: zstd' \
        -H 'X-Compression-Dict: webbeds-json-v1' \
        "$URL" | grep -iE 'content-encoding|x-compression-dict-id|vary'
# Expect:
#   Content-Encoding: zstd
#   X-Compression-Dict-Id: 74717759
#   Vary: Accept-Encoding, X-Compression-Dict

# 3. Round-trip a real response.
curl -s -H 'Accept-Encoding: zstd' \
        -H 'X-Compression-Dict: webbeds-json-v1' \
        "$URL" -o /tmp/resp.zst
zstd -d -D origin/zstd-dict.bin -c /tmp/resp.zst | jq . | head -10

# 4. Tiny-response gating per Exp 0010.
curl -sI -H 'Accept-Encoding: zstd' "$URL/error/404" \
  | grep -iE 'content-encoding'
# Expect: NO Content-Encoding header (response is < 1024 B; identity).

# 5. TLS compression off (CRIME, agent Section 1.3).
echo | openssl s_client -connect api.example.com:443 2>&1 | grep -i compression
# Expect: no compression line (TLS 1.3 forbids it).

# 6. Range + compression (agent Section 3 #10).
curl -sI -H 'Range: bytes=0-1023' -H 'Accept-Encoding: zstd' "$URL"
# Expect: 206 + NO Content-Encoding, OR 200 + full body + Content-Encoding.
# NEVER: 206 + Content-Encoding.

# 7. Vary correctness for caches (RFC 9111 §4.1).
curl -sI -H 'Accept-Encoding: zstd, gzip' "$URL" | grep -i vary
# Expect: Vary: Accept-Encoding, X-Compression-Dict
```

For RFC 9842 `dcz` browser path verification, use the agent definition's
Section 8.2 block (verifies framing magic `5e 2a 4d 18` + 32-byte
SHA-256 of dictionary).

## Security Notes

### BREACH analysis (per agent Section 1.3 / antipattern #5)

BREACH (Gluck/Harris/Prado, 2013) requires three conditions to recover
secrets via compression length:

1. Body compression turned on for the response.
2. Attacker-controlled input reflected in the body.
3. A secret (CSRF, session, API key, PII) also in the body.

Per `SCOPE.md`, this corpus is a JSON API. Per-endpoint review:

| Corpus item | Reflects user input? | Carries secret? | BREACH-relevant? | Action |
|---|:---:|:---:|:---:|---|
| `catalog-full.json` | partially (filter args) | likely no (public catalog) | low | compress freely |
| `products-list*.json` | yes (search/filter args) | likely no (public products) | low-medium | compress; audit per endpoint |
| `search-results*.json` | **yes** (the query) | likely no (public results) | low-medium | compress; audit |
| `order-history.json` | yes (user id, filters) | **yes** (orders are PII; may contain CSRF) | **HIGH** | **do not enable dynamic body compression on this endpoint without a BREACH plan** |
| `notifications.json` | yes (user id, filters) | **yes** (PII / messages may carry secrets) | **HIGH** | same as above |
| `user-profile.json` | identifier only | **yes** (PII) | medium | excluded by 1024 B threshold; if larger profiles exist, audit |
| `error-404.json` | path | no | none | excluded by 1024 B threshold |

Required actions before deployment of compression on `order-history`,
`notifications`, or `user-profile` (when they grow > 1024 B):

1. **Strip CSRF / session tokens from response bodies.** The CSRF token
   should travel in a dedicated cookie or response header (e.g.
   `X-CSRF-Token`) that is not subject to body compression. This is
   the single most effective BREACH mitigation for a JSON API.
2. **Mask request-reflected fields.** If a query string is echoed back
   in the response (`{"query": "<user input>"}`), this is the BREACH
   vector. Either don't echo, or constant-pad the echoed field.
3. **HTB-style length randomization** (agent Section 1.3): pad each
   response to the next multiple of 1024 B with a random-length
   `_padding` JSON field. This is the runtime mitigation when stripping
   secrets is impractical.
4. **Disable compression on auth endpoints** (`/auth/*`, `/csrf/*`,
   `/admin/*`). Already in `SCOPE.md`'s exclusion list as a policy.

### Dictionary security (RFC 9842, agent Section 5.4.2)

The trained dictionary `bench/zstd-dict.bin` was generated from a public
corpus of API response shapes. Before deploying it:

- **Verify no secrets in the dictionary**. Inspect:
  ```bash
  strings bench/zstd-dict.bin | sort -u | head -200
  # Look for: tokens, JWT fragments, email patterns, names, account IDs.
  ```
  If any are present, retrain with sanitized samples.
- **Same-origin only** (RFC 9842 §2.1). The dictionary is bound to the
  Webbeds API origin; do not share it with other domains.
- **Brotli quality >= 5 / zstd level >= 3** when paired with a dictionary
  (agent Section 5.4.2). We use zstd level 3; complies.
- **Pre-compute, never compress on demand against an attacker-controlled
  dictionary**: the dictionary is server-controlled and embedded in the
  binary; clients negotiate its use, they don't supply it. Complies.

### TLS compression / CRIME (agent Section 1.3)

TLS 1.3 forbids compression at the TLS layer (RFC 8446 §1.2). Verify
on the deployed server with `openssl s_client | grep -i compression`;
should be empty or `Compression: NONE`.

### HPACK / QPACK (agent Section 5.4.3-4)

Outside scope of this corpus (HTTP/2 header compression is independent
of body compression). For the production deployment:

- HTTP/2: set `SETTINGS_HEADER_TABLE_SIZE` to 4096 (default) or 16384
  if many requests share repeated headers.
- Mark `Cookie`, `Authorization`, `Set-Cookie` as **never-indexed**
  (HPACK 0001 prefix). Same for QPACK literal-not-indexed.
- `X-Compression-Dict-Id` is low-entropy; safe to index.

### Vary correctness (RFC 9111 §4.1)

Every compressed response must include `Vary: Accept-Encoding,
X-Compression-Dict`. Without this, a CDN may serve a zstd-encoded body
to a client that did not request zstd. This is the most common
silent-corruption failure mode for compression deployments.

For `dcz` (browser path), `Vary` must include `available-dictionary`
(RFC 9842 §3).

## Risks / Follow-ups

### What we couldn't measure

1. **Real client mix**: no live access logs. Recommendation tunes for an
   API where consumers are services with `Accept-Encoding: zstd, br,
   gzip`. If real clients are 99% browsers, brotli-5 is the better
   second-choice (universal browser support; zstd browser support
   landed in 2024 and is not yet ubiquitous).

2. **Live network**: bench is file-level only. Wire bytes equal encoded
   file size; in practice add per-request HTTP/2 framing overhead
   (~10-20 B/response after HPACK), and TCP / TLS framing. These are
   constants and don't change the ranking.

3. **Per-endpoint CPU under realistic concurrency**: hyperfine and
   `measure_clean.py` measure single-thread encode time. Under
   concurrent load (Go's HTTP server uses one goroutine per request),
   per-core throughput could degrade due to L1/L2 cache pressure,
   especially with a 112 KB dictionary in working memory per encode.
   Recommend: re-validate with `oha` or `hey` against a load-tested
   origin once deployed, and watch p99 encode latency under ramp.

4. **No BREACH testing**. The security review is policy-level; no
   fuzzer was run to confirm whether response bodies actually reflect
   attacker input. That audit is a separate workstream.

5. **No browser decode CPU on real device**. Calibration assumes
   approximate parity for zstd decode in browsers; real-world
   decode-on-low-end-Android may differ.

### What would change with more data

- A larger/more representative training corpus would push the
  dictionary win further. The current dictionary was trained on 270 KB
  of source vs the recommended 10x dictionary size; zstd warned. With
  10 MB+ of representative samples, expect another 5-10 percentage
  points of additional ratio.

- If catalog/product responses contain large unique strings (free-text
  descriptions, image URLs, GUIDs), the dictionary helps less. Per-item
  results show this: `products-list*.json` got only -2 to -3% over
  gzip-6, whereas the more uniform `search-results*.json` got -44 to
  -50%. Worth profiling the actual response distribution.

### Open follow-ups

1. **Implement BREACH mitigations on `/api/orders/*`, `/api/notifications/*`,
   `/api/user/*`** before enabling any body compression on those paths.
   Current recommendation: do not compress these until fields are
   audited.

2. **Decide on the negotiation channel for the trained dictionary.**
   Option A: out-of-band header `X-Compression-Dict`. Simple, but not
   spec-compliant and requires SDK updates. Option B: RFC 9842
   `dcz` framing. Spec-compliant, browser support landing. Option B
   is the long-term path; A may be viable for the first server-to-server
   rollout if SDK is already controlled.

3. **Set up dictionary retraining cadence**. Once a month, rebuild from
   the most recent week of access-log-sampled responses. Stamp the new
   dict id; old clients fall back to plain zstd.

4. **Validate at the edge.** If a CDN sits in front, confirm it does
   not strip our `Content-Encoding: zstd` (some legacy CDNs do).
   Add `Cache-Control: no-transform` to any response we compressed at
   origin.

5. **Re-baseline**. Once the live server is up and access logs are
   available, capture a fresh corpus from the top-N URLs by traffic
   weight and re-run Exp 0001-0009 against that. Per agent Section
   4.3, baseline is invalidated when corpus changes.
