SHELL := /bin/bash
NPM  ?= pnpm

.PHONY: help install dev build lint typecheck test test-watch test-coverage audit preview clean

help:
	@echo "Targets:"
	@echo "  install     npm ci"
	@echo "  dev         npm run dev"
	@echo "  build       npm run build"
	@echo "  lint        npm run lint"
	@echo "  typecheck   npx tsc -b"
	@echo "  test        vitest run (una pasada)"
	@echo "  test-watch  vitest (modo interactivo)"
	@echo "  test-coverage  vitest --coverage + HTML"
	@echo "  audit       npm audit --omit=dev (falla si high+)"
	@echo "  preview     npm run preview"
	@echo "  clean       rm -rf dist"

# El lockfile canónico es pnpm-lock.yaml; no existe package-lock.json.
# --frozen-lockfile falla si el lock no cuadra con package.json, que es
# justo lo que queremos en CI: instalar versiones distintas a las
# probadas localmente es la forma más fácil de tener un CI verde que no
# reproduce el bug.
install:
	$(NPM) install --frozen-lockfile

dev:
	$(NPM) run dev

build:
	$(NPM) run build

lint:
	$(NPM) run lint

typecheck:
	$(NPM) exec tsc -b

# `npm run test` = vitest run, una pasada y sale. El CI usa este, nunca
# test-watch: un watcher en CI se queda colgado esperando teclado.
test:
	$(NPM) run test

test-watch:
	$(NPM) run test:watch

test-coverage:
	$(NPM) run test:coverage
	@echo ""
	@echo "HTML en frontend/coverage/index.html"

# pnpm audit es el que corresponde al lockfile canónico. `npm audit` exige un
# package-lock.json y sin él aborta con "This command requires an existing
# lockfile", así que dejarlo aquí era un fallo silencioso esperando.
audit:
	@if [ "$$($(NPM) audit --json 2>/dev/null | jq '[.advisories[]? | select(.severity=="high" or .severity=="critical")] | length')" != "0" ]; then \
		echo "Vulnerabilidades high/critical detectadas"; $(NPM) audit; exit 1; \
	else \
		echo "OK: 0 vulnerabilidades high/critical"; \
	fi

preview:
	$(NPM) run preview

clean:
	rm -rf dist