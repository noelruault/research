#!/usr/bin/env python3
"""Generate the shapes-vs-codecs dashboard, embedding every asset as a data URI
(the Artifact CSP blocks external hosts, so nothing may be fetched at runtime)."""
import base64
import json
import pathlib

here = pathlib.Path(__file__).parent
assets = here / "assets"
rows = {}
for line in (here / "manifest.tsv").read_text().strip().splitlines():
    i, fam, label, byt, ps = line.split("\t")
    rows[i] = {"fam": fam, "label": label, "bytes": int(byt), "psnr": float(ps)}

for i in rows:
    b = (assets / f"{i}.webp").read_bytes()
    rows[i]["uri"] = "data:image/webp;base64," + base64.b64encode(b).decode()


def kb(n):
    return f"{n/1024:.1f} KB" if n >= 1024 else f"{n} B"


def card(rid, note="", win=None):
    r = rows[rid]
    # bar length is byte count relative to the heaviest card on the page that is not the source, so file weight reads as form and not only as a number
    frac = min(1.0, r["bytes"] / 28000)
    badge = ""
    if win is not None:
        cls = "win" if win < 0 else "loss"
        badge = f'<span class="badge {cls}">{win:+.1f}%</span>'
    return f'''<figure class="card {r['fam']}">
  <img src="{r['uri']}" alt="{r['label']}" width="512" height="288" loading="lazy">
  <figcaption>
    <div class="crow"><span class="cname">{r['label']}</span>{badge}</div>
    <div class="cnum"><b>{kb(r['bytes'])}</b><span class="dim">{r['psnr']:.2f} dB</span></div>
    <div class="bar"><i style="width:{frac*100:.1f}%"></i></div>
    {f'<p class="note">{note}</p>' if note else ''}
  </figcaption>
</figure>'''


WEBP = [(23.53,2508),(24.54,3642),(24.84,4060),(25.32,4742),(25.59,5306),(25.86,5790),
        (26.16,6352),(26.65,7344),(27.07,8310),(27.47,9234),(27.86,10286),(28.22,11100),
        (28.63,12350),(28.99,13404),(29.37,14454),(29.72,15510),(30.04,16592)]
AVIF = [(22.17,2145),(22.80,2368),(23.32,2606),(23.71,2838),(24.05,3047),(24.34,3262),
        (24.65,3534),(25.56,4400),(26.20,5112),(27.08,6258),(27.72,7242),(28.71,8911),
        (29.41,10263),(30.56,12797)]
SHAPES = [(24.03,2986),(24.52,3465),(25.10,4295),(26.09,6156),(26.44,6488),(27.10,7870),
          (27.60,9124),(28.12,10543),(28.66,12202),(29.17,13990),(29.70,16091)]

RANK = [
    ("AVIF q30", 8911, 28.71, "avif", ""),
    ("shape coder — research only, not built", 12202, 28.66, "shapes", "1,685 regions"),
    ("WebP q34", 12350, 28.63, "webp", ""),
    ("JPEG q36", 19503, 28.77, "jpeg", ""),
    ("indexed PNG of the n=16 grid", 27802, 28.61, "png", "lossless raster"),
    ("SVG rect cover — what ships today", 113562, 28.61, "shipped", "32,924 rects"),
]

KILLED = [
    ("Let the decoder regrow regions — “don’t serialize”", "collective systems",
     "Provably a rename. Any region reconstructible from decoded pixels is a function of the causal past, so it <em>is</em> a context model — bounded by H(X | causal past). ≥ 16.2 KB."),
    ("PDE diffusion inpainting (Weickert R-EED)", "physics / biology",
     "The literal morphogenesis codec. Seed mask <em>alone</em> costs 9.5 KB — over the wall before a single colour is sent. Total ~21–25 KB."),
    ("Centroidal Voronoi / Lloyd relaxation", "physics", "44.3 KB."),
    ("Seeds derived by the decoder — geometry costs exactly zero", "physics",
     "Colours alone still 17.6 KB. Collapses to downsample-then-upsample."),
    ("Shared dictionary across a best-case corpus", "compression",
     "Built to favour shapes: 8 images, one shared palette, identical dimensions. Shapes gain 1.02×, WebP 1.01×. The asymmetry does not exist."),
    ("Hyperspectral amortization — 200 bands, geometry free", "space systems",
     "Geometry falls to 2% of the budget and shapes still lose 3.4–14.5×."),
    ("Portilla-Simoncelli texture synthesis", "vision",
     "~5–6 KB — the only sub-wall number found anywhere. But PSNR collapses to ~18 dB. It is a generative texture codec, not a shape format."),
    ("Phase transition in palette size", "physics",
     "Swept n = 2…256. bpp monotonic. No knee, no regime where shapes get cheap."),
]

FALSIFIED = [
    ("Beats WebP by 31%", "Lossy pipeline compared against WebP in <em>lossless</em> mode. At matched fidelity it is 1.41× behind."),
    ("We’re at the entropy floor", "Order-3 was sample-<em>starved</em>, not diluted — 36 samples per context. Mixing recovers it: 16.2 KB."),
    ("Geometry alone costs 19.2 KB", "Artifact of my own weak segmenter. A proper energy-minimizing one costs 9.0 KB."),
    ("Renders at 8K for the same bytes", "Fake vectorness. A rect cover of a quantized grid <em>is</em> nearest-neighbour upscaling."),
    ("9.8× smaller than the source", "The win was the downscale. Against the same pixels it is 6.9× <em>larger</em>."),
    ("3–9% better at low rate", "Taken from one run of a nondeterministic merge, interpolated by eye. Truth: 1–6%."),
]

rank_rows = "".join(
    f'<tr class="{c}"><td class="rk">{n+1}</td><td class="nm"><span class="dot"></span>{lbl}'
    f'{f"<em>{note}</em>" if note else ""}</td>'
    f'<td class="nu">{b:,}</td><td class="nu dim">{p:.2f}</td>'
    f'<td class="nu">{b/8911:.2f}×</td></tr>'
    for n, (lbl, b, p, c, note) in enumerate(RANK))

killed_rows = "".join(
    f'<tr><td class="nm">{m}<em>{d}</em></td><td class="rs">{r}</td></tr>'
    for m, d, r in KILLED)

fals_rows = "".join(
    f'<li><b>&ldquo;{c}&rdquo;</b><span>{w}</span></li>' for c, w in FALSIFIED)

html = f'''<title>Shapes vs. the codecs — measured</title>
<style>
:root {{
  --bg:#f4f6f9; --surface:#ffffff; --line:#dfe4ec; --line2:#eef1f6;
  --ink:#151922; --ink2:#5b6479; --ink3:#8b94a8;
  --shapes:#0e8fb8; --avif:#b8791a; --webp:#7c5cd0; --jpeg:#6b7688; --png:#818b9e; --shipped:#c2412f;
  --good:#1a7f47; --bad:#c2412f;
  --grid:#e6eaf1;
}}
@media (prefers-color-scheme:dark) {{
  :root {{
    --bg:#0f1218; --surface:#161b23; --line:#252c38; --line2:#1c222c;
    --ink:#e7eaf1; --ink2:#98a2b6; --ink3:#6c7689;
    --shapes:#4cc9f0; --avif:#f5a524; --webp:#a78bfa; --jpeg:#7d8899; --png:#8b95a8; --shipped:#f85149;
    --good:#3fb950; --bad:#f85149;
    --grid:#222937;
  }}
}}
:root[data-theme="dark"] {{
  --bg:#0f1218; --surface:#161b23; --line:#252c38; --line2:#1c222c;
  --ink:#e7eaf1; --ink2:#98a2b6; --ink3:#6c7689;
  --shapes:#4cc9f0; --avif:#f5a524; --webp:#a78bfa; --jpeg:#7d8899; --png:#8b95a8; --shipped:#f85149;
  --good:#3fb950; --bad:#f85149; --grid:#222937;
}}
:root[data-theme="light"] {{
  --bg:#f4f6f9; --surface:#ffffff; --line:#dfe4ec; --line2:#eef1f6;
  --ink:#151922; --ink2:#5b6479; --ink3:#8b94a8;
  --shapes:#0e8fb8; --avif:#b8791a; --webp:#7c5cd0; --jpeg:#6b7688; --png:#818b9e; --shipped:#c2412f;
  --good:#1a7f47; --bad:#c2412f; --grid:#e6eaf1;
}}

*,*::before,*::after {{ box-sizing:border-box; }}
body {{
  margin:0; background:var(--bg); color:var(--ink);
  font:400 16px/1.6 ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;
  -webkit-font-smoothing:antialiased;
}}
.mono {{ font-family:ui-monospace,SFMono-Regular,"SF Mono",Menlo,Consolas,monospace; }}
.wrap {{ max-width:1180px; margin:0 auto; padding:0 24px 96px; }}

/* ---- masthead ---- */
header {{ padding:72px 0 40px; border-bottom:1px solid var(--line); }}
.eyebrow {{
  font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:11px;
  letter-spacing:.16em; text-transform:uppercase; color:var(--ink3); margin:0 0 20px;
}}
h1 {{
  margin:0 0 20px; font-size:clamp(2rem,5vw,3.4rem); line-height:1.03;
  letter-spacing:-.035em; font-weight:660; text-wrap:balance; max-width:16ch;
}}
h1 b {{ color:var(--shapes); font-weight:660; }}
.lede {{ margin:0; max-width:64ch; font-size:1.075rem; color:var(--ink2); }}
.lede strong {{ color:var(--ink); font-weight:600; }}

/* ---- stat strip ---- */
.stats {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(190px,1fr)); gap:1px;
  background:var(--line); border:1px solid var(--line); margin:40px 0 0; }}
.stat {{ background:var(--surface); padding:20px 22px; }}
.stat .k {{ font-family:ui-monospace,Menlo,monospace; font-size:10.5px; letter-spacing:.14em;
  text-transform:uppercase; color:var(--ink3); }}
.stat .v {{ font-family:ui-monospace,Menlo,monospace; font-variant-numeric:tabular-nums;
  font-size:1.85rem; font-weight:600; letter-spacing:-.02em; margin:8px 0 4px; }}
.stat .s {{ font-size:.82rem; color:var(--ink2); line-height:1.45; }}
.stat.hero .v {{ color:var(--shapes); }}
.stat.bad .v {{ color:var(--bad); }}

section {{ padding-top:72px; }}
h2 {{ font-size:1.55rem; letter-spacing:-.022em; margin:0 0 10px; font-weight:640; }}
.sub {{ margin:0 0 28px; color:var(--ink2); max-width:70ch; font-size:.965rem; }}

/* ---- chart ---- */
.chartbox {{ background:var(--surface); border:1px solid var(--line); padding:22px 20px 14px; }}
canvas {{ width:100%; height:auto; display:block; }}
.legend {{ display:flex; flex-wrap:wrap; gap:20px; padding:16px 4px 4px;
  font-family:ui-monospace,Menlo,monospace; font-size:12px; color:var(--ink2); }}
.legend i {{ display:inline-block; width:22px; height:2.5px; vertical-align:middle;
  margin-right:8px; border-radius:2px; }}

/* ---- image bands ---- */
.band {{ margin-top:44px; }}
.bandhead {{ display:flex; align-items:baseline; gap:14px; flex-wrap:wrap;
  padding-bottom:14px; margin-bottom:20px; border-bottom:1px solid var(--line); }}
.bandhead h3 {{ margin:0; font-size:1.05rem; font-weight:620; letter-spacing:-.01em; }}
.bandhead .fid {{ font-family:ui-monospace,Menlo,monospace; font-size:12px; color:var(--ink3);
  letter-spacing:.04em; }}
.bandhead p {{ margin:0; flex:1 1 340px; font-size:.9rem; color:var(--ink2); }}

.grid {{ display:grid; gap:20px; grid-template-columns:repeat(auto-fit,minmax(268px,1fr)); }}
.card {{ margin:0; background:var(--surface); border:1px solid var(--line); overflow:hidden; }}
.card img {{ width:100%; height:auto; display:block; border-bottom:1px solid var(--line2); }}
.card figcaption {{ padding:13px 15px 15px; }}
.crow {{ display:flex; align-items:center; gap:10px; justify-content:space-between; }}
.cname {{ font-size:.845rem; font-weight:560; letter-spacing:-.005em; }}
.cnum {{ display:flex; align-items:baseline; gap:10px; margin:7px 0 10px;
  font-family:ui-monospace,Menlo,monospace; font-variant-numeric:tabular-nums; }}
.cnum b {{ font-size:1.22rem; font-weight:600; letter-spacing:-.02em; }}
.dim {{ color:var(--ink3); font-size:.78rem; }}
.bar {{ height:3px; background:var(--line2); }}
.bar i {{ display:block; height:100%; background:currentColor; }}
.note {{ margin:11px 0 0; font-size:.79rem; line-height:1.5; color:var(--ink2); }}
.badge {{ font-family:ui-monospace,Menlo,monospace; font-size:11px; font-weight:600;
  padding:2.5px 7px; letter-spacing:.02em; white-space:nowrap; }}
.badge.win {{ color:var(--good); background:color-mix(in srgb,var(--good) 13%,transparent); }}
.badge.loss {{ color:var(--bad); background:color-mix(in srgb,var(--bad) 13%,transparent); }}

.shapes {{ color:var(--shapes); }} .avif {{ color:var(--avif); }} .webp {{ color:var(--webp); }}
.jpeg {{ color:var(--jpeg); }} .png {{ color:var(--png); }} .shipped {{ color:var(--shipped); }}
.source {{ color:var(--ink3); }}

/* ---- tables ---- */
.scroll {{ overflow-x:auto; border:1px solid var(--line); background:var(--surface); }}
table {{ border-collapse:collapse; width:100%; min-width:600px; }}
th {{ font-family:ui-monospace,Menlo,monospace; font-size:10.5px; letter-spacing:.13em;
  text-transform:uppercase; color:var(--ink3); text-align:left; font-weight:500;
  padding:13px 16px; border-bottom:1px solid var(--line); white-space:nowrap; }}
td {{ padding:14px 16px; border-bottom:1px solid var(--line2); vertical-align:top;
  font-size:.9rem; }}
tr:last-child td {{ border-bottom:0; }}
.nu {{ font-family:ui-monospace,Menlo,monospace; font-variant-numeric:tabular-nums;
  text-align:right; white-space:nowrap; }}
.rk {{ font-family:ui-monospace,Menlo,monospace; color:var(--ink3); width:1%; }}
.nm {{ font-weight:520; }}
.nm em {{ display:block; font-style:normal; font-weight:400; font-size:.8rem;
  color:var(--ink2); margin-top:3px; }}
.dot {{ display:inline-block; width:8px; height:8px; margin-right:9px;
  background:currentColor; vertical-align:baseline; }}
tbody tr.shapes .nm, tbody tr.avif .nm, tbody tr.webp .nm,
tbody tr.jpeg .nm, tbody tr.png .nm, tbody tr.shipped .nm {{ color:var(--ink); }}
.rs {{ color:var(--ink2); font-size:.86rem; }}

/* ---- falsification list ---- */
ol.fals {{ list-style:none; counter-reset:f; margin:0; padding:0;
  display:grid; gap:1px; background:var(--line); border:1px solid var(--line); }}
ol.fals li {{ counter-increment:f; background:var(--surface); padding:16px 18px 16px 52px;
  position:relative; }}
ol.fals li::before {{ content:counter(f); position:absolute; left:18px; top:16px;
  font-family:ui-monospace,Menlo,monospace; font-size:11px; color:var(--bad); font-weight:600; }}
ol.fals b {{ display:block; font-weight:560; font-size:.92rem; margin-bottom:4px;
  text-decoration:line-through; text-decoration-color:var(--bad);
  text-decoration-thickness:1.5px; color:var(--ink2); }}
ol.fals span {{ font-size:.86rem; color:var(--ink2); }}

.callout {{ border-left:3px solid var(--shapes); background:var(--surface);
  padding:18px 20px; margin-top:28px; font-size:.92rem; color:var(--ink2); }}
.callout b {{ color:var(--ink); }}

footer {{ margin-top:80px; padding-top:28px; border-top:1px solid var(--line);
  font-size:.85rem; color:var(--ink3); display:flex; gap:22px; flex-wrap:wrap; }}
footer a {{ color:var(--ink2); }}
@media (prefers-reduced-motion:reduce) {{ * {{ animation:none!important; transition:none!important; }} }}
</style>

<div class="wrap">
<header>
  <p class="eyebrow">Research record &middot; noelruault/shapes-image-file-format</p>
  <h1>Can an image made of <b>shapes</b> beat WebP?</h1>
  <p class="lede">Reduce a photograph to regions, code the boundaries, ship the geometry. Five rounds of measurement against the codecs that already exist. <strong>The answer is no &mdash; with one narrow exception, and it is not worth adopting a format for.</strong></p>

  <div class="stats">
    <div class="stat hero"><div class="k">Best shape result</div><div class="v">12.2 KB</div>
      <div class="s">1,685 regions at 28.66 dB. Ties WebP.</div></div>
    <div class="stat"><div class="k">AVIF, same fidelity</div><div class="v">8.9 KB</div>
      <div class="s">Never beaten, at any rate on this image.</div></div>
    <div class="stat bad"><div class="k">What ships today</div><div class="v">110.9 KB</div>
      <div class="s">32,924 rects. 12.8&times; behind AVIF.</div></div>
    <div class="stat"><div class="k">Mechanisms killed</div><div class="v">16</div>
      <div class="s">Physics, collective systems, vision, spacecraft.</div></div>
  </div>
</header>

<section>
  <h2>The rate&ndash;distortion curve</h2>
  <p class="sub">Every point is one encoder setting on one 512&times;288 photograph, measured on a single RGB&nbsp;PSNR definition. Down and to the right is better. The shape coder tracks WebP closely and crosses it at ~29.2&nbsp;dB &mdash; below that it is genuinely smaller; above, WebP pulls away. AVIF sits under both across the whole range.</p>
  <div class="chartbox">
    <canvas id="rd" width="1120" height="500" role="img"
      aria-label="Rate-distortion chart. AVIF is cheapest at every fidelity. The shape coder is slightly below WebP up to about 29.2 dB, then above it."></canvas>
    <div class="legend">
      <span style="color:var(--shapes)"><i style="background:currentColor"></i>shape coder</span>
      <span style="color:var(--webp)"><i style="background:currentColor"></i>WebP</span>
      <span style="color:var(--avif)"><i style="background:currentColor"></i>AVIF</span>
      <span style="color:var(--ink3)"><i style="background:currentColor;height:0;border-top:2px dashed currentColor"></i>crossover &asymp;29.2 dB</span>
    </div>
  </div>
</section>

<section>
  <h2>The same picture, at the same fidelity</h2>
  <p class="sub">Three fidelity bands. Within each band every encoder produces the same measured quality, so the only thing that differs is the byte count &mdash; and what the degradation <em>looks</em> like, which PSNR cannot see. The bar under each figure is its weight.</p>

  <div class="band">
    <div class="bandhead">
      <h3>Band A</h3><span class="fid mono">~28.7 dB</span>
      <p>The evaluation fidelity. The shape coder ties WebP and loses to AVIF by 1.37&times;. What the project ships is 9&times; heavier than its own best result.</p>
    </div>
    <div class="grid">
      {card('source', 'The uncompressed reference every row is measured against.')}
      {card('a_avif', 'The one to beat. Nothing here reaches it.')}
      {card('a_shapes', 'Research only — an idealised entropy estimate, no container. Not built.', -1.2)}
      {card('a_webp')}
      {card('a_jpeg')}
      {card('a_shipped', 'Same pixels as the indexed PNG (27.8 KB) — the geometry is pure overhead.')}
    </div>
  </div>

  <div class="band">
    <div class="bandhead">
      <h3>Band B</h3><span class="fid mono">~26.4 dB</span>
      <p>The low-rate band, where the shape coder does its best work: fewer, larger regions mean the perimeter tax stays small while a block codec keeps paying per-block overhead.</p>
    </div>
    <div class="grid">
      {card('b_shapes', 'Its widest margin over WebP anywhere on the curve.', -6.2)}
      {card('b_webp')}
      {card('b_avif', 'Still 26% cheaper than the shape coder at matched fidelity.')}
    </div>
  </div>

  <div class="band">
    <div class="bandhead">
      <h3>Band C</h3><span class="fid mono">~24 dB</span>
      <p>The floor. The merge bottoms out near 150 regions; WebP bottoms out at q0. Note the failure modes diverge completely &mdash; posterized flat regions against blur and blocking. PSNR scores them equal; an eye would not.</p>
    </div>
    <div class="grid">
      {card('c_shapes', 'Flat regions with hard edges. Reads as a poster, not as a damaged photo.')}
      {card('c_webp', 'WebP at its lowest setting — lower fidelity than the shape coder, in fewer bytes.')}
    </div>
  </div>
</section>

<section>
  <h2>Full ranking at matched fidelity</h2>
  <p class="sub">~28.7&nbsp;dB on the evaluation image. Ratio is against AVIF.</p>
  <div class="scroll"><table>
    <thead><tr><th>#</th><th>Encoder</th><th class="nu">Bytes</th><th class="nu">PSNR</th><th class="nu">vs AVIF</th></tr></thead>
    <tbody>{rank_rows}</tbody>
  </table></div>
  <div class="callout"><b>The shape coder is second, and it does not exist.</b> It is a measured cross-entropy bound from a research segmenter &mdash; no container, no header, no bitstream. What the repository actually ships is last, behind a 1996 PNG by 4&times; and a 1992 JPEG by 6&times;.</div>
</section>

<section>
  <h2>What was tried and killed</h2>
  <p class="sub">Eight of sixteen. Each was required to produce a mechanism, a runnable experiment and a predicted number before it was run &mdash; so every rejection carries a measurement rather than an opinion.</p>
  <div class="scroll"><table>
    <thead><tr><th>Mechanism</th><th>Result</th></tr></thead>
    <tbody>{killed_rows}</tbody>
  </table></div>
</section>

<section>
  <h2>Six claims this research killed &mdash; its own</h2>
  <p class="sub">Every one was produced by this investigation, believed, written into a README, then falsified by a later measurement in the same investigation. Five of the six were the same error: the wrong baseline, always in the direction that flattered the hypothesis.</p>
  <ol class="fals">{fals_rows}</ol>
  <div class="callout"><b>None was caught by reasoning.</b> Each was caught by a later measurement that happened to overlap. The only structural defence that worked was requiring every investigating agent to reproduce the shared evaluation before its findings were believed &mdash; which caught a false headline in flight, from an agent that had silently substituted a different image.</div>
</section>

<section>
  <h2>So where does this leave shapes?</h2>
  <p class="sub">Not as a codec. The byte win is 1&ndash;6% over the second-best format, in a band where the best format leads by 8&ndash;52%, and it sits inside the margin a real container would erase. But the win lands in the same place as everything shapes offer that rasters cannot &mdash; resolution-independent geometry, drawing with no decode step, per-region addressability, recolouring by palette entry &mdash; which is thumbnails, placeholders, e-ink, low-power displays and starved links. The representation is the product. The compression never was.</p>
</section>

<footer>
  <span>512&times;288 &middot; RGB PSNR &middot; one image, no BD-rate, no perceptual metric</span>
  <a href="https://github.com/noelruault/research/tree/main/shapes-image-file-format">Full record &amp; raw data</a>
</footer>
</div>

<script>
const WEBP={json.dumps(WEBP)}, AVIF={json.dumps(AVIF)}, SHAPES={json.dumps(SHAPES)};
const cv=document.getElementById('rd'), cx=cv.getContext('2d');
function css(n){{ return getComputedStyle(document.documentElement).getPropertyValue(n).trim(); }}
function draw(){{
  const dpr=window.devicePixelRatio||1, W=cv.clientWidth, H=Math.min(500,Math.max(320,W*0.45));
  cv.width=W*dpr; cv.height=H*dpr; cv.style.height=H+'px';
  cx.setTransform(dpr,0,0,dpr,0,0); cx.clearRect(0,0,W,H);
  const L=64, R=18, T=18, B=44, x0=22, x1=30.6, y0=0, y1=17408;
  const X=v=>L+(v-x0)/(x1-x0)*(W-L-R), Y=v=>T+(1-(v-y0)/(y1-y0))*(H-T-B);

  cx.strokeStyle=css('--grid'); cx.fillStyle=css('--ink3');
  cx.font='11px ui-monospace,Menlo,monospace'; cx.lineWidth=1;
  for(let b=0;b<=17408;b+=2048){{
    cx.beginPath(); cx.moveTo(L,Y(b)+.5); cx.lineTo(W-R,Y(b)+.5); cx.stroke();
    cx.textAlign='right'; cx.textBaseline='middle'; cx.fillText((b/1024).toFixed(0)+' KB',L-10,Y(b));
  }}
  for(let d=23;d<=30;d++){{
    cx.textAlign='center'; cx.textBaseline='top'; cx.fillText(d+' dB',X(d),H-B+12);
  }}
  // crossover marker
  cx.save(); cx.setLineDash([5,5]); cx.strokeStyle=css('--ink3'); cx.globalAlpha=.75;
  cx.beginPath(); cx.moveTo(X(29.17),T); cx.lineTo(X(29.17),H-B); cx.stroke(); cx.restore();

  const line=(pts,col,w)=>{{
    cx.strokeStyle=col; cx.lineWidth=w; cx.lineJoin='round';
    cx.beginPath(); pts.forEach(([d,b],i)=>i?cx.lineTo(X(d),Y(b)):cx.moveTo(X(d),Y(b))); cx.stroke();
    cx.fillStyle=col; pts.forEach(([d,b])=>{{ cx.beginPath(); cx.arc(X(d),Y(b),2.6,0,7); cx.fill(); }});
  }};
  line(AVIF,css('--avif'),2);
  line(WEBP,css('--webp'),2);
  line(SHAPES,css('--shapes'),2.75);

  cx.fillStyle=css('--ink3'); cx.font='10.5px ui-monospace,Menlo,monospace';
  cx.textAlign='left'; cx.textBaseline='top'; cx.fillText('crossover',X(29.17)+7,T+4);
  cx.save(); cx.translate(15,H/2); cx.rotate(-Math.PI/2); cx.textAlign='center';
  cx.fillText('file size',0,0); cx.restore();
}}
draw();
addEventListener('resize',draw);
new MutationObserver(draw).observe(document.documentElement,{{attributes:true,attributeFilter:['data-theme']}});
matchMedia('(prefers-color-scheme:dark)').addEventListener('change',draw);
</script>
'''

out = here / "dashboard.html"
out.write_text(html)
print(f"wrote {out}  {out.stat().st_size/1024/1024:.2f} MB")
