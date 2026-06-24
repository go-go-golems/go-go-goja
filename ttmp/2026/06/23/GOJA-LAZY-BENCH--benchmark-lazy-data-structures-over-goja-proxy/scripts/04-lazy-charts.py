"""04-lazy-charts.py — SVG chart generators for the lazy-data-structures report.

Pure functions returning inline SVG strings, scaled to the real data produced
by 03-parse-lazy-stats.py. Deterministic and dependency-free (no charting
library, no JavaScript) so the final report is a single static HTML file.

Loaded by 06-make-lazy-report.py via importlib. Each "rows" argument is an
ordered list of (display_name, stat_dict) tuples; stat_dict has keys:
  mean, min, max, median, cv, B, allocs, provider
"""
import math

ACCENT = "#E8470C"
INK = "#0a0a0a"
HAIR = "#e5e5e5"
MUTED = "#6b6b6b"
GREEN = "#2d7d46"
MONO = "ui-monospace,SFMono-Regular,Menlo,Consolas,monospace"


def _esc(s):
    return str(s).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def _svg_open(W, H, aria):
    return (f'<svg viewBox="0 0 {W} {H}" xmlns="http://www.w3.org/2000/svg" '
            f'role="img" aria-label="{_esc(aria)}" '
            f'style="width:100%;height:auto;display:block;max-width:{W}px;margin:8px 0">')


# ---------------------------------------------------------------------------
# Figure 1 — Object.keys cost by mechanism (log-scale horizontal bars),
#           provider/op annotated. The headline: flag-only = 0 provider calls.
# ---------------------------------------------------------------------------
def enum_logbars(rows):
    W, H, L, R, T, B = 780, 300, 150, 270, 26, 34
    pw, ph = W - L - R, H - T - B
    n = len(rows)
    rh = ph / n
    lo, hi = math.log10(20000), math.log10(1_000_000)

    def x(v):
        return L + (math.log10(v) - lo) / (hi - lo) * pw

    p = [_svg_open(W, H, "Object.keys cost by mechanism, log scale")]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    for t in [50000, 100000, 250000, 500000, 1000000]:
        xt = x(t)
        p.append(f'<line x1="{xt:.1f}" y1="{T}" x2="{xt:.1f}" y2="{T+ph}" stroke="{HAIR}"/>')
        lbl = f"{t/1000:.0f}k" if t < 1000000 else f"{t/1e6:.0f}M"
        p.append(f'<text x="{xt:.1f}" y="{H-12}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="middle">{lbl}</text>')
    p.append(f'<text x="{L+pw/2}" y="{H-1}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle">nanoseconds per Object.keys &#160;(log scale)</text>')
    for i, (name, r) in enumerate(rows):
        cy = T + rh * i + rh / 2
        bx = x(r["mean"])
        bw = bx - L
        prov = r["provider"]
        col = ACCENT if prov == 0 else INK
        p.append(f'<rect x="{L}" y="{cy-11:.1f}" width="{bw:.1f}" height="22" fill="{col}"/>')
        p.append(f'<text x="{L-12}" y="{cy+4:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="end">{_esc(name)}</text>')
        provlbl = "0 provider calls" if prov == 0 else f"{int(prov)} provider calls/op"
        p.append(f'<text x="{bx+9:.1f}" y="{cy+4:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}">{r["mean"]/1000:.0f} &#181;s &#160;&#183;&#160; {provlbl}</text>')
    return "\n".join(p) + "</svg>"


# ---------------------------------------------------------------------------
# Figure 2 — provider/op: the deterministic signal across operations.
#           The load-bearing chart: enumeration = 0, serialization = N, cold = 1.
# ---------------------------------------------------------------------------
def provider_signal(rows):
    W, H, L, R, T, B = 780, 300, 220, 40, 26, 40
    pw, ph = W - L - R, H - T - B
    n = len(rows)
    rh = ph / n
    xmax = max(r["provider"] for _, r in rows) * 1.15 if rows else 1

    def x(v):
        return L + v / xmax * pw

    p = [_svg_open(W, H, "Provider calls per operation — the deterministic signal")]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    for gv in [0, 25, 50, 75, 100]:
        if gv > xmax:
            break
        xg = x(gv)
        p.append(f'<line x1="{xg:.1f}" y1="{T}" x2="{xg:.1f}" y2="{T+ph}" stroke="{HAIR}"/>'
                 f'<text x="{xg:.1f}" y="{H-12}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="middle">{gv}</text>')
    p.append(f'<text x="{L+pw/2}" y="{H-1}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle">provider calls per operation &#160;(deterministic)</text>')
    for i, (name, r) in enumerate(rows):
        cy = T + rh * i + rh / 2
        bw = x(r["provider"]) - L
        col = GREEN if r["provider"] == 0 else ACCENT
        p.append(f'<rect x="{L}" y="{cy-11:.1f}" width="{bw:.1f}" height="22" fill="{col}"/>')
        p.append(f'<text x="{L-12}" y="{cy+4:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="end">{_esc(name)}</text>')
        lbl = "0 &#160;(lazy)" if r["provider"] == 0 else f"{int(r['provider'])} &#160;(materialises)"
        p.append(f'<text x="{x(r["provider"])+9:.1f}" y="{cy+4:.1f}" font-family="{MONO}" '
                 f'font-size="12" fill="{INK}">{lbl}</text>')
    p.append(f'<rect x="{L+8}" y="{H-30}" width="13" height="13" fill="{GREEN}"/>'
             f'<text x="{L+28}" y="{H-19}" font-family="{MONO}" font-size="11" '
             f'fill="{INK}">no provider work</text>')
    p.append(f'<rect x="{L+180}" y="{H-30}" width="13" height="13" fill="{ACCENT}"/>'
             f'<text x="{L+200}" y="{H-19}" font-family="{MONO}" font-size="11" '
             f'fill="{INK}">materialises values</text>')
    return "\n".join(p) + "</svg>"


# ---------------------------------------------------------------------------
# Figure 3 — scaling: Object.keys on lazy.Map, N = 10/100/1000 (linear).
#           provider/op = 0 throughout (annotated).
# ---------------------------------------------------------------------------
def scale_bars(rows):
    W, H, L, R, T, B = 720, 320, 70, 40, 26, 50
    pw, ph = W - L - R, H - T - B
    n = len(rows)
    bw = pw / n * 0.5
    xmax = max(r["mean"] for _, r in rows) * 1.1

    def x(v):
        return L + v / xmax * pw

    p = [_svg_open(W, H, "Object.keys on lazy.Map scaling with key count")]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    for i, (name, r) in enumerate(rows):
        cx = L + pw / n * (i + 0.5)
        bh = r["mean"] / xmax * ph
        by = T + ph - bh
        p.append(f'<rect x="{cx-bw/2:.1f}" y="{by:.1f}" width="{bw:.1f}" height="{bh:.1f}" fill="{ACCENT}"/>')
        p.append(f'<text x="{cx:.1f}" y="{by-6:.1f}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="middle">{r["mean"]/1000:.0f}k ns</text>')
        p.append(f'<text x="{cx:.1f}" y="{T+ph+18}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="middle">{_esc(name)}</text>')
        p.append(f'<text x="{cx:.1f}" y="{T+ph+34}" font-family="{MONO}" font-size="10.5" '
                 f'fill="{MUTED}" text-anchor="middle">{r["allocs"]} allocs &#160;&#183;&#160; 0 provider</text>')
    p.append(f'<text x="{L-44}" y="{T+ph/2}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle" '
             f'transform="rotate(-90 {L-44} {T+ph/2})">nanoseconds per Object.keys</text>')
    return "\n".join(p) + "</svg>"


# ---------------------------------------------------------------------------
# Figure 4 — reads: dot+range (min—median—max), mean as a tick, allocs annotated.
# ---------------------------------------------------------------------------
def dotrange(rows, xmin, xmax, title):
    rows = list(rows)
    n = len(rows)
    W, H, L, R, T, B = 720, 62 + 42 * n, 200, 120, 26, 36
    pw, ph = W - L - R, H - T - B
    rh = ph / n

    def sx(v):
        return L + (v - xmin) / (xmax - xmin) * pw

    p = [_svg_open(W, H, title)]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
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
    p.append(f'<text x="{L+pw/2}" y="{H}" font-family="{MONO}" font-size="11" '
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
        prov = "0" if r["provider"] == 0 else f"{int(r['provider'])}"
        p.append(f'<text x="{xmx+12:.1f}" y="{cy+4:.1f}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}">{r["allocs"]} allocs &#160;&#183;&#160; {prov} prov</text>')
    p.append(f'<circle cx="{L+pw-200}" cy="{H-30}" r="5" fill="{ACCENT}"/>'
             f'<text x="{L+pw-188}" y="{H-26}" font-family="{MONO}" font-size="10.5" '
             f'fill="{MUTED}">median</text>')
    p.append(f'<line x1="{L+pw-130}" y1="{H-34}" x2="{L+pw-130}" y2="{H-26}" stroke="{INK}"/>'
             f'<text x="{L+pw-122}" y="{H-26}" font-family="{MONO}" font-size="10.5" '
             f'fill="{MUTED}">mean</text>')
    return "\n".join(p) + "</svg>"


# ---------------------------------------------------------------------------
# Figure 5 — Proxy vs DynamicObject backend (read + enumeration), log scale.
# ---------------------------------------------------------------------------
def backend_compare(read, enum):
    # read / enum are dicts with keys "Proxy" and "DynamicObject", each a stat dict.
    W, H, L, R, T, B = 720, 320, 70, 40, 26, 50
    pw, ph = W - L - R, H - T - B
    groups = [("Single read", read), ("Object.keys (100 keys)", enum)]
    lo, hi = math.log10(100), math.log10(200000)

    def x(v):
        return L + (math.log10(v) - lo) / (hi - lo) * pw

    p = [_svg_open(W, H, "Proxy vs DynamicObject backend, log scale")]
    p.append(f'<rect width="{W}" height="{H}" fill="#fff"/>')
    p.append(f'<rect x="{L}" y="{T}" width="{pw}" height="{ph}" fill="none" stroke="{INK}"/>')
    for t in [100, 1000, 10000, 100000]:
        xt = x(t)
        p.append(f'<line x1="{xt:.1f}" y1="{T}" x2="{xt:.1f}" y2="{T+ph}" stroke="{HAIR}"/>')
        lbl = f"{t/1000:.0f}k" if t < 100000 else f"{t/1e6:.1f}M" if t >= 1000000 else f"{t//1000}k"
        p.append(f'<text x="{xt:.1f}" y="{H-12}" font-family="{MONO}" font-size="11" '
                 f'fill="{MUTED}" text-anchor="middle">{lbl}</text>')
    p.append(f'<text x="{L+pw/2}" y="{H-1}" font-family="{MONO}" font-size="11" '
             f'fill="{MUTED}" text-anchor="middle">nanoseconds (log scale)</text>')
    gw = pw / len(groups)
    bw = gw / 3.2
    for gi, (label, d) in enumerate(groups):
        gx = L + gw * gi + gw / 2
        p.append(f'<text x="{gx:.1f}" y="{T+ph+18}" font-family="{MONO}" font-size="12" '
                 f'fill="{INK}" text-anchor="middle">{_esc(label)}</text>')
        for bi, (name, col) in enumerate([("Proxy", INK), ("DynamicObject", ACCENT)]):
            r = d[name]
            bh = (math.log10(r["mean"]) - lo) / (hi - lo) * ph
            bx = gx - bw + bi * (bw * 1.1)
            by = T + ph - bh
            p.append(f'<rect x="{bx:.1f}" y="{by:.1f}" width="{bw:.1f}" height="{bh:.1f}" fill="{col}"/>')
            p.append(f'<text x="{bx+bw/2:.1f}" y="{by-6:.1f}" font-family="{MONO}" font-size="10.5" '
                     f'fill="{INK}" text-anchor="middle">{r["mean"]:.0f}</text>')
    p.append(f'<rect x="{L+8}" y="{H-30}" width="13" height="13" fill="{INK}"/>'
             f'<text x="{L+28}" y="{H-19}" font-family="{MONO}" font-size="11" '
             f'fill="{INK}">Proxy</text>')
    p.append(f'<rect x="{L+90}" y="{H-30}" width="13" height="13" fill="{ACCENT}"/>'
             f'<text x="{L+110}" y="{H-19}" font-family="{MONO}" font-size="11" '
             f'fill="{INK}">DynamicObject</text>')
    return "\n".join(p) + "</svg>"
