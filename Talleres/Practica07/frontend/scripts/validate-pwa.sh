#!/bin/bash

# PWA Phase C Validation Script
# Validates generated PWA artifacts without requiring browser

set -e

echo "🔍 PWA Phase C Validation"
echo "=========================="
echo ""

DIST="./dist"
ERRORS=0

# Check 1: manifest.webmanifest exists
echo "✓ Check 1: manifest.webmanifest"
if [ -f "$DIST/manifest.webmanifest" ]; then
    echo "  ✅ manifest.webmanifest found"
    # Validate it's valid JSON
    if jq empty "$DIST/manifest.webmanifest" 2>/dev/null; then
        echo "  ✅ manifest is valid JSON"
        # Check critical fields
        NAME=$(jq -r '.name // empty' "$DIST/manifest.webmanifest")
        START_URL=$(jq -r '.start_url // empty' "$DIST/manifest.webmanifest")
        DISPLAY=$(jq -r '.display // empty' "$DIST/manifest.webmanifest")

        [ -n "$NAME" ] && echo "  ✅ name: $NAME"
        [ -n "$START_URL" ] && echo "  ✅ start_url: $START_URL"
        [ "$DISPLAY" = "standalone" ] && echo "  ✅ display: standalone"
    else
        echo "  ❌ manifest is not valid JSON"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "  ❌ manifest.webmanifest NOT found"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 2: service worker exists
echo "✓ Check 2: Service Worker (sw.js)"
if [ -f "$DIST/sw.js" ]; then
    echo "  ✅ sw.js found ($(wc -c < $DIST/sw.js | numfmt --to=iec 2>/dev/null || wc -c < $DIST/sw.js) bytes)"

    # Check for precache
    if grep -q "precacheAndRoute" "$DIST/sw.js"; then
        echo "  ✅ precacheAndRoute registered"
    else
        echo "  ❌ precacheAndRoute not found"
        ERRORS=$((ERRORS + 1))
    fi

    # Check for runtime cache
    if grep -q "registerRoute" "$DIST/sw.js"; then
        echo "  ✅ registerRoute (runtime cache) found"
    else
        echo "  ⚠️  No runtime cache routes defined"
    fi

    # Check for cleanup
    if grep -q "cleanupOutdatedCaches" "$DIST/sw.js"; then
        echo "  ✅ cleanupOutdatedCaches registered"
    fi
else
    echo "  ❌ sw.js NOT found"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 3: Workbox runtime
echo "✓ Check 3: Workbox Runtime"
if [ -f "$DIST/workbox-"*.js ]; then
    echo "  ✅ Workbox runtime found ($(ls -la $DIST/workbox-*.js | awk '{print $5}' | numfmt --to=iec 2>/dev/null || echo 'unknown size'))"
else
    echo "  ❌ Workbox runtime NOT found"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 4: index.html
echo "✓ Check 4: index.html"
if [ -f "$DIST/index.html" ]; then
    echo "  ✅ index.html found"
    if grep -q 'registerSW' "$DIST/index.html"; then
        echo "  ✅ Service Worker registration code present"
    fi
else
    echo "  ❌ index.html NOT found"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 5: Icon/favicon
echo "✓ Check 5: Icons"
if [ -f "$DIST/favicon.svg" ]; then
    echo "  ✅ favicon.svg found"
else
    echo "  ⚠️  favicon.svg not found (optional)"
fi
echo ""

# Summary
echo "=========================="
if [ $ERRORS -eq 0 ]; then
    echo "✅ PWA Phase C VALIDATION PASSED"
    echo ""
    echo "📝 Manual validation still required (requires browser):"
    echo "  1. Run: npm run preview"
    echo "  2. Open: http://localhost:4173"
    echo "  3. DevTools → Application:"
    echo "     - manifest.webmanifest should load"
    echo "     - Service Workers should show 'registered'"
    echo "  4. DevTools → Network:"
    echo "     - Toggle offline"
    echo "     - Load /catalogo → should load cached /api/productos"
    echo "  5. Install prompt should appear after 3s"
    exit 0
else
    echo "❌ PWA Phase C VALIDATION FAILED ($ERRORS errors)"
    exit 1
fi
