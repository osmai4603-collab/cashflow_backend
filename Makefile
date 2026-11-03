.PHONY: all run build test clean add-feature

MODULES = ./apps/api/... ./internal/infrastructure/... ./internal/shared/...

all: build

run:
	go run ./apps/api

build:
	mkdir -p bin
	go build -o bin/api ./apps/api

test:
	go test -v -race $(MODULES)

clean:
	rm -rf bin/ logs/

# Usage: make add-feature name=budgets
add-feature:
	@if [ -z "$(name)" ]; then echo "Error: Please specify feature name (e.g., make add-feature name=budgets)"; exit 1; fi
	@echo "Creating new feature module: internal/features/$(name)"
	@mkdir -p internal/features/$(name)
	@echo "package $(name)\n" > internal/features/$(name)/model.go
	@echo "Feature $(name) created successfully under internal/features/$(name)!"
