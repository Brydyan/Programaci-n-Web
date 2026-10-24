SHELL := /bin/bash
GO    ?= go
PKG   := ./...

.PHONY: help install tidy vet build test test-race test-verbose coverage coverage-func lint vuln run clean

help:
	@echo "Targets:"
	@echo "  install    go mod download"
	@echo "  tidy       go mod tidy"
	@echo "  vet        go vet"
	@echo "  build      go build"
	@echo "  test       go test"
	@echo "  test-race  go test -race"
	@echo "  test-verbose go test -v"
	@echo "  coverage   perfil + HTML + resumen por función"
	@echo "  coverage-func  sólo el resumen por función (rápido)"
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

test-verbose:
	$(GO) test $(PKG) -v

# coverage genera el perfil, el HTML y el resumen por función.
# El HTML es un artefacto de trabajo: se regenera, no se versiona.
coverage:
	$(GO) test $(PKG) -coverprofile=coverage.out -covermode=atomic
	@echo ""
	@echo "── cobertura por paquete ──"
	@$(GO) test $(PKG) -cover
	@echo ""
	@echo "── resumen por función ──"
	@$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "HTML en backend/coverage.html (abrir con: make coverage-open)"

coverage-func:
	$(GO) test $(PKG) -coverprofile=coverage.out -covermode=atomic
	$(GO) tool cover -func=coverage.out

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
	rm -f server coverage.out coverage.html