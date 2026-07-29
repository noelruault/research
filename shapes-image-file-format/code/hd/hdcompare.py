#!/usr/bin/env python3
"""Matched-fidelity comparison of the shape coder against the shipped codecs, across a resolution ladder.

Every number here is interpolated from measured points, never extrapolated past the ends of a codec's
measured range, because the one time this study compared curves by eye it overstated the shape coder's
win by more than 3x.

Inputs, all produced by the scripts beside this file:
  ladder_<W>.txt   scale-space of the shape coder at width W  (./lab hd)
  ladder-codecs.txt  W codec setting bytes psnr                (./ladder-sweep.sh)
"""
import os

HERE = os.path.dirname(os.path.abspath(__file__))
WIDTHS = [3840, 1920, 960, 512]


def px(w):
    return w * w * 9 // 16


def load_shapes(w):
    """(psnr, bytes, regions) for each mark of the merge scale-space, plus the lossless point."""
    pts, lossless = [], None
    for line in open(f"{HERE}/ladder_{w}.txt"):
        if line.startswith("lossless_total_B"):
            lossless = float(line.split()[1])
        f = line.split()
        if len(f) == 7 and f[0].isdigit():
            regions, _crack, psnr, _wb, _cb, total, _bpp = f
            pts.append((float(psnr), float(total), int(regions)))
    pts.sort()
    return pts, lossless


def load_codecs():
    """{width: {codec: [(psnr, bytes), ...]}} — lossless entries kept separately, they have no PSNR."""
    curves, lossless = {}, {}
    for line in open(f"{HERE}/ladder-codecs.txt"):
        w, codec, _setting, b, p = line.split()
        w, b, p = int(w), int(b), float(p)
        if p >= 99:
            lossless.setdefault(w, {})[codec] = b
        else:
            curves.setdefault(w, {}).setdefault(codec, []).append((p, b))
    for w in curves:
        for c in curves[w]:
            curves[w][c].sort()
    return curves, lossless


def interp(curve, psnr):
    """Bytes for this codec at the given PSNR. None outside the measured range — never extrapolate."""
    if not curve or psnr < curve[0][0] or psnr > curve[-1][0]:
        return None
    for (p0, b0), (p1, b1) in zip(curve, curve[1:]):
        if p0 <= psnr <= p1:
            if p1 == p0:
                return b0
            return b0 + (b1 - b0) * (psnr - p0) / (p1 - p0)
    return None


def main():
    curves, lossless = load_codecs()

    print("=" * 100)
    print("LOSSLESS: what an exact region partition costs, against the codecs that are actually bit-exact")
    print("=" * 100)
    print(f"{'width':>6} {'shapes':>12} {'webp-ll':>12} {'png':>12} {'vs webp':>9} {'regions/px':>11}")
    for w in WIDTHS:
        pts, ll = load_shapes(w)
        wl = lossless.get(w, {}).get("webp-ll")
        src = os.path.getsize(f"{HERE}/src4k.png" if w == 3840 else f"{HERE}/src{w}.png")
        rpp = None
        for line in open(f"{HERE}/ladder_{w}.txt"):
            if line.startswith("px_per_region"):
                rpp = 1 / float(line.split()[1])
        print(f"{w:>6} {ll:>12,.0f} {wl:>12,} {src:>12,} {ll / wl:>8.2f}x {rpp:>11.3f}")

    print()
    print("=" * 100)
    print("LOSSY: shape-coder bytes vs each codec at the SAME PSNR, per resolution")
    print("negative = shape coder smaller.  '-' = the codec has no measured operating point at that fidelity")
    print("=" * 100)
    for w in WIDTHS:
        pts, _ = load_shapes(w)
        print(f"\n--- {w}x{w * 9 // 16}  ({px(w):,} px) ---")
        print(f"{'regions':>9} {'psnr':>7} {'shapes B':>11} {'webp B':>11} {'vs webp':>9} "
              f"{'avif B':>11} {'vs avif':>9}")
        for psnr, sb, regions in pts:
            row = f"{regions:>9,} {psnr:>7.2f} {sb:>11,.0f}"
            for codec in ("webp", "avif"):
                cb = interp(curves.get(w, {}).get(codec, []), psnr)
                if cb is None:
                    row += f" {'-':>11} {'-':>9}"
                else:
                    row += f" {cb:>11,.0f} {100 * (sb - cb) / cb:>8.1f}%"
            print(row)

    print()
    print("=" * 100)
    print("THE LADDER: does the shape coder's standing improve or decay with resolution?")
    print("Read at fixed PSNR, so the same picture quality is being bought at every rung.")
    print("=" * 100)
    for target in (28.7, 30.0, 31.5, 34.0):
        print(f"\nat {target:.1f} dB:")
        print(f"{'width':>6} {'shapes B':>11} {'webp B':>11} {'vs webp':>9} {'avif B':>11} {'vs avif':>9}")
        for w in WIDTHS:
            pts, _ = load_shapes(w)
            sb = interp([(p, b) for p, b, _ in pts], target)
            if sb is None:
                print(f"{w:>6} {'out of range':>11}")
                continue
            row = f"{w:>6} {sb:>11,.0f}"
            for codec in ("webp", "avif"):
                cb = interp(curves.get(w, {}).get(codec, []), target)
                row += f" {cb:>11,.0f} {100 * (sb - cb) / cb:>8.1f}%" if cb else f" {'-':>11} {'-':>9}"
            print(row)


if __name__ == "__main__":
    main()
