SHELL := /bin/bash
NPM  ?= npm

.PHONY: help install dev build lint typecheck audit preview clean

help:
	@echo "Targets:"
	@echo "  install     npm ci"
	@echo "  dev         npm run dev"
	@echo "  build       npm run build"
	@echo "  lint        npm run lint"
	@echo "  typecheck   npx tsc --noEmit"
	@echo "  audit       npm audit --omit=dev (falla si high+)"
	@echo "  preview     npm run preview"
	@echo "  clean       rm -rf dist"

install:
	$(NPM) ci

dev:
	$(NPM) run dev

build:
	$(NPM) run build

lint:
	$(NPM) run lint

typecheck:
	npx tsc --noEmit

audit:
	@if [ "$$(npm audit --omit=dev --json | jq '.metadata.vulnerabilities.high + .metadata.vulnerabilities.critical')" != "0" ]; then \
		echo "Vulnerabilidades high/critical detectadas"; npm audit --omit=dev; exit 1; \
	else \
		echo "OK: 0 vulnerabilidades high/critical"; \
	fi

preview:
	$(NPM) run preview

clean:
	rm -rf dist