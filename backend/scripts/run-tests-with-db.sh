#!/bin/bash

# Run backend tests with PostgreSQL
# Starts docker-compose services, runs tests, stops services

set -e

echo "🧪 Backend Integration Tests with PostgreSQL"
echo "=============================================="
echo ""

# Check if docker compose is available
if ! command -v docker compose &> /dev/null; then
    echo "❌ docker compose not found"
    echo "Install Docker Desktop or Docker Engine with Compose support"
    exit 1
fi

echo "📦 Starting PostgreSQL + Redis..."
docker compose up -d postgres redis 2>&1 | grep -E "Running|Created|Healthy" || true

echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5

# Check connection
if ! pg_isready -h localhost -p 5432 -U postgres -d multicatalogo &>/dev/null; then
    echo "⚠️  PostgreSQL not ready yet, waiting..."
    for i in {1..10}; do
        sleep 2
        if pg_isready -h localhost -p 5432 -U postgres -d multicatalogo &>/dev/null; then
            echo "✅ PostgreSQL is ready"
            break
        fi
    done
fi

echo ""
echo "🔬 Running integration tests..."
echo ""

# Export DATABASE_URL for tests
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/multicatalogo?sslmode=disable"

# Run tests
go test ./controllers -v \
    -timeout=30s \
    -run "Test" \
    2>&1 | tee /tmp/test-results.log

# Count results
PASS=$(grep -c "^--- PASS:" /tmp/test-results.log || echo "0")
FAIL=$(grep -c "^--- FAIL:" /tmp/test-results.log || echo "0")
SKIP=$(grep -c "^--- SKIP:" /tmp/test-results.log || echo "0")

echo ""
echo "=============================================="
echo "📊 Test Summary"
echo "=============================================="
echo "✅ PASS: $PASS"
echo "❌ FAIL: $FAIL"
echo "⏭️  SKIP: $SKIP"
echo ""

# Cleanup
echo "🧹 Stopping PostgreSQL + Redis..."
docker compose down -v postgres redis 2>&1 | grep -E "Removed|removed" || true

if [ "$FAIL" -gt 0 ]; then
    echo "❌ Tests failed"
    exit 1
else
    echo "✅ All tests passed"
    exit 0
fi
