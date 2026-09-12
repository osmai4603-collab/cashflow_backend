package metrics

import (
	"io"
	"net/http"
)

// IndexPage renders the management server home page. It is served on the
// dedicated 8066 listener (P4) and mirrors the role of Mattermost's :8067
// index: links to the metrics endpoints and pprof, plus a live dashboard that
// polls /metrics/json. No external assets are used so it works fully offline.
func IndexPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, indexHTML)
	})
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>cashflow — observability</title>
<style>
  :root { --bg:#0f172a; --panel:#1e293b; --accent:#34d399; --text:#e2e8f0; --muted:#94a3b8; --danger:#f87171; --warn:#fbbf24; }
  * { box-sizing:border-box; margin:0; }
  body { background:var(--bg); color:var(--text); font:14px/1.5 system-ui, "Segoe UI", sans-serif; padding:32px; }
  h1 { font-size:22px; letter-spacing:.5px; }
  h1 span { color:var(--accent); }
  .sub { color:var(--muted); margin:4px 0 24px; }
  .links { display:flex; gap:12px; flex-wrap:wrap; margin-bottom:24px; }
  .links a { background:var(--panel); color:var(--accent); border:1px solid #334155; border-radius:8px; padding:8px 14px; text-decoration:none; }
  .links a:hover { border-color:var(--accent); }
  .grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(180px,1fr)); gap:12px; }
  .card { background:var(--panel); border:1px solid #334155; border-radius:12px; padding:16px; }
  .card .label { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:.6px; }
  .card .value { font-size:24px; font-weight:600; margin-top:6px; font-variant-numeric:tabular-nums; }
  .ok   { color:var(--accent); } .warn { color:var(--warn); } .bad { color:var(--danger); }
  footer { margin-top:28px; color:var(--muted); font-size:12px; }
  pre { display:none; }
</style>
</head>
<body>
  <h1>cashflow <span>observability</span></h1>
  <p class="sub" id="server">management endpoint</p>
  <div class="links">
    <a href="/metrics">Prometheus text (.0.0.4)</a>
    <a href="/metrics/json">JSON (compat)</a>
    <a href="/livez" target="_blank">livez</a>
    <a href="/readyz" target="_blank">readyz</a>
    <a href="/debug/pprof/" target="_blank">pprof</a>
  </div>
  <div class="grid">
    <div class="card"><div class="label">Uptime</div><div class="value" id="uptime">–</div></div>
    <div class="card"><div class="label">Goroutines</div><div class="value" id="goroutines">–</div></div>
    <div class="card"><div class="label">Total requests</div><div class="value" id="requests">–</div></div>
    <div class="card"><div class="label">4xx errors</div><div class="value warn" id="err4xx">–</div></div>
    <div class="card"><div class="label">5xx errors</div><div class="value bad" id="err5xx">–</div></div>
    <div class="card"><div class="label">Avg latency (ms)</div><div class="value" id="latency">–</div></div>
    <div class="card"><div class="label">DB pool active</div><div class="value" id="dbActive">–</div></div>
    <div class="card"><div class="label">DB pool wait count</div><div class="value" id="dbWait">–</div></div>
  </div>
  <footer>Auto-refreshes every 2s from <code>/metrics/json</code>.</footer>
<script>
(function () {
  var el = function (id) { return document.getElementById(id); };
  function fmt(n, digits) { return n == null ? "–" : Number(n).toFixed(digits == null ? 0 : digits); }
  function refresh() {
    fetch("/metrics/json", { cache: "no-store" }).then(function (r) { return r.json(); }).then(function (m) {
      el("server").textContent = m.server_url || "management endpoint";
      el("uptime").textContent = fmt(m.uptime_seconds, 1) + "s";
      el("goroutines").textContent = fmt(m.num_goroutines);
      el("requests").textContent = fmt(m.http_requests_total);
      el("err4xx").textContent = fmt(m.http_4xx_total);
      el("err5xx").textContent = fmt(m.http_5xx_total);
      el("latency").textContent = fmt(m.avg_latency_ms, 2);
      if (m.db) { el("dbActive").textContent = fmt(m.db.active_conns); el("dbWait").textContent = fmt(m.db.wait_count); }
    }).catch(function () {
      el("server").textContent = "management endpoint unreachable";
    });
  }
  refresh();
  setInterval(refresh, 2000);
})();
</script>
</body>
</html>
`
