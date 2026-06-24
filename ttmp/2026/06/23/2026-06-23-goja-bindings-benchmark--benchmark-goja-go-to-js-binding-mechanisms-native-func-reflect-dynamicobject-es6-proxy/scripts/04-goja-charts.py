"""04-goja-charts.py — SVG chart generators for the goja-bindings report.

Pure functions returning inline SVG strings, scaled to the real data produced
by 03-parse-bench-stats.py. Deterministic and dependency-free (no charting
library, no JavaScript) so the final report is a single static HTML file.

Loaded by 06-make-report.py via importlib (the numeric prefix is not a valid
Python module name, so it cannot be `import`ed directly).

Each "rows" argument is an ordered list of (display_name, stat_dict) tuples,
where stat_dict comes from various/03-stats.json and has the keys:
  mean, min, max, median, cv, B, allocs
"""
import math

ACCENT = "#E8470C"
INK = "#0a0a0a"
HAIR = "#e5e5e5"
MUTED = "#6b6b6b"
MONO = "ui-monospace,SFMono-Regular,Menlo,Consolas,monospace"


def _esc(s):
    return str(s).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def _svg_open(W, H, aria):
    return (f'<svg viewBox="0 0 {W} {H}" xmlns="http://www.w3.org/2000/svg" '
            f'role="img" aria-label="{_esc(aria)}" '
            f'style="width:100%;height:auto;display:block;max-width:{W}px;margin:8px 0">')


# ---------------------------------------------------------------------------
# Figure 1 — function-call cost, log-scale horizontal bars
# ---------------------------------------------------------------------------
def calls_logbars(rows):
    W, H, L, R, T, B = 760, 330, 170, 250, 26, 30
    pw, ph = W - L - R, H - T - B
    n = len(rows)
    rh = ph / n
    lo, hi = math.log10(150), math.log10(2400)

    def x(v):
        return L + (math.log10(v) - lo) / (hi - lo) * pw

    p = [_svg_open(W, H, "Function-call cost per mechanism, log scale")]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    for t in [200, 300, 500, 1000, 2000]:
        xt = x(t)
        p.append(f'<line x1="{xt:.1f}" y1="{T}" x2="{xt:.1f}" y2="{T+ph}" stroke="{HAIR}"/>')
        p.append(f'<text x="{xt:.1f}" y="{H-9}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="middle">{t}</text>')
    p.append(f'<text x="{L+pw/2}" y="{H+2}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle">nanoseconds per call &#160;(log scale)</text>')
    for i, (name, r) in enumerate(rows):
        cy = T + rh * i + rh / 2
        bx, bw = x(r["mean"]), x(r["mean"]) - L
        col = ACCENT if name in ("Native func", "Proxy (Apply)") else INK
        p.append(f'<rect x="{L}" y="{cy-10:.1f}" width="{bw:.1f}" height="20" fill="{col}"/>')
        p.append(f'<text x="{L-12}" y="{cy+4:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="end">{_esc(name)}</text>')
        p.append(f'<text x="{bx+9:.1f}" y="{cy+4:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}">{r["mean"]:.0f} ns &#160;&#183;&#160; {r["allocs"]} allocs</text>')
    return "\n".join(p) + "</svg>"


# ---------------------------------------------------------------------------
# Figure 2 — allocations vs latency scatter (the scientific finding)
# ---------------------------------------------------------------------------
def scatter_alloc_time(calls, reads):
    W, H, L, R, T, B = 720, 384, 58, 210, 26, 46
    pw, ph = W - L - R, H - T - B
    xmax, ymax = 26, 2100

    def sx(v):
        return L + v / xmax * pw

    def sy(v):
        return T + ph - v / ymax * ph

    p = [_svg_open(W, H, "Allocations per operation versus latency, by regime")]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    for gx in [0, 5, 10, 15, 20, 25]:
        xg = sx(gx)
        p.append(f'<line x1="{xg:.1f}" y1="{T}" x2="{xg:.1f}" y2="{T+ph}" stroke="{HAIR}"/>'
                 f'<text x="{xg:.1f}" y="{H-26}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="middle">{gx}</text>')
    for gy in [0, 500, 1000, 1500, 2000]:
        yg = sy(gy)
        p.append(f'<line x1="{L}" y1="{yg:.1f}" x2="{L+pw}" y2="{yg:.1f}" stroke="{HAIR}"/>'
                 f'<text x="{L-9}" y="{yg+4:.1f}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="end">{gy}</text>')
    p.append(f'<text x="{L-42}" y="{T+ph/2}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle" '
             f'transform="rotate(-90 {L-42} {T+ph/2})">nanoseconds per operation</text>')
    p.append(f'<text x="{L+pw/2}" y="{H-6}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle">allocations per operation</text>')
    # call-regime reference ray (~86 ns/alloc), fit from the reflect-method point
    p.append(f'<line x1="{sx(0):.1f}" y1="{sy(0):.1f}" x2="{sx(xmax):.1f}" '
             f'y2="{sy(85.6*xmax):.1f}" stroke="{ACCENT}" stroke-width="1.3" '
             f'stroke-dasharray="6 4" opacity="0.85"/>')
    p.append(f'<text x="{sx(24.5):.1f}" y="{sy(85.6*24.5)+2:.1f}" font-family="{MONO}" '
             f'font-size="10" fill="{ACCENT}" text-anchor="end">'
             f'~86 ns/alloc (call regime)</text>')

    def pt(r, col, label, lx=10, ly=4):
        cx, cy = sx(r["allocs"]), sy(r["mean"])
        return (f'<circle cx="{cx:.1f}" cy="{cy:.1f}" r="5.5" fill="#fff" '
                f'stroke="{col}" stroke-width="1.7"/>'
                f'<text x="{cx+lx:.1f}" y="{cy+ly:.1f}" font-family="{MONO}" '
                f'font-size="10.5" fill="{INK}">{_esc(label)}</text>')

    # Calls are spread along the trend line and are the primary story, so each
    # gets an individual label (low-alloc calls to the left to avoid the read
    # cluster, high-alloc calls to the right).
    for n, r in calls:
        lx = -10 if r["allocs"] <= 12 else 10
        ta = "end" if lx < 0 else "start"
        cx, cy = sx(r["allocs"]), sy(r["mean"])
        p.append(f'<circle cx="{cx:.1f}" cy="{cy:.1f}" r="5.5" fill="#fff" '
                 f'stroke="{ACCENT}" stroke-width="1.7"/>')
        p.append(f'<text x="{cx+lx:.1f}" y="{cy+4:.1f}" font-family="{MONO}" '
                 f'font-size="10.5" fill="{INK}" text-anchor="{ta}">{_esc(n)}</text>')
    # Reads cluster tightly at low latency (a finding, not a labeling target).
    # Plot them as unlabeled points and annotate the cluster; Figure 3 gives the
    # per-mechanism breakdown, so labeling them here would only add clutter.
    for n, r in reads:
        cx, cy = sx(r["allocs"]), sy(r["mean"])
        p.append(f'<circle cx="{cx:.1f}" cy="{cy:.1f}" r="5" fill="#fff" '
                 f'stroke="{INK}" stroke-width="1.5"/>')
    # annotation pointing at the read cluster
    ax, ay = sx(1.5), sy(150)
    p.append(f'<text x="{ax+14:.1f}" y="{ay:.1f}" font-family="{MONO}" font-size="10.5" '
             f'fill="{MUTED}">property reads</text>')
    p.append(f'<text x="{ax+14:.1f}" y="{ay+13:.1f}" font-family="{MONO}" font-size="10.5" '
             f'fill="{MUTED}">(see Fig. 3)</text>')
    p.append(f'<rect x="{L+8}" y="{T+6}" width="13" height="13" fill="#fff" '
             f'stroke="{ACCENT}" stroke-width="1.7"/>'
             f'<text x="{L+28}" y="{T+17}" font-family="{MONO}" font-size="11" '
             f'fill="{INK}">function calls</text>')
    p.append(f'<rect x="{L+8}" y="{T+26}" width="13" height="13" fill="#fff" '
             f'stroke="{INK}" stroke-width="1.7"/>'
             f'<text x="{L+28}" y="{T+37}" font-family="{MONO}" font-size="11" '
             f'fill="{INK}">property reads</text>')
    return "\n".join(p) + "</svg>"


# ---------------------------------------------------------------------------
# Figures 3 & 4 — shared dot+range plot (min—median—max, mean as a tick)
# ---------------------------------------------------------------------------
def dotrange(rows, xmin, xmax, title):
    rows = list(rows)
    n = len(rows)
    W, H, L, R, T, B = 720, 62 + 42 * n, 158, 84, 26, 36
    pw, ph = W - L - R, H - T - B
    rh = ph / n

    def sx(v):
        return L + (v - xmin) / (xmax - xmin) * pw

    p = [_svg_open(W, H, title)]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    # "nice" ticks
    span = xmax - xmin
    step = 10 ** math.floor(math.log10(span))
    if span / step > 8:
        step *= 2
    if span / step < 3:
        step /= 2
    t, ticks = math.ceil(xmin / step), []
    while t * step <= xmax + 1e-9:
        ticks.append(round(t * step, 3))
        t += 1
    for tk in ticks:
        xt = sx(tk)
        p.append(f'<line x1="{xt:.1f}" y1="{T}" x2="{xt:.1f}" y2="{T+ph}" stroke="{HAIR}"/>'
                 f'<text x="{xt:.1f}" y="{H-13}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="middle">{tk:g}</text>')
    p.append(f'<text x="{L+pw/2}" y="{H+0}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle">'
             f'nanoseconds &#160;(min &#160;&#8212;&#160; median &#160;&#8212;&#160; max)</text>')
    for i, (name, r) in enumerate(rows):
        cy = T + rh * i + rh / 2
        xmn, xmx = sx(r["min"]), sx(r["max"])
        xmed, xmean = sx(r["median"]), sx(r["mean"])
        p.append(f'<text x="{L-12}" y="{cy+4:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="end">{_esc(name)}</text>')
        p.append(f'<line x1="{xmn:.1f}" y1="{cy:.1f}" x2="{xmx:.1f}" y2="{cy:.1f}" '
                 f'stroke="{INK}" stroke-width="1.4"/>')
        p.append(f'<line x1="{xmn:.1f}" y1="{cy-5:.1f}" x2="{xmn:.1f}" y2="{cy+5:.1f}" '
                 f'stroke="{INK}" stroke-width="1.4"/>')
        p.append(f'<line x1="{xmx:.1f}" y1="{cy-5:.1f}" x2="{xmx:.1f}" y2="{cy+5:.1f}" '
                 f'stroke="{INK}" stroke-width="1.4"/>')
        p.append(f'<circle cx="{xmed:.1f}" cy="{cy:.1f}" r="5" fill="{ACCENT}"/>')
        p.append(f'<line x1="{xmean:.1f}" y1="{cy-8:.1f}" x2="{xmean:.1f}" y2="{cy+8:.1f}" '
                 f'stroke="{INK}" stroke-width="1"/>')
    p.append(f'<circle cx="{L+pw-156}" cy="{H-30}" r="5" fill="{ACCENT}"/>'
             f'<text x="{L+pw-144}" y="{H-26}" font-family="{MONO}" font-size="10.5" '
             f'fill="{MUTED}">median</text>')
    p.append(f'<line x1="{L+pw-90}" y1="{H-34}" x2="{L+pw-90}" y2="{H-26}" stroke="{INK}"/>'
             f'<text x="{L+pw-82}" y="{H-26}" font-family="{MONO}" font-size="10.5" '
             f'fill="{MUTED}">mean</text>')
    return "\n".join(p) + "</svg>"
