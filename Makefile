BINARY := adb-tui
BUILD_DIR := ./bin
MODULE := github.com/alanisme/adb-tui

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)

COVERAGE_OUT := coverage.out
COVERAGE_MIN := 29

.PHONY: build install test test-race test-verbose cover cover-html lint lint-fix vet check fmt clean dev ci

## Build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/adb-tui

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/adb-tui

## Test

test:
	go test ./...

test-race:
	go test -race ./...

test-verbose:
	go test -v -race ./...

cover:
	go test -race -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_OUT) | tail -1
	@echo ""
	@echo "Per-package breakdown:"
	@go tool cover -func=$(COVERAGE_OUT) | grep "^total" || true
	@go test -cover ./... 2>&1 | grep -v "^ok" | grep -v "^?" || true

cover-html: cover
	go tool cover -html=$(COVERAGE_OUT)

cover-check: cover
	@total=$$(go tool cover -func=$(COVERAGE_OUT) | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
	threshold=$(COVERAGE_MIN); \
	echo "Total coverage: $${total}%  (minimum: $${threshold}%)"; \
	if [ $$(echo "$${total} < $${threshold}" | bc -l) -eq 1 ]; then \
		echo "FAIL: coverage below threshold"; exit 1; \
	fi

## Lint

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

fmt-check:
	@test -z "$$(gofmt -s -l . 2>&1)" || (echo "Files need formatting:"; gofmt -s -l .; exit 1)

## Quality gates — run all checks (for CI or pre-push)

check: fmt-check vet lint test-race

ci: fmt-check vet lint cover-check

## Dev

dev: build
	$(BUILD_DIR)/$(BINARY)

## Clean

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_OUT)

## Help

help:
	@echo "Targets:"
	@echo "  build        Build the binary"
	@echo "  install      Install to GOPATH/bin"
	@echo "  test         Run tests"
	@echo "  test-race    Run tests with race detector"
	@echo "  cover        Generate coverage report"
	@echo "  cover-html   Open coverage in browser"
	@echo "  cover-check  Fail if coverage below $(COVERAGE_MIN)%"
	@echo "  lint         Run golangci-lint"
	@echo "  lint-fix     Run golangci-lint with auto-fix"
	@echo "  vet          Run go vet"
	@echo "  fmt          Format code"
	@echo "  fmt-check    Check formatting (no write)"
	@echo "  check        Run all quality checks (local)"
	@echo "  ci           Run all CI checks with coverage gate"
	@echo "  dev          Build and run"
	@echo "  clean        Remove build artifacts"
