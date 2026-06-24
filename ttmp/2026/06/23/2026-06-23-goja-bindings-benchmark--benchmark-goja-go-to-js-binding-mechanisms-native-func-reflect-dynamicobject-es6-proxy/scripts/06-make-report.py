#!/usr/bin/env python3
# 06-make-report.py
#
# Stage 3 (final) of the report pipeline. Loads the parsed statistics, builds the
# narrative-ordered datasets, generates the four inline-SVG figures, fills the
# report template, and writes a single self-contained HTML report.
#
#   stats  (various/03-stats.json)
#   charts (scripts/04-goja-charts.py)
#   tmpl   (scripts/05-goja-report-template.html)
#   -> report (various/06-bindings-report.html)
#
# Every @@token@@ in the template is a flat name computed here (no arithmetic is
# parsed inside the template), so the mapping is explicit and auditable.
#
#   python3 scripts/06-make-report.py
import json
import sys
import datetime
import importlib.util
import statistics
from pathlib import Path

HERE = Path(__file__).resolve().parent
TICKET = HERE.parent
VAR = TICKET / "various"

STATS_PATH = VAR / "03-stats.json"
CHARTS_PATH = HERE / "04-goja-charts.py"
TEMPLATE_PATH = HERE / "05-goja-report-template.html"
OUT_PATH = VAR / "06-bindings-report.html"


def load_charts(path):
    """Import 04-goja-charts.py despite its numeric prefix."""
    spec = importlib.util.spec_from_file_location("goja_charts", path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def main():
    stats = json.loads(STATS_PATH.read_text())
    charts = load_charts(CHARTS_PATH)

    # ---- curated, narrative-ordered datasets (cheapest -> priciest) ----
    # maps a friendly display name -> the benchmark key in stats.json
    call_order = [
        ("Native func",    "BenchmarkBindingFunctionCall/NativeFunc-8"),
        ("Proxy (Apply)",  "BenchmarkBindingFunctionCall/ProxyApply-8"),
        ("Wrapped func",   "BenchmarkBindingFunctionCall/WrappedFunc-8"),
        ("SetExport",      "BenchmarkBindingFunctionCall/SetExport-8"),
        ("Map of funcs",   "BenchmarkBindingFunctionCall/MapOfFuncs-8"),
        ("Reflect method", "BenchmarkBindingFunctionCall/ReflectStructMethod-8"),
    ]
    read_order = [
        ("Plain object (reused)", "BenchmarkBindingPropertyGet/PlainObject_Reuse-8"),
        ("DynamicObject",         "BenchmarkBindingPropertyGet/DynamicObject-8"),
        ("Go map",                "BenchmarkBindingPropertyGet/GoMapSimple-8"),
        ("DynamicArray",          "BenchmarkBindingArrayIndexGet/DynamicArray-8"),
        ("Reflect field",         "BenchmarkBindingPropertyGet/ReflectStructField-8"),
        ("Proxy (Get)",           "BenchmarkBindingPropertyGet/Proxy_GetTrap-8"),
    ]
    proxy_order = [
        ("switch \u2014 hit",    "BenchmarkProxyDispatchDetail/GetTrap_Switch_Hit-8"),
        ("map \u2014 hit",       "BenchmarkProxyDispatchDetail/GetTrap_Map_Hit-8"),
        ("miss \u2192 undefined", "BenchmarkProxyDispatchDetail/GetTrap_Miss_Undefined-8"),
    ]

    def grab(key):
        # keep only the fields the charts + prose need
        s = stats[key]
        return {"mean": s["mean"], "min": s["min"], "max": s["max"],
                "median": s["median"], "cv": s["cv"], "B": s["B"], "allocs": s["allocs"]}

    by = {name: grab(key) for name, key in call_order + read_order + proxy_order}

    calls = [(n, by[n]) for n, _ in call_order]
    reads = [(n, by[n]) for n, _ in read_order]
    proxy = [(n, by[n]) for n, _ in proxy_order]

    # ---- derived numbers for the prose ----
    nat = by["Native func"]
    wrap = by["Wrapped func"]
    reflm = by["Reflect method"]
    mapf = by["Map of funcs"]
    pxyapply = by["Proxy (Apply)"]
    pxyget = by["Proxy (Get)"]
    dynobj = by["DynamicObject"]
    switch = by["switch \u2014 hit"]
    mapd = by["map \u2014 hit"]
    miss = by["miss \u2192 undefined"]

    # 1-alloc read floor = mean of the three indistinguishable 1-alloc readers
    floor = statistics.mean([by[n]["mean"] for n in
                             ("Plain object (reused)", "DynamicObject", "Go map")])

    # ---- flat token map (no arithmetic parsed in the template) ----
    tokens = {
        "date": datetime.date.today().isoformat(),
        # native (call floor)
        "nat_ns": round(nat["mean"]), "nat_alloc": nat["allocs"], "nat_cv": round(nat["cv"], 1),
        # wrapped func
        "wrap_ns": round(wrap["mean"]), "wrap_alloc": wrap["allocs"], "wrap_cv": round(wrap["cv"], 1),
        "wrap_alloc_delta": wrap["allocs"] - nat["allocs"],
        "wrap_ns_delta": round(wrap["mean"] - nat["mean"]),
        # map of funcs / reflect method
        "mapf_ns": round(mapf["mean"]), "mapf_alloc": mapf["allocs"],
        "reflm_ns": round(reflm["mean"]), "reflm_alloc": reflm["allocs"],
        # proxy apply (call-through)
        "pxyapply_ns": round(pxyapply["mean"]), "pxyapply_alloc": pxyapply["allocs"],
        "pxyapply_ns_delta": round(pxyapply["mean"] - nat["mean"]),
        # reads
        "floor_ns": round(floor),
        "dynobj_ns": round(dynobj["mean"]),
        "pxyget_ns": round(pxyget["mean"]), "pxyget_cv": round(pxyget["cv"], 1),
        "pxyget_ns_floor_delta": round(pxyget["mean"] - floor),
        # proxy dispatch detail
        "switch_ns": round(switch["mean"]), "switch_cv": round(switch["cv"], 1),
        "mapd_ns": round(mapd["mean"]), "mapd_cv": round(mapd["cv"], 1),
        "miss_ns": round(miss["mean"]), "miss_cv": round(miss["cv"], 1),
    }

    # ---- generate the four figures from real data ----
    tokens["chartA"] = charts.calls_logbars(calls)
    tokens["chartB"] = charts.scatter_alloc_time(calls, reads)
    tokens["chartC"] = charts.dotrange(reads, 100, 240,
                                       "Property-read latency by mechanism")
    tokens["chartD"] = charts.dotrange(proxy, 140, 300,
                                       "Proxy Get-trap latency by dispatch body")

    # ---- fill the template (every @@token@@ must be defined) ----
    tmpl = TEMPLATE_PATH.read_text()
    import re
    referenced = set(re.findall(r"@@([a-zA-Z0-9_]+)@@", tmpl))
    missing = referenced - set(tokens)
    if missing:
        sys.exit(f"ERROR: template references undefined tokens: {sorted(missing)}")

    def repl(m):
        v = tokens[m.group(1)]
        return str(v) if not isinstance(v, float) else f"{v:.1f}"

    out = re.sub(r"@@([a-zA-Z0-9_]+)@@", repl, tmpl)
    OUT_PATH.write_text(out)

    # ---- provenance echo ----
    print("report written:", OUT_PATH.relative_to(TICKET.parent.parent.parent))
    print(f"\ntokens used ({len(tokens)}):")
    for k in sorted(tokens):
        v = tokens[k]
        if k.startswith("chart"):
            v = f"<svg: {len(str(v))} chars>"
        print(f"  {k:<24} = {v}")


if __name__ == "__main__":
    sys.exit(main())
