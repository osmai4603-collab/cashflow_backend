#!/bin/bash

# High-performance metrics monitor for cashflow_backend
# Usage: ./tool/monitor.sh [url] [interval]

URL=${1:-"http://localhost:8070/metrics"}
INTERVAL=${2:-0.5}

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

while true; do
  DATA=$(curl -s --max-time 1 "$URL")

  # Move cursor to top-left instead of clearing
  printf "\033[H"

  if [ $? -ne 0 ] || [ -z "$DATA" ]; then
    echo -e "   ${RED}${BOLD}=== SERVER DOWN ===${NC}                                "
    echo -e "   Status: Could not connect to $URL                  "
    echo -e "   Time:   $(date "+%Y-%m-%d %H:%M:%S %p")            "
    echo -e "                                                      "
    echo -e "                                                      "
    echo -e "                                                      "
    echo -e "                                                      "
    sleep "$INTERVAL"
    continue
  fi

  # Parse JSON values using jq (Mandatory for this version)
  if ! command -v jq >/dev/null 2>&1; then
    echo -e "${RED}Error: 'jq' is required for the professional monitor.${NC}"
    exit 1
  fi

  # System Metrics
  SERVER_URL=$(echo "$DATA" | jq -r '.server_url // empty')
  ALLOC=$(echo "$DATA" | jq -r '.memory_alloc_bytes')
  TOTAL=$(echo "$DATA" | jq -r '.memory_total_bytes')
  SYS=$(echo "$DATA" | jq -r '.memory_sys_bytes')
  INUSE=$(echo "$DATA" | jq -r '.heap_inuse_bytes')
  IDLE=$(echo "$DATA" | jq -r '.heap_idle_bytes')
  GO=$(echo "$DATA" | jq -r '.num_goroutines')
  GC=$(echo "$DATA" | jq -r '.num_gc')
  UPTIME=$(echo "$DATA" | jq -r '.uptime_seconds')

  # Traffic Metrics
  REQS=$(echo "$DATA" | jq -r '.http_requests_total')
  S2XX=$(echo "$DATA" | jq -r '.http_2xx_total')
  E4XX=$(echo "$DATA" | jq -r '.http_4xx_total')
  E5XX=$(echo "$DATA" | jq -r '.http_5xx_total')
  LATENCY=$(echo "$DATA" | jq -r '.avg_latency_ms')

  # DB Metrics
  DB_MAX=$(echo "$DATA" | jq -r '.db.max_conns // 0')
  DB_ACTIVE=$(echo "$DATA" | jq -r '.db.active_conns // 0')
  DB_IDLE=$(echo "$DATA" | jq -r '.db.idle_conns // 0')
  DB_WAIT=$(echo "$DATA" | jq -r '.db.wait_count // 0')

  # Formatting MBs
  ALLOC_MB=$(printf "%.2f" $(echo "scale=2; $ALLOC/1048576" | bc -l))
  TOTAL_MB=$(printf "%.2f" $(echo "scale=2; $TOTAL/1048576" | bc -l))
  SYS_MB=$(printf "%.2f" $(echo "scale=2; $SYS/1048576" | bc -l))
  INUSE_MB=$(printf "%.2f" $(echo "scale=2; $INUSE/1048576" | bc -l))
  IDLE_MB=$(printf "%.2f" $(echo "scale=2; $IDLE/1048576" | bc -l))

  # Latency Indicator
  LAT_COLOR=$GREEN
  if (( $(echo "$LATENCY > 500" | bc -l) )); then LAT_COLOR=$RED; elif (( $(echo "$LATENCY > 100" | bc -l) )); then LAT_COLOR=$YELLOW; fi

  # Error Indicator
  ERR_COLOR=$NC
  if [ "$E5XX" -gt 0 ]; then ERR_COLOR=$RED; elif [ "$E4XX" -gt 0 ]; then ERR_COLOR=$YELLOW; fi

  # DB Indicator
  DB_COLOR=$GREEN
  if [ "$DB_MAX" -gt 0 ]; then
    DB_USAGE=$(echo "scale=2; $DB_ACTIVE * 100 / $DB_MAX" | bc -l)
    if (( $(echo "$DB_USAGE > 80" | bc -l) )); then DB_COLOR=$RED; elif (( $(echo "$DB_USAGE > 50" | bc -l) )); then DB_COLOR=$YELLOW; fi
  fi

  # Formatting Uptime
  UPTIME_INT=$(printf "%.0f" $UPTIME)
  UPTIME_H=$((UPTIME_INT / 3600))
  UPTIME_M=$((UPTIME_INT % 3600 / 60))
  UPTIME_S=$((UPTIME_INT % 60))
  UPTIME_STR=$(printf "%02dh %02dm %02ds" $UPTIME_H $UPTIME_M $UPTIME_S)

  # Render Dashboard
  echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "   ${BLUE}${BOLD}CASHFLOW PRODUCTION RUNTIME MONITOR${NC}               "
  echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "   ${CYAN}URL:${NC}     ${SERVER_URL:-$URL}"
  echo -e "   ${CYAN}Time:${NC}    $(date "+%Y-%m-%d %H:%M:%S %p")"
  echo -e "   ${CYAN}Uptime:${NC}  $UPTIME_STR | ${CYAN}Goroutines:${NC} $GO | ${CYAN}GCs:${NC} $GC"

  echo -e "\n   ${BOLD}[ SYSTEM & MEMORY ]${NC}"
  echo -e "   Allocated:  ${GREEN}${ALLOC_MB} MB${NC} (Active Objects)"
  echo -e "   In-Use:     ${GREEN}${INUSE_MB} MB${NC} (Heap Spans) | Idle: ${YELLOW}${IDLE_MB} MB${NC}"
  echo -e "   Total:      ${BLUE}${TOTAL_MB} MB${NC} (Cumulative) | System: ${MAGENTA}${SYS_MB} MB${NC}"

  echo -e "\n   ${BOLD}[ TRAFFIC & HEALTH ]${NC}"
  echo -e "   Requests:   ${BOLD}$REQS${NC} | Success: ${GREEN}$S2XX${NC}"
  echo -e "   Avg Latency: ${LAT_COLOR}${LATENCY} ms${NC}"
  echo -e "   Errors:     4xx: ${YELLOW}$E4XX${NC} | 5xx: ${RED}${BOLD}$E5XX${NC}"

  echo -e "\n   ${BOLD}[ DATABASE POOL ]${NC}"
  if [ "$DB_MAX" -eq 0 ]; then
    echo -e "   ${MAGENTA}Status: No database registered${NC}"
  else
    echo -e "   Connections: ${DB_COLOR}${DB_ACTIVE}${NC} active / ${DB_IDLE} idle (Max: ${DB_MAX})"
    echo -e "   Wait Count:  ${DB_WAIT}"
  fi
  echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

  sleep "$INTERVAL"
done
