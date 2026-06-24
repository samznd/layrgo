BINARY ?= layrgo
VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/samznd/layrgo/internal/cli.Version=$(VERSION) \
	-X github.com/samznd/layrgo/internal/cli.Commit=$(COMMIT) \
	-X github.com/samznd/layrgo/internal/cli.Date=$(DATE)

.PHONY: build install test test-integration vet run clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/layrgo

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/layrgo

test:
	go test ./... -short -count=1

test-integration:
	go test ./internal/scaffold/... -count=1 -timeout=10m

vet:
	go vet ./...

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin/
