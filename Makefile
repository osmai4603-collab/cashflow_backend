.PHONY: all build run stop test test-race bench lint clean docker-up docker-down docker-logs migrate-up migrate-down seed

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
		kill -SIGTERM $$pid && echo "Sent SIGTERM to server (PID $$pid)"; \
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

