#!/usr/bin/env python3
"""Matched-fidelity comparison of the shape coder against WebP and AVIF.

Reads the two regenerated data files and interpolates each codec's rate curve at
every PSNR the shape coder actually hits, so no number in the report is
hand-interpolated. An earlier draft of this comparison was done by eye and
overstated the low-rate win by 7%.

It prints the same table twice, once against each codec sweep. The original sweep
left cwebp and avifenc below the settings anyone would actually ship, and the
difference between the two tables is larger than the effect the report was
claiming -- so both are shown rather than one being quietly swapped in.

AVIF is additionally reported as payload: its container floor is 297 B on an 8x8
image, which is 10-14% of a file at the bottom of this rate range, and comparing
a whole AVIF file against an idealised cross-entropy estimate would flatter the
shape coder for a reason that has nothing to do with coding.
"""
import re
import sys
from pathlib import Path

PIXELS = 512 * 288
AVIF_CONTAINER = 280  # measured floor 297 B, less a few bytes of real payload

here = Path(__file__).resolve().parent.parent


def read_frontier(path):
    """(psnr, bytes, regions) for the fully-relaxed variant at each scale-space mark."""
    rows = []
    for line in path.read_text().splitlines():
        if not line.startswith("+relax30"):
            continue
        f = line.split()
        regions, psnr, bpp = int(f[2]), float(f[3]), float(f[-1])
        rows.append((psnr, bpp * PIXELS / 8, regions))
    return sorted(rows)


def read_codecs(path):
    curves = {}
    for line in path.read_text().splitlines():
        m = re.match(r"^(webp|avif)\s+\d+\s+(\d+)\s+[\d.]+\s+([\d.]+)", line)
        if m:
            curves.setdefault(m.group(1), []).append((float(m.group(3)), int(m.group(2))))
    return {k: sorted(v) for k, v in curves.items()}


def interp(curve, psnr):
    """Bytes the codec needs for this PSNR. None outside the measured range -- no extrapolation."""
    for (p0, b0), (p1, b1) in zip(curve, curve[1:]):
        if p0 <= psnr <= p1:
            t = 0.0 if p1 == p0 else (psnr - p0) / (p1 - p0)
            return b0 + t * (b1 - b0)
    return None


def main():
    frontier = read_frontier(here / "04-region-merge-frontier-data.txt")

    # Both sweeps, because which one you use changes the answer by more than the effect being measured.
    # The original ran cwebp at its default method 4 and avifenc at -s 4; the strong sweep runs -m 6 and
    # -s 0, which is what anyone shipping these files would use.
    for label, path in (
        ("report 05 as first published  --  cwebp default (-m 4), avifenc -s 4",
         "05-codec-rd-sweep-data.txt"),
        ("strong settings               --  cwebp -m 6,           avifenc -s 0",
         "05-codec-rd-sweep-strong-data.txt"),
    ):
        codecs = read_codecs(here / path)
        print("=" * 78)
        print(label)
        print("=" * 78)
        print(f"{'PSNR':>6} {'regions':>8} {'shapes':>8} {'webp':>8} {'vs webp':>9} "
              f"{'avif_pay':>9} {'vs avif':>9}")
        crossover = None
        for psnr, sb, regions in frontier:
            wb = interp(codecs["webp"], psnr)
            ab = interp(codecs["avif"], psnr)
            if wb is None:
                continue
            ap = ab - AVIF_CONTAINER if ab is not None else None
            dw = (sb - wb) / wb * 100
            da = (sb - ap) / ap * 100 if ap else None
            if crossover is None and dw > 0:
                crossover = psnr
            print(f"{psnr:6.2f} {regions:8d} {sb:8.0f} {wb:8.0f} {dw:+8.1f}% "
                  f"{ap if ap else 0:9.0f} {da if da else 0:+8.1f}%")
        print()
        print(f"first PSNR at which WebP wins: {crossover:.2f} dB" if crossover
              else "the shape coder is smaller at every measured point")
        print()

    print("negative = shape coder smaller. AVIF column is payload (container discounted).")
    print("Note the sign alternates between adjacent samples under strong settings: that is a wash,")
    print("not a win. See report 05.")


if __name__ == "__main__":
    sys.exit(main())
