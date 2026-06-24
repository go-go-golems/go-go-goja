#!/usr/bin/env python3
# 06-make-lazy-report.py — final stage of the lazy-benchmark report pipeline.
#
# Loads the parsed statistics, builds the narrative-ordered datasets, generates
# the four inline-SVG figures, fills the report template, and writes a single
# self-contained HTML report.
#
#   stats  (various/03-stats.json)
#   charts (scripts/04-lazy-charts.py)
#   tmpl   (scripts/05-lazy-report-template.html)
#   -> report (various/06-lazy-report.html)
#
#   python3 scripts/06-make-lazy-report.py
import json, sys, datetime, importlib.util, re
from pathlib import Path

HERE = Path(__file__).resolve().parent
TICKET = HERE.parent
VAR = TICKET / "various"
STATS_PATH = VAR / "03-stats.json"
CHARTS_PATH = HERE / "04-lazy-charts.py"
TEMPLATE_PATH = HERE / "05-lazy-report-template.html"
OUT_PATH = VAR / "06-lazy-report.html"
DYN_STATS_PATH = VAR / "05-dyn-stats.json"


def load_charts(path):
    spec = importlib.util.spec_from_file_location("lazy_charts", path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def grab(stats, key):
    s = stats[key]
    return {"mean": s["mean"], "min": s["min"], "max": s["max"],
            "median": s["median"], "cv": s["cv"], "B": s["B"],
            "allocs": s["allocs"], "provider": s["provider"]}


def main():
    stats = json.loads(STATS_PATH.read_text())
    charts = load_charts(CHARTS_PATH)

    R = "BenchmarkLazyRead/"
    E = "BenchmarkLazyEnumeration/"
    S = "BenchmarkLazyScale/LazyMap_Keys/N="
    M = "BenchmarkLazyMaterialize/"

    e_obj = grab(stats, R + "Eager_Object-8")
    cached = grab(stats, R + "LazyMap_Cached-8")
    cold = grab(stats, R + "LazyMap_Cold_NoCache-8")
    e_keys = grab(stats, E + "Eager_Keys-8")
    flag = grab(stats, E + "LazyMap_Keys_FlagOnly-8")
    naive = grab(stats, E + "NaiveLazy_Keys_Materializing-8")
    forin = grab(stats, E + "LazyMap_ForIn-8")
    n10 = grab(stats, S + "10-8")
    n100 = grab(stats, S + "100-8")
    n1000 = grab(stats, S + "1000-8")
    str_e = grab(stats, M + "Eager_Stringify-8")
    str_l = grab(stats, M + "LazyMap_Stringify-8")

    enum_rows = [("Eager object", e_keys), ("Lazy map (flag-only)", flag),
                 ("Naive lazy (materialising)", naive), ("for...in (flag-only)", forin)]
    signal_rows = [("Object.keys (flag-only)", flag), ("for...in (flag-only)", forin),
                   ("cached read", cached), ("cold read", cold), ("JSON.stringify", str_l)]
    scale_rows = [("N=10", n10), ("N=100", n100), ("N=1000", n1000)]
    read_rows = [("Eager object", e_obj), ("Lazy map (cached)", cached),
                 ("Lazy map (cold, no cache)", cold)]

    tokens = {
        "date": datetime.date.today().isoformat(),
        # reads
        "read_eager_ns": round(e_obj["mean"]),
        "read_cached_ns": round(cached["mean"]), "read_cached_allocs": cached["allocs"],
        "read_cold_ns": round(cold["mean"]), "read_cold_allocs": cold["allocs"],
        "read_ratio": round(cached["mean"] / e_obj["mean"], 1),
        # enumeration
        "enum_eager_ns": round(e_keys["mean"]), "enum_eager_allocs": e_keys["allocs"],
        "enum_flagonly_ns": round(flag["mean"]), "enum_flagonly_allocs": flag["allocs"],
        "enum_naive_ns": round(naive["mean"]), "enum_naive_allocs": naive["allocs"],
        "enum_forin_ns": round(forin["mean"]), "enum_forin_allocs": forin["allocs"],
        "enum_ratio": round(flag["mean"] / e_keys["mean"], 1),
        # scaling
        "scale_n1000_ns": round(n1000["mean"]),
        # serialisation
        "str_eager_ns": round(str_e["mean"]), "str_eager_allocs": str_e["allocs"],
        "str_lazy_ns": round(str_l["mean"]), "str_lazy_allocs": str_l["allocs"],
        "str_lazy_provider": int(str_l["provider"]),
        "str_ratio": round(str_l["mean"] / str_e["mean"], 1),
        # figures
        "chartA": charts.enum_logbars(enum_rows),
        "chartB": charts.provider_signal(signal_rows),
        "chartC": charts.scale_bars(scale_rows),
        "chartD": charts.dotrange(read_rows, 0, 1400, "Single-read latency by mechanism"),
    }

    # DynamicObject backend comparison (phase5)
    dyn = json.loads(DYN_STATS_PATH.read_text()) if DYN_STATS_PATH.exists() else {}
    if dyn:
        dg = lambda k: {"mean": dyn[k]["mean"], "min": dyn[k]["min"], "max": dyn[k]["max"],
                        "median": dyn[k]["median"], "allocs": dyn[k]["allocs"],
                        "provider": dyn[k].get("provider", 0.0)}
        backend_read = {"Proxy": dg("BenchmarkLazyBackendRead/Proxy"),
                        "DynamicObject": dg("BenchmarkLazyBackendRead/DynamicObject")}
        backend_enum = {"Proxy": dg("BenchmarkLazyBackendEnum/Proxy"),
                        "DynamicObject": dg("BenchmarkLazyBackendEnum/DynamicObject")}
        tokens["chartE"] = charts.backend_compare(backend_read, backend_enum)
    else:
        tokens["chartE"] = ""

    tmpl = TEMPLATE_PATH.read_text()
    referenced = set(re.findall(r"@@([a-zA-Z0-9_]+)@@", tmpl))
    missing = referenced - set(tokens)
    if missing:
        sys.exit(f"ERROR: template references undefined tokens: {sorted(missing)}")

    def repl(m):
        v = tokens[m.group(1)]
        return str(v) if not isinstance(v, float) else f"{v:.1f}"

    OUT_PATH.write_text(re.sub(r"@@([a-zA-Z0-9_]+)@@", repl, tmpl))
    print("report written:", OUT_PATH.relative_to(TICKET.parent.parent.parent))
    print(f"tokens used ({len(tokens)}):")
    for k in sorted(tokens):
        v = tokens[k]
        print(f"  {k:<22} = {('<svg: %d chars>' % len(str(v))) if k.startswith('chart') else v}")


if __name__ == "__main__":
    sys.exit(main())
