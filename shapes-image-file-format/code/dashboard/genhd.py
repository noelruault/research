#!/usr/bin/env python3
"""Builds hd-mirror.html: the 1:1 comparison at 3840x2160.

Every image embedded here is a native-resolution window of a real decoded file, encoded for display as
lossless WebP. Nothing is resampled and nothing is re-encoded lossily, so what the page shows is each
encoder's own artefacts and not the display pipeline's.

Numbers come from, and only from:
  hd/ladder_<W>.txt      shape-coder scale-space per resolution   (lab hd)
  hd/ladder-codecs.txt   WebP + AVIF rate-distortion per resolution (ladder-sweep.sh)
  hd/sweep-4k.txt        the full codec sweep at 4K                (sweep.sh)
  hd/mirror-stats.txt    per-window fidelity                       (mirror-build.sh)
"""
import base64
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
HD = HERE.parent / "hd"
PIX4K = 3840 * 2160


def uri(p):
    return "data:image/webp;base64," + base64.b64encode(pathlib.Path(p).read_bytes()).decode()


def kb(n):
    return f"{n / 1024:,.0f} KB" if n < 1024 * 1024 else f"{n / 1048576:.2f} MB"


# ---------------------------------------------------------------- measured inputs
LOSSLESS = [
    ("JPEG XL", "cjxl -d 0 -e 7", 6_468_598, "jxl"),
    ("WebP", "cwebp -lossless -z 9", 7_718_506, "webp"),
    ("AVIF", "avifenc --lossless", 11_969_137, "avif"),
    ("shape coder", "exact region partition", 12_159_385, "shapes"),
    ("PNG", "as delivered", 12_278_280, "png"),
]

# The exact partition, from ladder_3840.txt.
EXACT = dict(regions=6_356_392, px_per_region=1.305, cracks=14_508_938,
             walls=822_369, colours=11_337_015, bits_per_edge=0.4534, distinct=529_557)

WINDOWS = [
    dict(key="sky", title="Sky", where="288, 230",
         note="A smooth gradient with film grain. Nothing to segment: no edges, no regions, just a ramp.",
         hi=53.07, lo=36.90, webp=42.29, iso=41.61, mae_hi=0.310),
    dict(key="ridge", title="Sunlit ridge", where="1344, 1000",
         note="Dense texture at every scale, plus the hard terminator between lit rock and shadow.",
         hi=54.67, lo=26.01, webp=26.44, iso=25.57, mae_hi=0.204),
    dict(key="snow", title="Snow and rock", where="2400, 1350",
         note="Large near-flat snowfields split by sharp dark edges. The case a region model is built for.",
         hi=54.10, lo=26.82, webp=27.39, iso=26.61, mae_hi=0.234),
]

# Every panel, described once. Bytes are the WHOLE 3840x2160 file; "overall" dB is the whole image.
# The per-window dB lives in WINDOWS, because a 448x448 crop of a photograph is not representative of it
# and quoting one number for both would be the same sleight of hand this study spent nine corrections on.
PANELS = {
    "orig":     dict(name="WebP lossless", bytes=7_718_506, overall=None, win=None),
    "shapeshi": dict(name="shape coder",   bytes=8_055_367, overall=53.37, win="hi"),
    "diffhi":   dict(name="error &times;32", bytes=None,    overall=None, win=None),
    "webplo":   dict(name="WebP q14",      bytes=200_024,   overall=30.38, win="webp"),
    "webpiso":  dict(name="WebP q8",       bytes=165_042,   overall=29.52, win="iso"),
    "shapeslo": dict(name="shape coder",   bytes=203_511,   overall=29.44, win="lo"),
}

PAIRS = [
    dict(id="hi", left="orig", right="shapeshi",
         lname="WebP lossless", rname="shape coder",
         lmeta="7,718,506 B &middot; bit-exact", rmeta="8,055,367 B &middot; 53.37 dB",
         pick="same bytes, one is exact",
         blurb="The shape coder is given <b>more bytes than WebP needs to be perfect</b> &mdash; and is still not perfect."),
    dict(id="diff", left="orig", right="diffhi",
         lname="WebP lossless", rname="error &times;32",
         lmeta="7,718,506 B &middot; bit-exact", rmeta="peak error 3/255",
         pick="what 8 MB still gets wrong",
         blurb="Where those extra bytes went. Amplified 32&times;, because at 53 dB the error is below what an eye resolves."),
    dict(id="lo", left="webplo", right="shapeslo",
         lname="WebP q14", rname="shape coder",
         lmeta="200,024 B &middot; 30.38 dB", rmeta="203,511 B &middot; 29.44 dB",
         pick="same file size &mdash; judge the picture",
         blurb="Both files are ~200&nbsp;KB. WebP is <b>0.94 dB</b> ahead overall &mdash; and almost all of that gap is in one window."),
    dict(id="iso", left="webpiso", right="shapeslo",
         lname="WebP q8", rname="shape coder",
         lmeta="165,042 B &middot; 29.52 dB", rmeta="203,511 B &middot; 29.44 dB",
         pick="same quality &mdash; judge the weight",
         blurb="Matched on PSNR instead, so the file size is the answer: WebP needs <b>18.9% fewer bytes</b> for the same measured quality. Per window it is still 4.7&nbsp;dB ahead on the sky and now slightly <em>behind</em> on both texture windows &mdash; the same split, sharper."),
]

LADDER = [
    (28.7, [(512, 9_399, 9_382), (960, 23_172, 22_660), (1920, 64_112, 60_052), (3840, 163_471, 137_033)]),
    (30.0, [(512, 13_168, 12_363), (960, 33_193, 30_493), (1920, 93_104, 82_084), (3840, 245_180, 184_715)]),
    (31.5, [(512, 18_633, 17_060), (960, 47_951, 41_486), (1920, 139_056, 111_804), (3840, 381_339, 256_699)]),
    (34.0, [(512, 29_952, 30_500), (960, 80_350, 67_648), (1920, 247_112, 177_174), (3840, 726_980, 425_685)]),
]


def load_curve(path, codec, width):
    out = []
    for line in open(path):
        w, c, _s, b, p = line.split()
        if int(w) == width and c == codec and float(p) < 99:
            out.append((float(p), int(b)))
    return sorted(out)


def load_shapes(width):
    out = []
    for line in open(HD / f"ladder_{width}.txt"):
        f = line.split()
        if len(f) == 7 and f[0].isdigit():
            out.append((float(f[2]), int(f[5])))
    return sorted(out)


SHAPES4K = load_shapes(3840)
WEBP4K = load_curve(HD / "ladder-codecs.txt", "webp", 3840)
AVIF4K = load_curve(HD / "ladder-codecs.txt", "avif", 3840)

# ---------------------------------------------------------------- html


def lossless_rows():
    worst = max(r[2] for r in LOSSLESS)
    out = []
    for name, how, b, cls in LOSSLESS:
        ratio = b / 7_718_506
        mark = "&mdash;" if cls == "webp" else f"{ratio:.2f}&times;"
        hl = ' class="hl"' if cls == "shapes" else ""
        out.append(
            f'<tr{hl}><td><span class="dot {cls}"></span>{name}</td>'
            f'<td class="mono dim">{how}</td>'
            f'<td class="mono num">{b:,}</td>'
            f'<td class="bar"><i style="width:{100 * b / worst:.1f}%" class="{cls}"></i></td>'
            f'<td class="mono num">{mark}</td></tr>')
    return "\n".join(out)


def legend_rows(w):
    """One row per panel: what it weighs as a whole file, and how it scores overall and in this window."""
    out = []
    for k, m in PANELS.items():
        if k == "diffhi":
            wt, sc = "&mdash;", f"peak 3 &middot; mean {w['mae_hi']:.2f} /255"
        else:
            wt = f"{m['bytes']:,} B"
            sc = "exact" if m["overall"] is None else \
                 f"{w[m['win']]:.2f} dB here &middot; {m['overall']:.2f} overall"
        out.append(f'<div data-k="{k}"><dt>{m["name"]}<b>{wt}</b></dt><dd>{sc}</dd></div>')
    return "\n      ".join(out)


def window_blocks():
    out = []
    for w in WINDOWS:
        imgs = "".join(
            f'<img class="lyr" data-k="{k}" src="{uri(HD / "mirror" / f"{w["key"]}_{k}.webp")}" '
            f'alt="{w["title"]} &mdash; {k}" draggable="false">'
            for k in ("orig", "shapeshi", "shapeslo", "webplo", "webpiso", "diffhi"))
        out.append(f'''
<figure class="win" data-win="{w['key']}">
  <figcaption>
    <h3>{w['title']} <span class="coord mono">@ {w['where']}</span></h3>
    <p>{w['note']}</p>
    <p class="legkey">every panel, measured &mdash; lit rows are the two on screen</p>
    <dl class="wstat mono">
      {legend_rows(w)}
    </dl>
  </figcaption>
  <div class="stage" role="group" aria-label="{w['title']} comparison, drag to wipe">
    <div class="frame">
      {imgs}
      <div class="handle" tabindex="0" role="slider" aria-label="wipe position"
           aria-valuemin="0" aria-valuemax="100" aria-valuenow="50"><span></span></div>
      <div class="tagL mono"></div><div class="tagR mono"></div>
    </div>
    <p class="framenote">448&times;448 native pixels from the 3840&times;2160 image. Byte counts are the
      whole file; &ldquo;here&rdquo; is measured on this window alone.</p>
  </div>
</figure>''')
    return "\n".join(out)


def ladder_table():
    head = "".join(f"<th>{w}px</th>" for w, _, _ in LADDER[0][1])
    rows = []
    for psnr, pts in LADDER:
        cells = "".join(
            f'<td class="mono num {"bad" if s > c else "good"}">{100 * (s - c) / c:+.1f}%</td>'
            for _w, s, c in pts)
        rows.append(f"<tr><th>{psnr:.1f} dB</th>{cells}</tr>")
    return f"<thead><tr><th></th>{head}</tr></thead><tbody>{''.join(rows)}</tbody>"


html = f'''<title>1:1 &mdash; shapes against WebP at 4K</title>
<style>
:root {{
  --bg:#f4f6f9; --surface:#ffffff; --line:#dfe4ec; --line2:#eef1f6;
  --ink:#151922; --ink2:#5b6479; --ink3:#8b94a8;
  --shapes:#0e8fb8; --avif:#b8791a; --webp:#7c5cd0; --jxl:#2f8f6b; --png:#818b9e;
  --good:#1a7f47; --bad:#c2412f; --grid:#e6eaf1;
}}
@media (prefers-color-scheme:dark) {{
  :root {{
    --bg:#0f1218; --surface:#161b23; --line:#252c38; --line2:#1c222c;
    --ink:#e7eaf1; --ink2:#98a2b6; --ink3:#6c7689;
    --shapes:#4cc9f0; --avif:#f5a524; --webp:#a78bfa; --jxl:#3fb98c; --png:#8b95a8;
    --good:#3fb950; --bad:#f85149; --grid:#222937;
  }}
}}
:root[data-theme="dark"] {{
  --bg:#0f1218; --surface:#161b23; --line:#252c38; --line2:#1c222c;
  --ink:#e7eaf1; --ink2:#98a2b6; --ink3:#6c7689;
  --shapes:#4cc9f0; --avif:#f5a524; --webp:#a78bfa; --jxl:#3fb98c; --png:#8b95a8;
  --good:#3fb950; --bad:#f85149; --grid:#222937;
}}
:root[data-theme="light"] {{
  --bg:#f4f6f9; --surface:#ffffff; --line:#dfe4ec; --line2:#eef1f6;
  --ink:#151922; --ink2:#5b6479; --ink3:#8b94a8;
  --shapes:#0e8fb8; --avif:#b8791a; --webp:#7c5cd0; --jxl:#2f8f6b; --png:#818b9e;
  --good:#1a7f47; --bad:#c2412f; --grid:#e6eaf1;
}}
*,*::before,*::after {{ box-sizing:border-box; }}
body {{ margin:0; background:var(--bg); color:var(--ink);
  font:400 16px/1.6 ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;
  -webkit-font-smoothing:antialiased; }}
.mono {{ font-family:ui-monospace,SFMono-Regular,"SF Mono",Menlo,Consolas,monospace;
  font-variant-numeric:tabular-nums; }}
.wrap {{ max-width:1180px; margin:0 auto; padding:0 24px 96px; }}
a {{ color:var(--shapes); }}

header {{ padding:72px 0 40px; border-bottom:1px solid var(--line); }}
.eyebrow {{ font-family:ui-monospace,Menlo,monospace; font-size:11px; letter-spacing:.16em;
  text-transform:uppercase; color:var(--ink3); margin:0 0 20px; display:flex; gap:14px; flex-wrap:wrap; }}
.eyebrow a {{ color:var(--ink3); }}
h1 {{ margin:0 0 20px; font-size:clamp(2rem,5vw,3.4rem); line-height:1.03; letter-spacing:-.035em;
  font-weight:660; text-wrap:balance; max-width:18ch; }}
h1 b {{ color:var(--shapes); font-weight:660; }}
.lede {{ margin:0; max-width:66ch; font-size:1.075rem; color:var(--ink2); }}
.lede strong {{ color:var(--ink); font-weight:600; }}

.stats {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(200px,1fr)); gap:1px;
  background:var(--line); border:1px solid var(--line); margin:40px 0 0; }}
.stat {{ background:var(--surface); padding:20px 22px; }}
.stat .k {{ font-family:ui-monospace,Menlo,monospace; font-size:10.5px; letter-spacing:.14em;
  text-transform:uppercase; color:var(--ink3); }}
.stat .v {{ font-family:ui-monospace,Menlo,monospace; font-size:1.5rem; font-weight:600;
  letter-spacing:-.02em; margin-top:6px; font-variant-numeric:tabular-nums; }}
.stat .n {{ font-size:13px; color:var(--ink2); margin-top:4px; }}

section {{ padding:64px 0 0; }}
h2 {{ font-size:1.6rem; letter-spacing:-.02em; margin:0 0 8px; font-weight:640; }}
.sub {{ color:var(--ink2); margin:0 0 28px; max-width:70ch; }}

.tw {{ overflow-x:auto; border:1px solid var(--line); background:var(--surface); }}
table {{ border-collapse:collapse; width:100%; font-size:14.5px; min-width:560px; }}
th,td {{ padding:11px 16px; text-align:left; border-bottom:1px solid var(--line2); }}
thead th {{ font-size:10.5px; letter-spacing:.13em; text-transform:uppercase; color:var(--ink3);
  font-weight:600; border-bottom:1px solid var(--line); }}
tbody tr:last-child td, tbody tr:last-child th {{ border-bottom:0; }}
.num {{ text-align:right; }}
.dim {{ color:var(--ink3); font-size:12.5px; }}
tr.hl td {{ background:color-mix(in srgb, var(--shapes) 8%, transparent); }}
.good {{ color:var(--good); }} .bad {{ color:var(--bad); }}
.dot {{ display:inline-block; width:9px; height:9px; border-radius:2px; margin-right:9px;
  vertical-align:baseline; }}
.dot.shapes{{background:var(--shapes)}} .dot.webp{{background:var(--webp)}}
.dot.avif{{background:var(--avif)}} .dot.jxl{{background:var(--jxl)}} .dot.png{{background:var(--png)}}
td.bar {{ width:38%; }}
td.bar i {{ display:block; height:8px; border-radius:2px; opacity:.75; }}
i.shapes{{background:var(--shapes)}} i.webp{{background:var(--webp)}}
i.avif{{background:var(--avif)}} i.jxl{{background:var(--jxl)}} i.png{{background:var(--png)}}

/* ---- the mirror ---- */
.picker {{ display:flex; gap:0; border:1px solid var(--line); background:var(--surface);
  margin:0 0 8px; flex-wrap:wrap; }}
.picker button {{ flex:1 1 200px; appearance:none; border:0; background:transparent; cursor:pointer;
  padding:14px 18px; text-align:left; color:var(--ink2); font:inherit; font-size:14px;
  border-right:1px solid var(--line); }}
.picker button:last-child {{ border-right:0; }}
.picker button b {{ display:block; color:var(--ink); font-weight:600; font-size:14.5px; margin-bottom:2px; }}
.picker button span {{ font-size:12.5px; color:var(--ink3); }}
.picker button[aria-pressed="true"] {{ background:color-mix(in srgb, var(--shapes) 10%, transparent);
  box-shadow:inset 0 -2px 0 var(--shapes); color:var(--ink); }}
.picker button:focus-visible {{ outline:2px solid var(--shapes); outline-offset:-2px; }}
.blurb {{ margin:0 0 28px; color:var(--ink2); font-size:14.5px; }}
.blurb b {{ color:var(--ink); font-weight:600; }}

.win {{ margin:0 0 28px; border:1px solid var(--line); background:var(--surface);
  display:grid; grid-template-columns:minmax(0,1fr) minmax(0,auto); }}
@media (max-width:900px) {{ .win {{ grid-template-columns:minmax(0,1fr); }} }}
.win figcaption {{ padding:22px 24px; min-width:0; }}
.win h3 {{ margin:0 0 6px; font-size:1.1rem; font-weight:620; letter-spacing:-.01em; }}
.coord {{ color:var(--ink3); font-size:12px; font-weight:400; }}
.win p {{ margin:0 0 18px; color:var(--ink2); font-size:14px; max-width:42ch; }}
.wstat {{ margin:0; display:flex; flex-direction:column; gap:1px; background:var(--line);
  border:1px solid var(--line); font-size:12.5px; }}
.wstat div {{ display:flex; justify-content:space-between; gap:14px; background:var(--surface);
  padding:8px 12px; opacity:.34; }}
.wstat dt b {{ display:block; font-weight:600; color:var(--ink2); font-size:11.5px; }}
.wstat div.on {{ opacity:1; box-shadow:inset 3px 0 0 var(--shapes); }}
.legkey {{ margin:0 0 8px !important; font-size:11px; letter-spacing:.04em; color:var(--ink3);
  text-transform:uppercase; }}
.wstat dt {{ color:var(--ink3); }}
.wstat dd {{ margin:0; font-weight:600; }}
.wstat dd.best {{ color:var(--good); }}

.stage {{ padding:16px; border-left:1px solid var(--line); min-width:0; }}
@media (max-width:900px) {{ .stage {{ border-left:0; border-top:1px solid var(--line); }} }}
/* 448px is the source size, so at 448px and up one screen pixel is one image pixel. Narrower than that the frame scales: a comparison you cannot see on a phone is worse than one that is not literally 1:1. */
.frame {{ position:relative; width:min(448px,100%); aspect-ratio:1;
  overflow:hidden; user-select:none; touch-action:pan-y; cursor:ew-resize;
  background:var(--line2); }}
.frame img {{ position:absolute; inset:0; width:100%; height:100%; display:none; }}
.frame img.showR {{ display:block; }}
/* The left panel is revealed by clipping, not by nesting it in a resized container: one element, one coordinate system, nothing to fall out of alignment. */
.frame img.showL {{ display:block; z-index:2; clip-path:inset(0 calc(100% - var(--wipe,50%)) 0 0); }}
.handle {{ position:absolute; top:0; bottom:0; left:50%; width:2px; background:#fff;
  box-shadow:0 0 0 1px rgba(0,0,0,.45); transform:translateX(-1px); }}
.handle span {{ position:absolute; top:50%; left:50%; width:34px; height:34px; margin:-17px 0 0 -17px;
  border-radius:50%; background:#fff; box-shadow:0 1px 6px rgba(0,0,0,.4);
  display:grid; place-items:center; }}
.handle span::before {{ content:"\\2194"; color:#111; font-size:15px; line-height:1; }}
.handle:focus-visible {{ outline:2px solid var(--shapes); outline-offset:3px; }}
.tagL,.tagR {{ position:absolute; bottom:10px; font-size:11px; line-height:1.35;
  background:rgba(10,12,16,.82); color:#fff; padding:6px 9px; border-radius:3px;
  pointer-events:none; max-width:calc(50% - 14px); }}
.tagL {{ left:10px; }} .tagR {{ right:10px; text-align:right; }}
.tagL b,.tagR b {{ display:block; font-size:11.5px; letter-spacing:.05em; text-transform:uppercase;
  font-weight:700; }}
.tagL i,.tagR i {{ font-style:normal; opacity:.82; }}
.framenote {{ margin:12px 0 0 !important; font-size:11.5px; color:var(--ink3); max-width:448px; }}

.chart {{ border:1px solid var(--line); background:var(--surface); padding:20px; }}
canvas {{ display:block; width:100%; height:340px; }}
.legend {{ display:flex; gap:22px; flex-wrap:wrap; margin:14px 0 0; font-size:12.5px;
  color:var(--ink2); }}
.legend i {{ display:inline-block; width:14px; height:3px; border-radius:2px; margin-right:7px;
  vertical-align:middle; }}

.why {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(280px,1fr)); gap:1px;
  background:var(--line); border:1px solid var(--line); }}
.why div {{ background:var(--surface); padding:24px; }}
.why h4 {{ margin:0 0 8px; font-size:14px; font-weight:640; }}
.why p {{ margin:0; color:var(--ink2); font-size:14px; }}
.why code {{ font-family:ui-monospace,Menlo,monospace; font-size:12.5px; color:var(--ink); }}
footer {{ margin-top:72px; padding-top:28px; border-top:1px solid var(--line);
  color:var(--ink3); font-size:13px; }}
@media (prefers-reduced-motion:reduce) {{ * {{ transition:none !important; }} }}
</style>

<div class="wrap">
<header>
  <p class="eyebrow"><span>shapes-image-file-format &middot; report 08</span>
    <a href="dashboard.html">&larr; results dashboard</a></p>
  <h1>At 4K, <b>lossless</b> is where the shape idea runs out.</h1>
  <p class="lede">The earlier rounds ran on a 512&times;288 crop, where the region coder appeared to edge
    WebP below 29.2 dB &mdash; a result that did not survive taking WebP off its default encoder setting.
    This is the same picture at <strong>3840&times;2160</strong>, its native size, priced by the same
    coder. Every panel below is a <strong>1:1 window</strong> &mdash; one screen pixel is one image
    pixel, no resampling, and the display copies are lossless so the artefacts are each encoder's own.</p>
  <div class="stats">
    <div class="stat"><div class="k">exact partition</div><div class="v">{EXACT['regions']:,}</div>
      <div class="n">regions for a bit-exact image &mdash; {EXACT['px_per_region']} px each</div></div>
    <div class="stat"><div class="k">shapes, lossless</div><div class="v">{kb(12_159_385)}</div>
      <div class="n">1.58&times; WebP lossless, 1.88&times; JPEG XL</div></div>
    <div class="stat"><div class="k">walls vs colours</div><div class="v">93%</div>
      <div class="n">of those bytes are colour; the geometry is nearly free</div></div>
    <div class="stat"><div class="k">deficit at 28.7 dB</div><div class="v">+19.3%</div>
      <div class="n">vs WebP at 4K &mdash; against +0.2% at 512&times;288</div></div>
  </div>
</header>

<section>
  <h2>Being exactly right</h2>
  <p class="sub">What each encoder costs to reproduce all 8,294,400 pixels with zero error. The shape coder
    is priced with this study's own crack-edge and colour coders, with no container and no header &mdash;
    an idealised lower bound, competing against real files.</p>
  <div class="tw"><table>
    <thead><tr><th>encoder</th><th>setting</th><th class="num">bytes</th><th></th>
      <th class="num">vs WebP</th></tr></thead>
    <tbody>{lossless_rows()}</tbody>
  </table></div>
  <div class="why" style="margin-top:20px">
    <div><h4>The partition degenerates</h4><p>To be exact, every run of identical pixels becomes its own
      region: <code>{EXACT['regions']:,}</code> of them across {PIX4K:,} pixels, averaging
      <code>{EXACT['px_per_region']}</code> px. There is no geometry left to exploit.</p></div>
    <div><h4>So the walls become free</h4><p>{EXACT['cracks']:,} crack edges cost
      <code>{EXACT['bits_per_edge']}</code> bits each &mdash; when almost every pixel boundary is a wall,
      the wall map is nearly constant and compresses to {kb(EXACT['walls'])}.</p></div>
    <div><h4>And colour is the whole bill</h4><p>{kb(EXACT['colours'])} of {kb(12_159_385)}. At this point
      the coder <em>is</em> a raster coder, with one predictor where WebP has fourteen, plus a colour
      transform, LZ77 with 2D distances, and a colour cache.</p></div>
  </div>
</section>

<section>
  <h2>1:1</h2>
  <p class="sub">Drag to wipe. Three windows of 448&times;448 native pixels, chosen for the three ways a
    piecewise-constant model can fail. At 448&nbsp;px and wider each panel is exactly 1:1; on a narrower screen it scales to fit.</p>
  <div class="picker" role="group" aria-label="choose comparison">
    {"".join(f'<button data-pair="{p["id"]}" aria-pressed="{"true" if i == 0 else "false"}">'
             f'<b>{p["lname"]} vs {p["rname"]}</b><span>{p["lmeta"]}</span></button>'
             for i, p in enumerate(PAIRS))}
  </div>
  <p class="blurb" id="blurb">{PAIRS[0]['blurb']}</p>
  {window_blocks()}
</section>

<section>
  <h2>The 512&times;288 win does not survive</h2>
  <p class="sub">The same photograph, resampled to four sizes, each encoded by both coders and read at a
    fixed PSNR so the same picture quality is being bought at every rung. Positive means the shape coder
    needs more bytes than WebP for the same result.</p>
  <div class="chart">
    <canvas id="cv" width="1100" height="340"></canvas>
    <div class="legend">
      {"".join(f'<span><i style="background:hsl({200 - k * 34} 70% 50%)"></i>{p:.1f} dB</span>'
               for k, (p, _) in enumerate(LADDER))}
    </div>
  </div>
  <div class="tw" style="margin-top:20px"><table>{ladder_table()}</table></div>
  <p class="sub" style="margin-top:20px">Boundary length grows with the linear dimension and area with its
    square, so a 16&times; pixel count should only be a 4&times; boundary cost &mdash; the isoperimetric
    term that sinks the shape coder ought to <em>weaken</em> at 4K. It does. The deficit still grows,
    because WebP's own cost per pixel falls faster: at 4K it has more correlated neighbourhood to predict
    from, longer LZ77 matches, and more tiles to segment its entropy image over, while the shape coder
    gains only a slightly better area-to-perimeter ratio.</p>
</section>

<section>
  <h2>Where the bytes actually go</h2>
  <p class="sub">Whole-image PSNR at the 200 KB operating point says WebP is 0.94 dB ahead. Per window it
    says something more specific.</p>
  <div class="tw"><table>
    <thead><tr><th>window</th><th>content</th><th class="num">shapes</th><th class="num">WebP</th>
      <th class="num">gap</th></tr></thead>
    <tbody>
    {"".join(f'<tr><td>{w["title"]}</td><td class="dim">{w["note"].split(".")[0]}</td>'
             f'<td class="mono num">{w["lo"]:.2f} dB</td><td class="mono num">{w["webp"]:.2f} dB</td>'
             f'<td class="mono num bad">{w["lo"] - w["webp"]:+.2f} dB</td></tr>' for w in WINDOWS)}
    </tbody>
  </table></div>
  <div class="why" style="margin-top:20px">
    <div><h4>Texture is a near-tie</h4><p>On the ridge and the snowfield the shape coder is within
      0.6 dB. Sharp edges and busy detail are what a region model is for, and it holds its own.</p></div>
    <div><h4>The gradient is the loss</h4><p>The sky costs it <b>5.39 dB</b>. A piecewise-constant model
      has one way to draw a smooth ramp &mdash; stack flat bands and pay for every boundary. A DCT spends
      one low-frequency coefficient.</p></div>
    <div><h4>Which is fixable, and still not enough</h4><p>Report 04 priced the affine variant: linear
      colour per region closes most of the gradient gap and costs more in coefficients than it saves.
      The bands are a symptom; the explicit boundary is the disease.</p></div>
  </div>
</section>

<footer>
  <p>Source: the macOS Sierra wallpaper at 3840&times;2160, decoded from a q95 JPEG &mdash; so the
  &ldquo;original&rdquo; carries JPEG artefacts and film grain that a downscale would have averaged away.
  Both coders see the identical PNG. Shape-coder bytes are contour + colour with no container; WebP, AVIF,
  JPEG XL and PNG bytes are whole files. PSNR is this study's RGB definition throughout. Codec settings:
  <span class="mono">cwebp -m 6</span>, <span class="mono">avifenc -s 6</span>,
  <span class="mono">cjxl -e 7</span>.</p>
</footer>
</div>

<script>
const PAIRS = {json.dumps({p["id"]: p for p in PAIRS})};
const PANELS = {json.dumps({k: {"name": v["name"], "bytes": f"{v['bytes']:,}" if v["bytes"] else None} for k, v in PANELS.items()})};
const STATS = {json.dumps({w["key"]: {k: ("exact, 0 error" if k == "orig" else
                 (f"peak 3/255 here" if k == "diffhi" else f"{w[v['win']]:.2f} dB here"))
                 for k, v in PANELS.items()} for w in WINDOWS})};
let cur = "hi";

function apply() {{
  const p = PAIRS[cur];
  document.getElementById("blurb").innerHTML = p.blurb;
  document.querySelectorAll(".picker button").forEach(b =>
    b.setAttribute("aria-pressed", b.dataset.pair === cur ? "true" : "false"));
  document.querySelectorAll(".win").forEach(win => {{
    const frame = win.querySelector(".frame");
    frame.querySelectorAll("img.lyr").forEach(img => {{
      img.classList.toggle("showR", img.dataset.k === p.right);
      img.classList.toggle("showL", img.dataset.k === p.left);
    }});
    win.querySelectorAll(".wstat div").forEach(d =>
      d.classList.toggle("on", d.dataset.k === p.left || d.dataset.k === p.right));
    const tag = (el, k) => {{
      const m = PANELS[k], st = STATS[win.dataset.win];
      el.innerHTML = "<b>" + m.name + "</b><i>" + (m.bytes ? m.bytes + " B" : "&mdash;") +
        (st[k] ? "<br>" + st[k] : "") + "</i>";
    }};
    tag(frame.querySelector(".tagL"), p.left);
    tag(frame.querySelector(".tagR"), p.right);
  }});
}}

function wire(frame) {{
  const handle = frame.querySelector(".handle");
  const set = pct => {{
    pct = Math.max(0, Math.min(100, pct));
    frame.style.setProperty("--wipe", pct + "%");
    handle.style.left = pct + "%";
    handle.setAttribute("aria-valuenow", Math.round(pct));
  }};
  const at = e => {{
    const r = frame.getBoundingClientRect();
    set(((e.touches ? e.touches[0].clientX : e.clientX) - r.left) / r.width * 100);
  }};
  frame.addEventListener("pointerdown", e => {{ frame.setPointerCapture(e.pointerId); at(e); }});
  frame.addEventListener("pointermove", e => {{ if (e.buttons) at(e); }});
  handle.addEventListener("keydown", e => {{
    const step = e.shiftKey ? 10 : 2;
    const now = parseFloat(handle.style.left) || 50;
    if (e.key === "ArrowLeft") {{ set(now - step); e.preventDefault(); }}
    if (e.key === "ArrowRight") {{ set(now + step); e.preventDefault(); }}
  }});
  set(50);
}}

document.querySelectorAll(".picker button").forEach(b =>
  b.addEventListener("click", () => {{ cur = b.dataset.pair; apply(); }}));
document.querySelectorAll(".frame").forEach(wire);
apply();

// ---- the ladder chart ----
const LADDER = {json.dumps([[p, [[w, s, c] for w, s, c in pts]] for p, pts in LADDER])};
const cv = document.getElementById("cv"), cx = cv.getContext("2d");
const css = v => getComputedStyle(document.documentElement).getPropertyValue(v).trim();

function draw() {{
  const dpr = devicePixelRatio || 1;
  const W = cv.clientWidth, H = 340;
  cv.width = W * dpr; cv.height = H * dpr; cx.setTransform(dpr, 0, 0, dpr, 0, 0);
  cx.clearRect(0, 0, W, H);
  const L = 62, R = W - 18, T = 18, B = H - 34;
  const xs = [512, 960, 1920, 3840];
  const X = i => L + (R - L) * i / (xs.length - 1);
  const maxY = 80, Y = v => B - (B - T) * v / maxY;

  cx.strokeStyle = css("--grid"); cx.lineWidth = 1;
  cx.fillStyle = css("--ink3"); cx.font = '11px ui-monospace,Menlo,monospace';
  for (let v = 0; v <= maxY; v += 20) {{
    cx.beginPath(); cx.moveTo(L, Y(v)); cx.lineTo(R, Y(v)); cx.stroke();
    cx.textAlign = "right"; cx.textBaseline = "middle"; cx.fillText(v + "%", L - 10, Y(v));
  }}
  // Zero is the line that matters: below it the shape coder is genuinely smaller.
  cx.strokeStyle = css("--ink3"); cx.setLineDash([3, 3]);
  cx.beginPath(); cx.moveTo(L, Y(0)); cx.lineTo(R, Y(0)); cx.stroke(); cx.setLineDash([]);

  cx.textAlign = "center"; cx.textBaseline = "top"; cx.fillStyle = css("--ink3");
  xs.forEach((w, i) => cx.fillText(w + "\\u00d7" + (w * 9 / 16), X(i), B + 10));

  LADDER.forEach(([psnr, pts], k) => {{
    const col = `hsl(${{200 - k * 34}} 70% 50%)`;
    cx.strokeStyle = col; cx.lineWidth = 2.4; cx.lineJoin = "round"; cx.beginPath();
    pts.forEach(([w, s, c], i) => {{
      const v = 100 * (s - c) / c;
      i ? cx.lineTo(X(i), Y(v)) : cx.moveTo(X(i), Y(v));
    }});
    cx.stroke();
    cx.fillStyle = col;
    pts.forEach(([w, s, c], i) => {{
      const v = 100 * (s - c) / c;
      cx.beginPath(); cx.arc(X(i), Y(v), 3.2, 0, 7); cx.fill();
    }});
  }});

  cx.save(); cx.translate(15, H / 2); cx.rotate(-Math.PI / 2);
  cx.textAlign = "center"; cx.fillStyle = css("--ink3");
  cx.fillText("extra bytes vs WebP, same PSNR", 0, 0); cx.restore();
}}
draw();
addEventListener("resize", draw);
new MutationObserver(draw).observe(document.documentElement,
  {{ attributes: true, attributeFilter: ["data-theme"] }});
matchMedia("(prefers-color-scheme:dark)").addEventListener("change", draw);
</script>
'''

out = HERE / "hd-mirror.html"
out.write_text(html)
print(f"wrote {out}  {out.stat().st_size / 1048576:.2f} MB")
