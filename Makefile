.PHONY: all build run test test-race lint clean

BINARY_NAME=bin/server

all: test build

build:
	@mkdir -p bin
	go build -o $(BINARY_NAME) ./cmd/server

run: build
	@./$(BINARY_NAME)

test:
	go test -v ./...

test-race:
	go test -v -race ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
