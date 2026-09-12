import glob, json, re, sys

files = sorted(glob.glob("deploy/monitoring/baseline/snap_*.txt"))
if not files:
    sys.exit("no snapshots in deploy/monitoring/baseline/")

def parse(path):
    buckets = {}; counters = []; gauges = {}; uptime = None
    for line in open(path):
        m = re.match(r'^([a-z0-9_]+)(?:\{([^}]*)\})?\s+(-?[0-9.eE+]+)', line.rstrip("\n"))
        if not m:
            continue
        name, labels, val = m.group(1), m.group(2), float(m.group(3))
        lab = dict(re.findall(r'(\w+)="([^"]*)"', labels or ""))
        if name.endswith("_bucket") and "le" in lab:
            buckets.setdefault(name[: -len("_bucket")], []).append((lab, val))
        elif name == "cashflow_http_requests_total":
            counters.append((lab, val))
        elif name in (
            "cashflow_go_memory_alloc_bytes",
            "cashflow_go_goroutines",
            "cashflow_db_pool_active_connections",
            "cashflow_db_pool_max_connections",
            "cashflow_db_pool_wait_count_total",
        ):
            gauges.setdefault(name, []).append(val)
        elif name == "cashflow_process_uptime_seconds":
            uptime = val
    return buckets, counters, gauges, uptime

def percentile(bucket_data, pct):
    agg = {}
    for lab, val in bucket_data:
        agg[lab["le"]] = agg.get(lab["le"], 0.0) + val
    total = agg.get("+Inf", 0.0)
    if total <= 0:
        return None
    target = pct * total
    ordered = sorted((float(le), v) for le, v in agg.items() if le != "+Inf")
    cum, prev_le, prev_cum = 0.0, 0.0, 0.0
    for le, v in ordered:
        cum += v
        if cum >= target:
            if cum == prev_cum:
                return le
            return prev_le + (le - prev_le) * (target - prev_cum) / (cum - prev_cum)
        prev_le, prev_cum = le, cum
    return ordered[-1][0]

first = parse(files[0])
last = parse(files[-1])

total = sum(v for _, v in last[1])
share = {"2xx": 0.0, "3xx": 0.0, "4xx": 0.0, "5xx": 0.0}
for lab, v in last[1]:
    share[lab.get("status_class", "2xx")] = share.get(lab.get("status_class", "2xx"), 0.0) + v
share = {k: round(v / total, 4) if total else 0 for k, v in share.items()}

durs = []
pool_ratios = []
for f in files:
    b, _, g, _ = parse(f)
    serie = b.get("cashflow_http_request_duration_seconds")
    if serie:
        p = percentile(serie, 0.95)
        if p is not None:
            durs.append(p)
    act = g.get("cashflow_db_pool_active_connections", [None])
    mx = g.get("cashflow_db_pool_max_connections", [None])
    if act[-1] is not None and mx[-1]:
        pool_ratios.append(act[-1] / mx[-1])
durs.sort()

first_g = first[2]
last_g = last[2]
waits = last_g.get("cashflow_db_pool_wait_count_total", [0])[-1] - first_g.get("cashflow_db_pool_wait_count_total", [0])[0]
run_secs = ((last[3] or 0) - (first[3] or 0)) or 1

summary = {
    "snapshots": len(files),
    "uptime_seconds": last[3],
    "business_total": total,
    "business_rps_over_run": round(total / (last[3] or 1), 1),
    "share_2xx": share["2xx"],
    "share_4xx": share["4xx"],
    "share_5xx": share["5xx"],
    "business_p95_min_s": round(durs[0], 4) if durs else None,
    "business_p95_median_s": round(durs[len(durs) // 2], 4) if durs else None,
    "business_p95_max_s": round(durs[-1], 4) if durs else None,
    "probe_p95_s": round(percentile(last[0].get("cashflow_probe_duration_seconds", []), 0.95) or 0, 4),
    "heap_delta_bytes": int((last_g.get("cashflow_go_memory_alloc_bytes", [0])[-1]) - (first_g.get("cashflow_go_memory_alloc_bytes", [0])[0] or 0)),
    "goroutines_end": last_g.get("cashflow_go_goroutines", [None])[-1],
    "pool_active_max_ratio": round(max(pool_ratios), 4) if pool_ratios else 0,
    "pool_active_end": last_g.get("cashflow_db_pool_active_connections", [0])[-1],
    "pool_max_conns": last_g.get("cashflow_db_pool_max_connections", [0])[-1],
    "pool_waits_total": int(waits),
    "pool_waits_per_s": round(waits / run_secs, 2),
}
json.dump(summary, open("deploy/monitoring/baseline/summary.json", "w"), indent=2)

print("=" * 60)
print("BASELINE SUMMARY")
print("=" * 60)
for k, v in summary.items():
    print(f"  {k:24s} : {v}")
print("=" * 60)
print("  suggested 5xx-share alert = max(measured_5xx*3, SLO floor 0.005)")
print("  suggested p95 alert       = SLO 500ms - 20% margin = 0.4s")
print("  suggested heap alert      = max(measured_delta*10, 128MiB)")
print("  suggested pool alert      = max(observed_ratio*1.25, 0.8) sustained")
