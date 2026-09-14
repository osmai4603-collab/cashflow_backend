.PHONY: all build run stop test test-race bench lint clean docker-up docker-down docker-logs migrate-up migrate-down seed monitoring-up monitoring-down monitoring-logs monitoring-baseline monitoring-check alertmanager-logs db-up db-down monitor

BINARY_NAME=bin/server
MIGRATE_NAME=bin/migrate

all: test build

build:
	@mkdir -p bin
	go build -o $(BINARY_NAME) ./cmd/server
	go build -o $(MIGRATE_NAME) ./cmd/migrate

run: build
	@CASHFLOW_LOG_COLOR=true ./$(BINARY_NAME)

stop:
	@pid=$$(pgrep -f "^(\./)?$(BINARY_NAME)"); \
	if [ -n "$$pid" ]; then \
		kill -TERM $$pid && echo "Sent SIGTERM to server (PID $$pid)"; \
	else \
		echo "No running server process found"; \
	fi

test:
	go test -v ./...

test-race:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem ./...

lint:
	go vet ./...

clean:
	rm -rf bin/

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

seed:
	@echo "Seeding database with initial ERP configuration..."
	@echo "Database is ready for domain entities."

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

db-up:
	docker compose up -d postgres redis

db-down:
	docker compose stop postgres redis

monitor:
	@chmod +x tool/monitor.sh
	./tool/monitor.sh

monitoring-up:
	docker compose up -d
	docker compose -f docker-compose.monitoring.yml up -d

monitoring-down:
	docker compose -f docker-compose.monitoring.yml down

monitoring-logs:
	docker compose -f docker-compose.monitoring.yml logs -f

alertmanager-logs:
	docker compose -f docker-compose.monitoring.yml logs -f alertmanager

# فحص صياغة ملفات المراقبة الاثنين دون سحب صور (يعمل حتى بدون إنترنت Docker).
monitoring-check:
	@echo "==> main compose" && docker compose config --quiet
	@echo "==> monitoring compose" && docker compose -f docker-compose.monitoring.yml config --quiet
	@python3 -c "import yaml,json;[yaml.safe_load(open(f)) or None for f in ['deploy/monitoring/prometheus.yml','deploy/monitoring/prometheus-alerts.yml','deploy/monitoring/alertmanager/alertmanager.yml']];json.load(open('deploy/monitoring/grafana/dashboards/cashflow-overview.json'));print('==> yaml/json ok')"

monitoring-baseline:
	@chmod +x tool/calibrate_baseline.sh
	./tool/calibrate_baseline.sh

