#!/usr/bin/env bash
#
# Calibrate alert thresholds from a measured baseline (Observability launch
# step "staging baseline" run locally with the in-memory driver).
#
# What it does:
#   1. Starts the server (in-memory driver) on its management port.
#   2. Drives a bounded load profile with tool/loadtest against probe and
#      business-reject paths (far from the request-serving critical path).
#   3. Samples /metrics (Prometheus text) every 500ms into deploy/monitoring/baseline/.
#   4. Analyses histogram buckets + counters + heap gauge into percentiles,
#      error shares and growth deltas, prints them and writes baseline/summary.json.
#
# Usage:
#   tool/calibrate_baseline.sh                                             # memory driver
#   STORAGE=postgres tool/calibrate_baseline.sh                            # real postgres (DB_* env)
#   STORAGE=postgres BUSINESS_URL=/api/v1/companies tool/calibrate_baseline.sh
#   PORT=9797 MANAGEMENT_PORT=9798 tool/calibrate_baseline.sh              # custom ports
#
# Notes:
#   - STORAGE=memory: DB-pool gauges are always 0, so DBPool* thresholds stay
#     PENDING. STORAGE=postgres: the run also measures the pool (active/max
#     ratio over time + waits rate) and yields a real 5xx figure when business
#     paths hit the DB.
#   - BUSINESS_URL (optional): path driven in round A (default
#     /api/v1/companies). For a staged run point it at a real business route
#     UNAUTHENTICATED (e.g. a public listing) or an auth-protected one to
#     measure the 401 reject path.
#   - DB_* envs default to localhost/postgres/postgres/cashflow for a locally
#     started postgres (docker compose up -d postgres).
set -euo pipefail

PORT="${PORT:-16666}"
MGMT_PORT="${MGMT_PORT:-16667}"
SAMPLE_INTERVAL_S=0.5
STORAGE="${STORAGE:-memory}"
BUSINESS_URL="${BUSINESS_URL:-/api/v1/companies}"
WORK="$(mktemp -d /tmp/cashflow-calibrate.XXXXXX)"
BASELINE_DIR="deploy/monitoring/baseline"
APP_PID=""
APP_ADDR="http://127.0.0.1:${MGMT_PORT}"

cleanup() {
  if [[ -n "${APP_PID}" ]] && kill -0 "${APP_PID}" 2>/dev/null; then
    kill "${APP_PID}" 2>/dev/null || true
    wait "${APP_PID}" 2>/dev/null || true
  fi
}
trap 'cleanup; rm -rf "${WORK}"' EXIT

mkdir -p "${BASELINE_DIR}"

echo "==> building server and loadtest"
go build -o "${WORK}/cashflow-server" ./cmd/server
go build -o "${WORK}/loadtest" ./tool/loadtest

echo "==> starting server (public :${PORT}, management :${MGMT_PORT}, storage=${STORAGE})"
DBENV=()
if [[ "${STORAGE}" == "postgres" ]]; then
  DBENV+=(
    DB_HOST="${DB_HOST:-127.0.0.1}"
    DB_PORT="${DB_PORT:-5432}"
    DB_USER="${DB_USER:-postgres}"
    DB_PASSWORD="${DB_PASSWORD:-postgres}"
    DB_NAME="${DB_NAME:-cashflow}"
    DB_SSLMODE="${DB_SSLMODE:-disable}"
  )
fi
env $(
  for kv in \
    PORT="${PORT}" HTTP_INTERFACE=127.0.0.1 STORAGE_DRIVER="${STORAGE}" \
    MANAGEMENT_PORT="${MGMT_PORT}" MANAGEMENT_INTERFACE=127.0.0.1 \
    DRAIN_SECONDS=0 SHUTDOWN_SECONDS=5; do printf '%s ' "$kv"; done
  for kv in "${DBENV[@]}"; do printf '%s ' "$kv"; done
) "${WORK}/cashflow-server" >"${WORK}/server.log" 2>&1 &
APP_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS -o /dev/null "${APP_ADDR}/metrics" 2>/dev/null; then
    break
  fi
  sleep 0.25
done
if ! curl -fsS -o /dev/null "${APP_ADDR}/metrics"; then
  echo "ERROR: management endpoint never came up" >&2
  cat "${WORK}/server.log" >&2
  exit 1
fi

echo "==> sampling /metrics every ${SAMPLE_INTERVAL_S}s while driving load"
(
  i=0
  while : ; do
    i=$((i + 1))
    curl -fsS "${APP_ADDR}/metrics" >"${BASELINE_DIR}/snap_$(printf "%05d" "$i").txt" 2>/dev/null || true
    sleep "${SAMPLE_INTERVAL_S}"
  done
) &
SAMPLER_PID=$!

finish() {
  kill "${SAMPLER_PID}" 2>/dev/null || true
  wait "${SAMPLER_PID}" 2>/dev/null || true
  cleanup
  rm -rf "${WORK}"
}
trap 'finish' EXIT

echo "==> round A: business path (${BUSINESS_URL}) — c=32, n=30000"
"${WORK}/loadtest" -url "http://127.0.0.1:${PORT}${BUSINESS_URL}" -c 32 -n 30000 | tail -6

echo "==> round B: readiness probe — c=64, n=30000"
"${WORK}/loadtest" -url "http://127.0.0.1:${PORT}/readyz" -c 64 -n 30000 | tail -6

echo "==> round C: liveness probe — c=16, n=5000"
"${WORK}/loadtest" -url "http://127.0.0.1:${PORT}/livez" -c 16 -n 5000 | tail -6

sleep 1
SNAPS=$(ls "${BASELINE_DIR}"/snap_*.txt 2>/dev/null | wc -l)
echo "==> collected ${SNAPS} snapshots"

cat > "${BASELINE_DIR}/analyze.py" <<'PYEOF'
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
PYEOF

python3 "${BASELINE_DIR}/analyze.py"

echo "==> artifacts: ${BASELINE_DIR}/{snap_*.txt, summary.json, analyze.py}"