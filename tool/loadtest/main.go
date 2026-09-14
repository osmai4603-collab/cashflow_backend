package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	BaseURL     string
	MgmtURL     string
	Concurrency int
	TotalReqs   int
	Duration    time.Duration
	Mode        string
	AuthToken   string
	Login       string
	Password    string
}

type Stats struct {
	TotalReqs    int64
	Success2xx   int64
	Success3xx   int64
	ClientErr4xx int64
	ServerErr5xx int64
	NetworkErrs  int64
	TotalBytes   int64
	latencies    []time.Duration
}

func (s *Stats) record(status int, bytesCount int64, err error) {
	atomic.AddInt64(&s.TotalReqs, 1)
	atomic.AddInt64(&s.TotalBytes, bytesCount)

	if err != nil {
		atomic.AddInt64(&s.NetworkErrs, 1)
	} else {
		switch {
		case status >= 200 && status < 300:
			atomic.AddInt64(&s.Success2xx, 1)
		case status >= 300 && status < 400:
			atomic.AddInt64(&s.Success3xx, 1)
		case status >= 400 && status < 500:
			atomic.AddInt64(&s.ClientErr4xx, 1)
		case status >= 500:
			atomic.AddInt64(&s.ServerErr5xx, 1)
		}
	}
}

func loginAdmin(baseURL, login, password string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	payload, _ := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})

	resp, err := client.Post(baseURL+"/api/v1/users/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}
	return res.Data.Token, nil
}

func sendRUMSample(mgmtURL string, client *http.Client) {
	types := []string{"web", "mobile", "desktop"}
	clientType := types[rand.Intn(len(types))]

	lcp := 600.0 + rand.Float64()*1800.0 // 600 - 2400 ms
	inp := 20.0 + rand.Float64()*100.0   // 20 - 120 ms
	cls := rand.Float64() * 0.08         // 0 - 0.08
	ttfb := 40.0 + rand.Float64()*150.0  // 40 - 190 ms

	payload, _ := json.Marshal(map[string]any{
		"client_type": clientType,
		"lcp_ms":      lcp,
		"inp_ms":      inp,
		"cls":         cls,
		"ttfb_ms":     ttfb,
	})

	req, _ := http.NewRequest(http.MethodPost, mgmtURL+"/rum", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.BaseURL, "url", "http://localhost:8070", "Target Base URL")
	flag.StringVar(&cfg.MgmtURL, "mgmt-url", "http://127.0.0.1:8066", "Management URL")
	flag.IntVar(&cfg.Concurrency, "c", 25, "Concurrency (worker goroutines)")
	flag.IntVar(&cfg.TotalReqs, "n", 3000, "Total number of requests (0 for duration-based)")
	flag.DurationVar(&cfg.Duration, "d", 0, "Test duration (e.g. 15s, 30s)")
	flag.StringVar(&cfg.Mode, "mode", "erp-suite", "Mode: erp-suite, stress, rum, mixed, single")
	flag.StringVar(&cfg.AuthToken, "token", "", "JWT Bearer Token (optional, auto-logs in if empty)")
	flag.StringVar(&cfg.Login, "login", "admin", "Admin login name")
	flag.StringVar(&cfg.Password, "password", "admin123", "Admin password")
	flag.Parse()

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	cfg.MgmtURL = strings.TrimRight(cfg.MgmtURL, "/")

	fmt.Printf("=================================================================\n")
	fmt.Printf("   CASHFLOW HIGH-INTENSITY LOAD TESTER & MONITOR VERIFIER\n")
	fmt.Printf("=================================================================\n")
	fmt.Printf("  Target URL:       %s\n", cfg.BaseURL)
	fmt.Printf("  Management URL:   %s\n", cfg.MgmtURL)
	fmt.Printf("  Mode:             %s\n", cfg.Mode)
	fmt.Printf("  Concurrency:      %d workers\n", cfg.Concurrency)
	if cfg.Duration > 0 {
		fmt.Printf("  Duration:         %v\n", cfg.Duration)
	} else {
		fmt.Printf("  Total Requests:   %d\n", cfg.TotalReqs)
	}

	// 1. Obtain Auth Token if needed
	if cfg.AuthToken == "" && (cfg.Mode == "erp-suite" || cfg.Mode == "mixed") {
		fmt.Printf("--> Authenticating against %s/api/v1/users/login...\n", cfg.BaseURL)
		token, err := loginAdmin(cfg.BaseURL, cfg.Login, cfg.Password)
		if err != nil {
			fmt.Printf("    [WARN] Admin login failed: %v. Proceeding without auth.\n", err)
		} else {
			cfg.AuthToken = token
			fmt.Printf("    [OK] JWT Bearer token acquired successfully!\n")
		}
	}

	// 2. Define Workload Endpoints
	type endpoint struct {
		path   string
		auth   bool
		method string
		weight int
	}

	var endpoints []endpoint
	switch cfg.Mode {
	case "stress":
		endpoints = []endpoint{
			{path: "/", auth: false, method: "GET", weight: 60},
			{path: "/livez", auth: false, method: "GET", weight: 40},
		}
	case "rum":
		// Only RUM telemetry generator
		runRUMGenerator(cfg)
		return
	case "single":
		endpoints = []endpoint{
			{path: "/", auth: false, method: "GET", weight: 100},
		}
	case "mixed":
		endpoints = []endpoint{
			{path: "/", auth: false, method: "GET", weight: 25},
			{path: "/livez", auth: false, method: "GET", weight: 15},
			{path: "/readyz", auth: false, method: "GET", weight: 15},
			{path: "/api/v1/partners", auth: true, method: "GET", weight: 15},
			{path: "/api/v1/products", auth: true, method: "GET", weight: 10},
			{path: "/api/v1/currencies", auth: true, method: "GET", weight: 10},
			{path: "/api/v1/unknown_resource", auth: false, method: "GET", weight: 5}, // triggers 404
			{path: "/api/v1/partners", auth: false, method: "GET", weight: 5},         // triggers 401 unauth
		}
	default: // "erp-suite"
		endpoints = []endpoint{
			{path: "/", auth: false, method: "GET", weight: 20},
			{path: "/readyz", auth: false, method: "GET", weight: 10},
			{path: "/api/v1/partners", auth: true, method: "GET", weight: 20},
			{path: "/api/v1/products", auth: true, method: "GET", weight: 15},
			{path: "/api/v1/currencies", auth: true, method: "GET", weight: 15},
			{path: "/api/v1/sequences", auth: true, method: "GET", weight: 10},
			{path: "/api/v1/projects", auth: true, method: "GET", weight: 5},
			{path: "/api/v1/payments", auth: true, method: "GET", weight: 5},
		}
	}

	// Expand weighted distribution table
	var weightedRoutes []endpoint
	for _, ep := range endpoints {
		for i := 0; i < ep.weight; i++ {
			weightedRoutes = append(weightedRoutes, ep)
		}
	}

	// Background RUM simulator during ERP / Mixed test
	stopRUM := make(chan struct{})
	if cfg.Mode == "erp-suite" || cfg.Mode == "mixed" {
		go func() {
			rumClient := &http.Client{Timeout: 3 * time.Second}
			ticker := time.NewTicker(300 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stopRUM:
					return
				case <-ticker.C:
					sendRUMSample(cfg.MgmtURL, rumClient)
				}
			}
		}()
	}

	// Capture baseline telemetry from server to isolate test-run deltas
	var baseReqs, base4xx, base5xx float64
	if initResp, err := (&http.Client{Timeout: 2 * time.Second}).Get(cfg.MgmtURL + "/metrics/json"); err == nil {
		var initMap map[string]any
		if json.NewDecoder(initResp.Body).Decode(&initMap) == nil {
			baseReqs = getFloat(initMap, "http_requests_total")
			base4xx = getFloat(initMap, "http_4xx_total")
			base5xx = getFloat(initMap, "http_5xx_total")
		}
		initResp.Body.Close()
	}

	stats := &Stats{latencies: make([]time.Duration, 0, 50000)}
	allWorkerLatencies := make([][]time.Duration, cfg.Concurrency)

	// Setup workers
	var wg sync.WaitGroup
	reqChan := make(chan endpoint, cfg.Concurrency*4)
	stopChan := make(chan struct{})

	// Shared HTTP transport with high connection reuse
	tr := &http.Transport{
		MaxIdleConns:        cfg.Concurrency * 2,
		MaxIdleConnsPerHost: cfg.Concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	// Worker routines
	for i := 0; i < cfg.Concurrency; i++ {
		workerID := i
		allWorkerLatencies[workerID] = make([]time.Duration, 0, 10000)
		wg.Add(1)
		go func() {
			defer wg.Done()
			wLats := allWorkerLatencies[workerID]
			for target := range reqChan {
				reqURL := cfg.BaseURL + target.path
				req, err := http.NewRequest(target.method, reqURL, nil)
				if err != nil {
					stats.record(0, 0, err)
					continue
				}
				if target.auth && cfg.AuthToken != "" {
					req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
				}

				t0 := time.Now()
				resp, err := client.Do(req)
				dur := time.Since(t0)

				if err != nil {
					stats.record(0, 0, err)
					continue
				}

				n, _ := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				stats.record(resp.StatusCode, n, nil)
				wLats = append(wLats, dur)
			}
			allWorkerLatencies[workerID] = wLats
		}()
	}

	startTime := time.Now()

	// Feed generator
	go func() {
		if cfg.Duration > 0 {
			timer := time.NewTimer(cfg.Duration)
			defer timer.Stop()
			for {
				select {
				case <-timer.C:
					close(reqChan)
					return
				default:
					ep := weightedRoutes[rand.Intn(len(weightedRoutes))]
					select {
					case reqChan <- ep:
					case <-timer.C:
						close(reqChan)
						return
					}
				}
			}
		} else {
			for i := 0; i < cfg.TotalReqs; i++ {
				ep := weightedRoutes[rand.Intn(len(weightedRoutes))]
				reqChan <- ep
			}
			close(reqChan)
		}
	}()

	// Real-time live status reporter
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		var lastCount int64
		lastTime := time.Now()

		for {
			select {
			case <-stopChan:
				return
			case now := <-ticker.C:
				current := atomic.LoadInt64(&stats.TotalReqs)
				dt := now.Sub(lastTime).Seconds()
				rps := float64(current-lastCount) / dt
				lastCount = current
				lastTime = now

				s2 := atomic.LoadInt64(&stats.Success2xx)
				s4 := atomic.LoadInt64(&stats.ClientErr4xx)
				s5 := atomic.LoadInt64(&stats.ServerErr5xx)

				elapsed := int(now.Sub(startTime).Seconds())
				elapsedMin := elapsed / 60
				elapsedSec := elapsed % 60

				fmt.Printf("   [Elapsed: %02dm%02ds] Reqs: %-7d | Rate: %6.1f rps | 2xx: %-7d | 4xx: %-4d | 5xx: %-4d\n",
					elapsedMin, elapsedSec, current, rps, s2, s4, s5)
			}
		}
	}()

	wg.Wait()
	close(stopChan)
	close(stopRUM)

	for _, wl := range allWorkerLatencies {
		stats.latencies = append(stats.latencies, wl...)
	}
	totalDuration := time.Since(startTime)

	// Summary calculations
	fmt.Printf("\n=================================================================\n")
	fmt.Printf("   TEST COMPLETED IN %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("=================================================================\n")

	sort.Slice(stats.latencies, func(i, j int) bool {
		return stats.latencies[i] < stats.latencies[j]
	})
	n := len(stats.latencies)
	var avgLat time.Duration
	var minLat, maxLat, p50, p90, p95, p99 time.Duration

	if n > 0 {
		minLat = stats.latencies[0]
		maxLat = stats.latencies[n-1]
		var totalDur time.Duration
		for _, d := range stats.latencies {
			totalDur += d
		}
		avgLat = totalDur / time.Duration(n)
		p50 = stats.latencies[int(float64(n)*0.50)]
		p90 = stats.latencies[int(float64(n)*0.90)]
		p95 = stats.latencies[int(float64(n)*0.95)]
		p99 = stats.latencies[int(float64(n)*0.99)]
	}

	rps := float64(stats.TotalReqs) / totalDuration.Seconds()
	mbTransferred := float64(stats.TotalBytes) / 1048576.0

	fmt.Printf("  Throughput:       %.2f requests/sec (%.2f MB total)\n", rps, mbTransferred)
	fmt.Printf("  Total Requests:   %d\n", stats.TotalReqs)
	fmt.Printf("  Status Breakdown: 2xx: %d (%.1f%%) | 3xx: %d | 4xx: %d (%.1f%%) | 5xx: %d (%.1f%%)\n",
		stats.Success2xx, float64(stats.Success2xx)*100/float64(max(stats.TotalReqs, 1)),
		stats.Success3xx,
		stats.ClientErr4xx, float64(stats.ClientErr4xx)*100/float64(max(stats.TotalReqs, 1)),
		stats.ServerErr5xx, float64(stats.ServerErr5xx)*100/float64(max(stats.TotalReqs, 1)),
	)
	if stats.NetworkErrs > 0 {
		fmt.Printf("  Network Errors:   %d\n", stats.NetworkErrs)
	}

	fmt.Printf("\n  [ LATENCY DISTRIBUTION ]\n")
	fmt.Printf("  Min:    %v\n", minLat.Round(time.Microsecond))
	fmt.Printf("  Avg:    %v\n", avgLat.Round(time.Microsecond))
	fmt.Printf("  p50:    %v\n", p50.Round(time.Microsecond))
	fmt.Printf("  p90:    %v\n", p90.Round(time.Microsecond))
	fmt.Printf("  p95:    %v\n", p95.Round(time.Microsecond))
	fmt.Printf("  p99:    %v\n", p99.Round(time.Microsecond))
	fmt.Printf("  Max:    %v\n", maxLat.Round(time.Microsecond))

	// Cross-verify with Server Monitor endpoint
	fmt.Printf("\n=================================================================\n")
	fmt.Printf("   CROSS-VERIFICATION WITH MANAGEMENT MONITOR (:8066)\n")
	fmt.Printf("=================================================================\n")
	verifyWithMonitor(cfg.MgmtURL, baseReqs, base4xx, base5xx)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func verifyWithMonitor(mgmtURL string, baseReqs, base4xx, base5xx float64) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(mgmtURL + "/metrics/json")
	if err != nil {
		fmt.Printf("  [WARN] Could not fetch server monitor metrics: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		fmt.Printf("  [WARN] Failed to parse metrics JSON: %v\n", err)
		return
	}

	curReqs := getFloat(m, "http_requests_total")
	cur4xx := getFloat(m, "http_4xx_total")
	cur5xx := getFloat(m, "http_5xx_total")

	deltaReqs := curReqs - baseReqs
	delta4xx := cur4xx - base4xx
	delta5xx := cur5xx - base5xx

	fmt.Printf("  Monitor Telemetry Check:\n")
	if baseReqs > 0 {
		fmt.Printf("    Test Run Session:      %.0f requests (4xx: %.0f, 5xx: %.0f in this run)\n", deltaReqs, delta4xx, delta5xx)
	}
	fmt.Printf("    Server Lifetime Total: %.0f requests (Cumulative since boot)\n", curReqs)
	fmt.Printf("    Server Lifetime 2xx:   %.0f\n", getFloat(m, "http_2xx_total"))
	fmt.Printf("    Server Lifetime 4xx:   %.0f (Cumulative total)\n", cur4xx)
	fmt.Printf("    Server Lifetime 5xx:   %.0f (Cumulative total)\n", cur5xx)
	fmt.Printf("    Server Avg Latency:    %.2f ms\n", getFloat(m, "avg_latency_ms"))
	fmt.Printf("    Server p50 Latency:    %.2f ms\n", getFloat(m, "p50_latency_ms"))
	fmt.Printf("    Server p95 Latency:    %.2f ms\n", getFloat(m, "p95_latency_ms"))
	fmt.Printf("    Server p99 Latency:    %.2f ms\n", getFloat(m, "p99_latency_ms"))
	fmt.Printf("    Server Goroutines:     %.0f\n", getFloat(m, "num_goroutines"))
	fmt.Printf("    Server Mem Allocated:  %.2f MB\n", getFloat(m, "memory_alloc_bytes")/1048576.0)

	if recent, ok := m["recent_errors"].([]any); ok && len(recent) > 0 {
		if lastErr, ok := recent[len(recent)-1].(map[string]any); ok {
			fmt.Printf("    Latest Server Error:   [%.0f] %v %v (%.0f ms)\n",
				getFloat(lastErr, "status"), lastErr["method"], lastErr["path"], getFloat(lastErr, "duration_ms"))
		}
	}

	if db, ok := m["db"].(map[string]any); ok {
		fmt.Printf("    DB Pool Connections:   %.0f active / %.0f idle (Max: %.0f)\n",
			getFloat(db, "active_conns"), getFloat(db, "idle_conns"), getFloat(db, "max_conns"))
		fmt.Printf("    DB Wait Count:         %.0f\n", getFloat(db, "wait_count"))
	}
	if rum, ok := m["rum"].(map[string]any); ok {
		fmt.Printf("    RUM Web Vitals:        Samples: %.0f | TTFB: %.1f ms | LCP: %.1f ms | INP: %.1f ms | CLS: %.3f\n",
			getFloat(rum, "samples_count"), getFloat(rum, "avg_ttfb_ms"), getFloat(rum, "avg_lcp_ms"),
			getFloat(rum, "avg_inp_ms"), getFloat(rum, "avg_cls"))
	}
	fmt.Printf("=================================================================\n")
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func runRUMGenerator(cfg Config) {
	fmt.Printf("--> Sending 50 RUM Telemetry batches to %s/rum...\n", cfg.MgmtURL)
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < 50; i++ {
		sendRUMSample(cfg.MgmtURL, client)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Printf("    [OK] Sent 50 RUM samples successfully.\n")
	verifyWithMonitor(cfg.MgmtURL, 0, 0, 0)
}
