#!/bin/bash

# High-performance metrics monitor for cashflow_backend
# Usage: ./tool/monitor.sh [url] [interval] [json|text]
#
#   url      default: http://127.0.0.1:8066/metrics/json (isolated management port)
#   interval default: 0.5s
#   mode     "json" (live dashboard) or "text" (raw Prometheus/OpenMetrics output)
#
# Set MONITOR_TOKEN to send a Bearer token when management.require_auth=true
# (e.g. when running under docker-compose).

URL=${1:-"http://127.0.0.1:8066/metrics/json"}
INTERVAL=${2:-0.5}
MODE=${3:-json}

AUTH=()
if [ -n "$MONITOR_TOKEN" ]; then
  AUTH=(-H "Authorization: Bearer $MONITOR_TOKEN")
fi

fetch() {
  curl -s --max-time 1 "${AUTH[@]}" "$URL"
}

# Plain text (Prometheus / OpenMetrics) repeater.
if [ "$MODE" = "text" ]; then
  while true; do
    clear
    DATA=$(fetch)
    echo "──────────────────────────────────────────────────────────────"
    echo "  CASHFLOW METRICS (raw exposition) — $URL"
    echo "  $(date "+%Y-%m-%d %H:%M:%S %p")"
    echo "──────────────────────────────────────────────────────────────"
    if [ -z "$DATA" ]; then
      echo -e "   \033[0;31mSERVER DOWN\033[0m — could not connect to $URL"
    else
      echo "$DATA"
    fi
    sleep "$INTERVAL"
  done
fi

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Hide cursor and ensure it shows back on exit
tput civis
trap "tput cnorm; exit" INT TERM EXIT

# Clear once at start
clear

PREV_TIME=""
PREV_REQS=""

while true; do
  DATA=$(fetch)

  # Move cursor to top-left instead of clearing
  printf "\033[H"

  if [ $? -ne 0 ] || [ -z "$DATA" ]; then
    echo -e "   ${RED}${BOLD}=== SERVER DOWN ===${NC}\033[K"
    echo -e "   Status: Could not connect to $URL\033[K"
    echo -e "   Time:   $(date "+%Y-%m-%d %H:%M:%S %p")\033[K"
    echo -e "   \033[K"
    echo -e "   \033[K"
    echo -e "   \033[K"
    echo -e "   \033[K"
    sleep "$INTERVAL"
    continue
  fi

  # Parse JSON values in a single jq invocation for maximum performance
  if ! command -v jq >/dev/null 2>&1; then
    echo -e "${RED}Error: 'jq' is required for the professional monitor.${NC}"
    exit 1
  fi

  PARSED=$(echo "$DATA" | jq -r '[
    (.server_url // "-"),
    (.memory_alloc_bytes // 0),
    (.memory_total_bytes // 0),
    (.memory_sys_bytes // 0),
    (.heap_inuse_bytes // 0),
    (.heap_idle_bytes // 0),
    (.num_goroutines // 0),
    (.num_gc // 0),
    (.gc_pause_seconds_total // 0),
    (.uptime_seconds // 0),
    (.http_requests_total // 0),
    (.http_2xx_total // 0),
    (.http_3xx_total // 0),
    (.http_4xx_total // 0),
    (.http_5xx_total // 0),
    (.avg_latency_ms // 0),
    (.p50_latency_ms // 0),
    (.p95_latency_ms // 0),
    (.p99_latency_ms // 0),
    (.avg_response_bytes // 0),
    (.readyz_latency_ms // 0),
    (.livez_latency_ms // 0),
    (.db.max_conns // 0),
    (.db.active_conns // 0),
    (.db.idle_conns // 0),
    (.db.wait_count // 0),
    (.db.empty_acquire_count // 0),
    (.db.wait_duration_ns // 0),
    (.window_seconds // 60),
    (.rum.samples_count // 0),
    (.rum.avg_lcp_ms // 0),
    (.rum.avg_cls // 0),
    (.rum.avg_inp_ms // 0),
    (.rum.avg_ttfb_ms // 0),
    (if .recent_errors and (.recent_errors | length) > 0 then
      (.recent_errors[-1] | "\(.status)|\(.method)|\(.path)|\(.duration_ms)")
    else
      "-"
    end)
  ] | @tsv' 2>/dev/null)

  if [ -z "$PARSED" ]; then
    sleep "$INTERVAL"
    continue
  fi

  read -r SERVER_URL ALLOC TOTAL SYS INUSE IDLE GO GC GC_PAUSE UPTIME \
          REQS S2XX S3XX E4XX E5XX LATENCY P50 P95 P99 RESP_BYTES \
          READYZ LIVEZ DB_MAX DB_ACTIVE DB_IDLE DB_WAIT \
          DB_EMPTY_ACQ DB_WAIT_DUR WIN_SEC \
          RUM_COUNT RUM_LCP RUM_CLS RUM_INP RUM_TTFB LAST_ERR <<< "$PARSED"

  # Formatting MBs
  ALLOC_MB=$(printf "%.2f" $(echo "scale=2; $ALLOC/1048576" | bc -l 2>/dev/null || echo 0))
  TOTAL_MB=$(printf "%.2f" $(echo "scale=2; $TOTAL/1048576" | bc -l 2>/dev/null || echo 0))
  SYS_MB=$(printf "%.2f" $(echo "scale=2; $SYS/1048576" | bc -l 2>/dev/null || echo 0))
  INUSE_MB=$(printf "%.2f" $(echo "scale=2; $INUSE/1048576" | bc -l 2>/dev/null || echo 0))
  IDLE_MB=$(printf "%.2f" $(echo "scale=2; $IDLE/1048576" | bc -l 2>/dev/null || echo 0))

  # Calculate real-time RPS
  CURR_TIME=$(date +%s.%N)
  RPS="0.0"
  if [ -n "$PREV_TIME" ] && [ -n "$PREV_REQS" ]; then
    TIME_DIFF=$(echo "$CURR_TIME - $PREV_TIME" | bc -l 2>/dev/null || echo 1)
    REQS_DIFF=$(echo "$REQS - $PREV_REQS" | bc -l 2>/dev/null || echo 0)
    if (( $(echo "$TIME_DIFF > 0.05" | bc -l 2>/dev/null || echo 0) )); then
      CALC_RPS=$(echo "$REQS_DIFF / $TIME_DIFF" | bc -l 2>/dev/null || echo 0)
      if (( $(echo "$CALC_RPS >= 0" | bc -l 2>/dev/null || echo 0) )); then
        RPS=$(printf "%.1f" "$CALC_RPS")
      fi
    fi
  fi
  PREV_TIME=$CURR_TIME
  PREV_REQS=$REQS

  # Average Latency Indicator
  LAT_COLOR=$GREEN
  if (( $(echo "$LATENCY > 400" | bc -l 2>/dev/null || echo 0) )); then
    LAT_COLOR=$RED
  elif (( $(echo "$LATENCY > 100" | bc -l 2>/dev/null || echo 0) )); then
    LAT_COLOR=$YELLOW
  fi

  # P95 SLO Indicator (SLO 500ms - 20% margin = 400ms alert threshold)
  P95_COLOR=$GREEN
  if (( $(echo "$P95 > 400" | bc -l 2>/dev/null || echo 0) )); then
    P95_COLOR=$RED
  elif (( $(echo "$P95 > 100" | bc -l 2>/dev/null || echo 0) )); then
    P95_COLOR=$YELLOW
  fi

  # P99 Indicator
  P99_COLOR=$GREEN
  if (( $(echo "$P99 > 500" | bc -l 2>/dev/null || echo 0) )); then
    P99_COLOR=$RED
  elif (( $(echo "$P99 > 200" | bc -l 2>/dev/null || echo 0) )); then
    P99_COLOR=$YELLOW
  fi

  # Error Indicator & 5xx Share (Alert rule: High5xxShare > 0.5%)
  SHARE_5XX="0.00"
  E5XX_COLOR=$NC
  if [ "$REQS" -gt 0 ]; then
    SHARE_5XX=$(printf "%.2f" $(echo "$E5XX * 100 / $REQS" | bc -l 2>/dev/null || echo 0))
    if (( $(echo "$SHARE_5XX > 0.5" | bc -l 2>/dev/null || echo 0) )); then
      E5XX_COLOR="${RED}${BOLD}"
    elif [ "$E5XX" -gt 0 ]; then
      E5XX_COLOR=$YELLOW
    fi
  fi
  E4XX_COLOR=$NC
  if [ "$E4XX" -gt 0 ]; then E4XX_COLOR=$YELLOW; fi

  ERR_LINE=""
  if [ "$LAST_ERR" != "-" ] && [ -n "$LAST_ERR" ]; then
    IFS='|' read -r ERR_STATUS ERR_METHOD ERR_PATH ERR_DUR <<< "$LAST_ERR"
    if [ -n "$ERR_STATUS" ] && [ "$ERR_STATUS" != "-" ]; then
      ERR_COLOR=$YELLOW
      if [ "$ERR_STATUS" -ge 500 ]; then ERR_COLOR="${RED}${BOLD}"; fi
      ERR_LINE="   Last Error: ${ERR_COLOR}[${ERR_STATUS}] ${ERR_METHOD} ${ERR_PATH} (${ERR_DUR}ms)${NC}"
    fi
  fi

  # DB Indicators (Alert rule: DBPoolSaturated > 80%, DBWaitClimbing > 50)
  DB_COLOR=$GREEN
  DB_USAGE="0.0"
  if [ "$DB_MAX" -gt 0 ]; then
    DB_USAGE=$(printf "%.1f" $(echo "$DB_ACTIVE * 100 / $DB_MAX" | bc -l 2>/dev/null || echo 0))
    if (( $(echo "$DB_USAGE > 80" | bc -l 2>/dev/null || echo 0) )); then
      DB_COLOR=$RED
    elif (( $(echo "$DB_USAGE > 50" | bc -l 2>/dev/null || echo 0) )); then
      DB_COLOR=$YELLOW
    fi
  fi
  WAIT_COLOR=$NC
  if [ "$DB_EMPTY_ACQ" -gt 0 ]; then
    WAIT_COLOR=$YELLOW
    if (( $(echo "$DB_USAGE > 80" | bc -l 2>/dev/null || echo 0) )); then
      WAIT_COLOR=$RED
    fi
  fi

  # Calculate acquire rate (differential like RPS)
  ACQ_RATE="0.0"
  if [ -n "$PREV_WAIT" ] && [ -n "$TIME_DIFF" ]; then
    WAIT_DIFF=$(echo "$DB_WAIT - $PREV_WAIT" | bc -l 2>/dev/null || echo 0)
    if (( $(echo "$TIME_DIFF > 0.05" | bc -l 2>/dev/null || echo 0) )); then
      CALC_ACQ=$(echo "$WAIT_DIFF / $TIME_DIFF" | bc -l 2>/dev/null || echo 0)
      if (( $(echo "$CALC_ACQ >= 0" | bc -l 2>/dev/null || echo 0) )); then
        ACQ_RATE=$(printf "%.1f" "$CALC_ACQ")
      fi
    fi
  fi
  PREV_WAIT=$DB_WAIT

  # Response size formatting
  if (( $(echo "$RESP_BYTES >= 1048576" | bc -l 2>/dev/null || echo 0) )); then
    RESP_STR="$(printf "%.2f MB" $(echo "$RESP_BYTES / 1048576" | bc -l 2>/dev/null || echo 0))"
  elif (( $(echo "$RESP_BYTES >= 1024" | bc -l 2>/dev/null || echo 0) )); then
    RESP_STR="$(printf "%.2f KB" $(echo "$RESP_BYTES / 1024" | bc -l 2>/dev/null || echo 0))"
  else
    RESP_STR="$(printf "%.0f B" "$RESP_BYTES")"
  fi

  # Health Probes Indicators
  READYZ_COLOR=$GREEN
  if (( $(echo "$READYZ > 20" | bc -l 2>/dev/null || echo 0) )); then READYZ_COLOR=$YELLOW; fi
  if (( $(echo "$READYZ > 100" | bc -l 2>/dev/null || echo 0) )); then READYZ_COLOR=$RED; fi

  LIVEZ_COLOR=$GREEN
  if (( $(echo "$LIVEZ > 20" | bc -l 2>/dev/null || echo 0) )); then LIVEZ_COLOR=$YELLOW; fi
  if (( $(echo "$LIVEZ > 100" | bc -l 2>/dev/null || echo 0) )); then LIVEZ_COLOR=$RED; fi

  # Formatting Uptime
  UPTIME_INT=$(printf "%.0f" "$UPTIME")
  UPTIME_H=$((UPTIME_INT / 3600))
  UPTIME_M=$((UPTIME_INT % 3600 / 60))
  UPTIME_S=$((UPTIME_INT % 60))
  UPTIME_STR=$(printf "%02dh %02dm %02ds" $UPTIME_H $UPTIME_M $UPTIME_S)

  # Render Dashboard
  echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\033[K"
  echo -e "   ${BLUE}${BOLD}CASHFLOW PRODUCTION RUNTIME MONITOR${NC}\033[K"
  echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\033[K"
  echo -e "   ${CYAN}Target:${NC}   ${SERVER_URL:-$URL}\033[K"
  echo -e "   ${CYAN}Time:${NC}     $(date "+%Y-%m-%d %H:%M:%S %p")\033[K"
  echo -e "   ${CYAN}Uptime:${NC}   $UPTIME_STR | ${CYAN}Goroutines:${NC} $GO | ${CYAN}GCs:${NC} $GC (${GC_PAUSE}s pause)\033[K"

  echo -e "\n   ${BOLD}[ SYSTEM & MEMORY ]${NC}\033[K"
  echo -e "   Allocated:  ${GREEN}${ALLOC_MB} MB${NC} (Active Objects)\033[K"
  echo -e "   In-Use:     ${GREEN}${INUSE_MB} MB${NC} (Heap Spans) | Idle: ${YELLOW}${IDLE_MB} MB${NC}\033[K"
  echo -e "   Total:      ${BLUE}${TOTAL_MB} MB${NC} (Cumulative)  | System: ${MAGENTA}${SYS_MB} MB${NC}\033[K"

  echo -e "\n   ${BOLD}[ TRAFFIC & THROUGHPUT ]${NC}\033[K"
  echo -e "   Requests:   ${BOLD}$REQS${NC} (Rate: ${CYAN}${RPS} req/s${NC})\033[K"
  echo -e "   Responses:  2xx: ${GREEN}$S2XX${NC} | 3xx: ${CYAN}$S3XX${NC} | Avg Size: ${BOLD}$RESP_STR${NC}\033[K"
  echo -e "   Errors:     4xx: ${E4XX_COLOR}$E4XX${NC} | 5xx: ${E5XX_COLOR}$E5XX${NC} (${E5XX_COLOR}${SHARE_5XX}%${NC})\033[K"
  if [ -n "$ERR_LINE" ]; then
    echo -e "${ERR_LINE}\033[K"
  fi

  echo -e "\n   ${BOLD}[ LATENCY & SLO TARGETS (${WIN_SEC:-60}s Window) ]${NC}\033[K"
  if (( $(echo "$LATENCY == 0 && $P95 == 0" | bc -l 2>/dev/null || echo 0) )); then
    echo -e "   Business:   ${CYAN}idle (no traffic in last ${WIN_SEC:-60}s)${NC} | p50: 0.00 ms | p95: 0.00 ms | p99: 0.00 ms\033[K"
  else
    echo -e "   Business:   Avg: ${LAT_COLOR}$(printf "%.2f" $LATENCY) ms${NC} | p50: ${GREEN}$(printf "%.2f" $P50) ms${NC} | p95: ${P95_COLOR}$(printf "%.2f" $P95) ms${NC} | p99: ${P99_COLOR}$(printf "%.2f" $P99) ms${NC}\033[K"
  fi
  echo -e "   Probes:     readyz: ${READYZ_COLOR}$(printf "%.2f" $READYZ) ms${NC} (DB Check) | livez: ${LIVEZ_COLOR}$(printf "%.2f" $LIVEZ) ms${NC}\033[K"

  echo -e "\n   ${BOLD}[ DATABASE POOL ]${NC}\033[K"
  if [ "$DB_MAX" -eq 0 ]; then
    echo -e "   ${MAGENTA}Status: No database registered${NC}\033[K"
  else
    echo -e "   Pool Conns: ${DB_COLOR}${DB_ACTIVE}${NC} active / ${DB_IDLE} idle (Max: ${DB_MAX}) [${DB_COLOR}${DB_USAGE}%${NC}]\033[K"
    echo -e "   Acquires:   ${DB_WAIT} total (${CYAN}${ACQ_RATE}/s${NC}) | Blocked: ${WAIT_COLOR}${DB_EMPTY_ACQ}${NC}\033[K"
  fi

  if [ "$RUM_COUNT" -gt 0 ]; then
    echo -e "\n   ${BOLD}[ REAL USER MONITORING (RUM) ]${NC}\033[K"
    echo -e "   Samples:    ${BOLD}$RUM_COUNT${NC} | TTFB: ${GREEN}$(printf "%.1f" $RUM_TTFB) ms${NC}\033[K"
    echo -e "   Core Web:   LCP: ${CYAN}$(printf "%.1f" $RUM_LCP) ms${NC} | INP: ${CYAN}$(printf "%.1f" $RUM_INP) ms${NC} | CLS: ${CYAN}$(printf "%.3f" $RUM_CLS)${NC}\033[K"
  fi

  echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\033[K"

  sleep "$INTERVAL"
done

