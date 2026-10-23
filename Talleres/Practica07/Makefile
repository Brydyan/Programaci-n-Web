# Makefile raíz · MultiCatálogo (Practica07)
#
# Orquestación de alto nivel: base de datos, backend y frontend.
# Cada proyecto tiene su propio Makefile; aquí sólo se delegan.
#
#   make help            lista los targets
#   make up              levanta postgres + redis (sin backend)
#   make db-migrate      aplica las migraciones en orden
#   make db-reset        borra y recrea la base, luego aplica todo
#   make db-status       muestra tablas y conteos (verificación rápida)

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ── Configuración DB ─────────────────────────────────────────────────────────
# Puerto 5433 en el host: 5432 lo ocupa el Postgres local del sistema y 5433
# el contenedor `postgres-funcional` de otro proyecto (ver PROJECT_MAP.md).
DB_HOST     ?= localhost
DB_PORT     ?= 5433
DB_USER     ?= admin
DB_PASSWORD ?= secret
DB_NAME     ?= multicatalogo
TEST_DB_NAME ?= multicatalogo_test
DB_URL      ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
TEST_DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(TEST_DB_NAME)?sslmode=disable
ADMIN_URL   ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/postgres?sslmode=disable

MIGRATIONS_DIR := database/migrations
COMPOSE       := docker compose

.PHONY: help up down ps logs db-up db-migrate db-reset db-psql db-status db-test \
        backend-test frontend-lint verify clean-volumes

help:
	@echo "MultiCatálogo · targets disponibles:"
	@echo ""
	@echo "  infra"
	@echo "    up              docker compose up -d postgres redis"
	@echo "    down            docker compose down"
	@echo "    ps              estado de los contenedores"
	@echo "    logs            sigue los logs de todos los servicios"
	@echo ""
	@echo "  base de datos"
	@echo "    db-up           levanta postgres y espera a que esté healthy"
	@echo "    db-migrate      aplica $(MIGRATIONS_DIR)/*.sql en orden"
	@echo "    db-reset        DESTRUCTIVO: borra la base y la reconstruye"
	@echo "    db-psql         shell psql contra la base de desarrollo"
	@echo "    db-status       tablas + conteos del seed"
	@echo "    db-test         crea $(TEST_DB_NAME) + migraciones para los tests"
	@echo ""
	@echo "  calidad"
	@echo "    verify          vet + build + tests (con DB de test) + lint + typecheck"
	@echo ""
	@echo "  danger"
	@echo "    clean-volumes   DESTRUCTIVO: borra el volumen de Postgres"

# ── Infra ────────────────────────────────────────────────────────────────────
up:
	$(COMPOSE) up -d postgres redis

down:
	$(COMPOSE) down

ps:
	$(COMPOSE) ps -a

logs:
	$(COMPOSE) logs -f

# ── Base de datos ────────────────────────────────────────────────────────────
db-up: up
	@echo "→ esperando a que Postgres acepte conexiones en localhost:$(DB_PORT)..."
	@for i in $$(seq 1 60); do \
		if psql "$(DB_URL)" -tAc "SELECT 1" >/dev/null 2>&1; then \
			echo "→ Postgres listo"; exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "✗ Postgres no respondió en 60s. Revisa: docker compose logs postgres"; exit 1

# Aplica cada migración en orden alfabético. ON_ERROR_STOP=1 para que un error
# en un archivo aborta en vez de dejar la base a medias.
db-migrate:
	@for f in $(MIGRATIONS_DIR)/*.sql; do \
		echo "→ aplicando $$f"; \
		psql "$(DB_URL)" -v ON_ERROR_STOP=1 -q -f "$$f" || exit 1; \
	done
	@echo "→ migraciones aplicadas"

db-reset:
	@echo "⚠  DESTRUCTIVO: se borra la base '$(DB_NAME)' en localhost:$(DB_PORT)"
	@psql "$(ADMIN_URL)" -v ON_ERROR_STOP=1 -q \
		-c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$(DB_NAME)' AND pid <> pg_backend_pid();" \
		-c "DROP DATABASE IF EXISTS $(DB_NAME);" \
		-c "CREATE DATABASE $(DB_NAME);"
	@$(MAKE) --no-print-directory db-migrate

db-psql:
	psql "$(DB_URL)"

db-status:
	@psql "$(DB_URL)" -c "\dt" \
		-c "SELECT (SELECT count(*) FROM usuarios) AS usuarios, \
		    (SELECT count(*) FROM productos) AS productos, \
		    (SELECT count(*) FROM referidos) AS referidos, \
		    (SELECT count(*) FROM pedidos) AS pedidos, \
		    (SELECT count(*) FROM comisiones) AS comisiones;"

# db-test crea una base separada para los tests que pegan contra PostgreSQL y
# le aplica las migraciones. Motivo: los tests de integración insertan usuarios
# y nodos de referidos; si corrieran contra la base de desarrollo dejarían
# basura entre sesiones.
db-test:
	@if ! psql "$(ADMIN_URL)" -tAc "SELECT 1 FROM pg_database WHERE datname='$(TEST_DB_NAME)'" | grep -q 1; then \
		psql "$(ADMIN_URL)" -v ON_ERROR_STOP=1 -q -c "CREATE DATABASE $(TEST_DB_NAME)"; \
	fi
	@for f in $(MIGRATIONS_DIR)/*.sql; do \
		echo "→ aplicando $$f en $(TEST_DB_NAME)"; \
		psql "$(TEST_DB_URL)" -v ON_ERROR_STOP=1 -q -f "$$f" || exit 1; \
	done
	@echo "→ base de test lista"

# ── Calidad ──────────────────────────────────────────────────────────────────
backend-test:
	@$(MAKE) -C backend vet build test

# test-backend apunta los tests de integración a la base de test, de modo que
# `make verify` cubre SQL de verdad y no se salta nada.
test-backend: db-test
	@TEST_DATABASE_URL="$(TEST_DB_URL)" $(MAKE) -C backend test

frontend-lint:
	@$(MAKE) -C frontend lint typecheck

verify: test-backend frontend-lint
	@$(MAKE) -C backend vet build
	@echo "→ todo verde"

# ── Peligro ──────────────────────────────────────────────────────────────────
clean-volumes:
	@echo "⚠  DESTRUCTIVO: se borra el volumen multicatalogo_pgdata"
	$(COMPOSE) down -v