.PHONY: all build test test-coverage clean dist

BINARY_NAME=acprun
BIN_DIR=bin
VERSION?=1.0.0
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-s -w -X github.com/baldaworks/acprun/internal/cli.Version=$(VERSION) \
              -X github.com/baldaworks/acprun/internal/cli.Commit=$(COMMIT) \
              -X github.com/baldaworks/acprun/internal/cli.Date=$(DATE)

all: build

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/acprun

test:
	CGO_ENABLED=0 go test -v -race ./...

test-coverage:
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -rf $(BIN_DIR) .omnidist/default/dist .omnidist/default/stage coverage.out

dist:
	omnidist build
	omnidist stage
