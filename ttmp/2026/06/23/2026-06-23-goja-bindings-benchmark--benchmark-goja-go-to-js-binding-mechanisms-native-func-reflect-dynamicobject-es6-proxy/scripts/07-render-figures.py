#!/usr/bin/env python3
# 07-render-figures.py
#
# Stage 4 of the report pipeline. Extracts each of the four inline SVG figures
# from the generated report (various/06-bindings-report.html) into a standalone
# SVG file, then rasterizes each to a high-resolution PNG.
#
# Rasterization uses a headless Chromium via Playwright (deviceScaleFactor 2 for
# crisp output), because the SVGs use HTML entities and CSS that a plain SVG
# rasterizer may not expand. Each figure is wrapped in a minimal white-background
# HTML page so the screenshot is deterministic and borderless.
#
#   report  (various/06-bindings-report.html)
#   -> standalone svgs  (various/figures/figN.svg)
#   -> rasterized pngs  (various/figures/figN.png)
#
# Rasterization uses Inkscape headless export (rsvg/cairo-quality, no browser
# dependency) at a fixed wide pixel width so the PNGs stay crisp when embedded
# in the Obsidian vault at display size. Requires `inkscape` on PATH.
#
# Outputs are written into a destination directory (default: the ticket's
# various/figures/). Copy the PNGs into the Obsidian vault's Attachments/ when
# embedding them.
#
#   python3 scripts/07-render-figures.py
import re
import shutil
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
TICKET = HERE.parent
VAR = TICKET / "various"
REPORT = VAR / "06-bindings-report.html"
FIGDIR = VAR / "figures"

CAPTIONS = {
    1: "fig1-call-cost-logscale",
    2: "fig2-allocs-vs-latency",
    3: "fig3-read-latency",
    4: "fig4-proxy-dispatch",
}


def extract_svgs(report_text):
    return re.findall(r"<svg.*?</svg>", report_text, re.S)


def render_all(svgs, outdir, width=1800):
    if not shutil.which("inkscape"):
        sys.exit("ERROR: inkscape not found on PATH; install it to rasterize SVGs.")
    outdir.mkdir(parents=True, exist_ok=True)
    for i, svg in enumerate(svgs, 1):
        slug = CAPTIONS.get(i, f"fig{i}")
        svg_path = outdir / f"{slug}.svg"
        svg_path.write_text(svg)
        png_path = outdir / f"{slug}.png"
        # Inkscape 1.x headless export. Exporting by width keeps the aspect ratio
        # from the viewBox and produces crisp text at display size.
        subprocess.run(
            ["inkscape", str(svg_path),
             "--export-type=png",
             f"--export-filename={png_path}",
             f"--export-width={width}"],
            check=True, capture_output=True)
        print(f"  fig {i}: wrote {png_path.name} ({png_path.stat().st_size//1024} KiB)")


def main():
    text = REPORT.read_text()
    svgs = extract_svgs(text)
    if len(svgs) != 4:
        sys.exit(f"ERROR: expected 4 svgs, found {len(svgs)}")
    print(f"rendering {len(svgs)} figures into {FIGDIR} ...")
    render_all(svgs, FIGDIR)
    print(f"\nDone. PNGs in {FIGDIR.relative_to(TICKET.parent.parent.parent)}")


if __name__ == "__main__":
    sys.exit(main())
