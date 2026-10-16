SHELL := /bin/bash
GO    ?= go
PKG   := ./...

.PHONY: help install tidy vet build test test-race vuln run clean

help:
	@echo "Targets:"
	@echo "  install    go mod download"
	@echo "  tidy       go mod tidy"
	@echo "  vet        go vet"
	@echo "  build      go build"
	@echo "  test       go test"
	@echo "  test-race  go test -race"
	@echo "  vuln       govulncheck (requiere instalación)"
	@echo "  run        go run . (lee .env)"
	@echo "  clean      go clean"

install:
	$(GO) mod download

tidy:
	$(GO) mod tidy

vet:
	$(GO) vet $(PKG)

build:
	$(GO) build $(PKG)

test:
	$(GO) test $(PKG)

test-race:
	$(GO) test -race $(PKG)

vuln:
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck $(PKG); \
	else \
		echo "[vuln] instala govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

run:
	$(GO) run .

clean:
	$(GO) clean
	rm -f server