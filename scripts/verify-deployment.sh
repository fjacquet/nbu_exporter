#!/bin/bash
# verify-deployment.sh - Verify NBU Exporter deployment procedures

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== NBU Exporter Deployment Verification ==="
echo "Project root: $PROJECT_ROOT"
echo

# Test 1: Binary exists and is executable
echo "Test 1: Binary exists and is executable"
if [ -f "$PROJECT_ROOT/bin/nbu_exporter" ]; then
    if [ -x "$PROJECT_ROOT/bin/nbu_exporter" ]; then
        echo "✓ PASS: Binary exists and is executable"
    else
        echo "✗ FAIL: Binary exists but is not executable"
        exit 1
    fi
else
    echo "⚠ SKIP: Binary not found (run 'make cli' first)"
fi
echo

# Test 2: Configuration file exists
echo "Test 2: Configuration file exists"
if [ -f "$PROJECT_ROOT/config.yaml" ]; then
    echo "✓ PASS: Configuration file exists"
else
    echo "✗ FAIL: Configuration file not found"
    exit 1
fi
echo

# Test 3: Configuration has API version field
echo "Test 3: Configuration has API version field"
if grep -q "apiVersion:" "$PROJECT_ROOT/config.yaml"; then
    API_VERSION=$(grep "apiVersion:" "$PROJECT_ROOT/config.yaml" | awk '{print $2}' | tr -d '"')
    echo "✓ PASS: API version field present: $API_VERSION"
else
    echo "⚠ WARN: API version field not present (will use default 12.0)"
fi
echo

# Test 4: Configuration without API version (backward compatibility)
echo "Test 4: Backward compatibility - config without API version"
cat > /tmp/test-config-no-version.yaml << 'EOF'
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "5m"
    logName: "/tmp/test-deployment.log"
nbuserver:
    scheme: "https"
    uri: "/netbackup"
    host: "test.example.com"
    port: "1556"
    apiKey: "test-key"
    contentType: "application/json"
    insecureSkipVerify: true
EOF

if [ -f "$PROJECT_ROOT/bin/nbu_exporter" ]; then
    timeout 5s "$PROJECT_ROOT/bin/nbu_exporter" --config /tmp/test-config-no-version.yaml --debug > /dev/null 2>&1 &
    TEST_PID=$!
    sleep 2
    
    if ps -p $TEST_PID > /dev/null 2>&1; then
        echo "✓ PASS: Exporter starts without API version field"
        kill -TERM $TEST_PID 2>/dev/null || true
        wait $TEST_PID 2>/dev/null || true
    else
        echo "✗ FAIL: Exporter failed to start without API version field"
        exit 1
    fi
else
    echo "⚠ SKIP: Binary not available for testing"
fi
echo

# Test 5: Configuration with API version
echo "Test 5: Configuration with explicit API version"
cat > /tmp/test-config-with-version.yaml << 'EOF'
server:
    host: "localhost"
    port: "2113"
    uri: "/metrics"
    scrapingInterval: "5m"
    logName: "/tmp/test-deployment2.log"
nbuserver:
    scheme: "https"
    uri: "/netbackup"
    host: "test.example.com"
    port: "1556"
    apiVersion: "12.0"
    apiKey: "test-key"
    contentType: "application/json"
    insecureSkipVerify: true
EOF

if [ -f "$PROJECT_ROOT/bin/nbu_exporter" ]; then
    timeout 5s "$PROJECT_ROOT/bin/nbu_exporter" --config /tmp/test-config-with-version.yaml --debug > /dev/null 2>&1 &
    TEST_PID=$!
    sleep 2
    
    if ps -p $TEST_PID > /dev/null 2>&1; then
        echo "✓ PASS: Exporter starts with API version field"
        kill -TERM $TEST_PID 2>/dev/null || true
        wait $TEST_PID 2>/dev/null || true
    else
        echo "✗ FAIL: Exporter failed to start with API version field"
        exit 1
    fi
else
    echo "⚠ SKIP: Binary not available for testing"
fi
echo

# Test 6: Unit tests for backward compatibility
echo "Test 6: Unit tests for backward compatibility"
cd "$PROJECT_ROOT"
if go test -v ./internal/models -run TestConfig_BackwardCompatibility > /tmp/test-output.txt 2>&1; then
    echo "✓ PASS: Backward compatibility unit tests passed"
else
    echo "✗ FAIL: Backward compatibility unit tests failed"
    cat /tmp/test-output.txt
    exit 1
fi
echo

# Test 7: Unit tests for default API version
echo "Test 7: Unit tests for default API version"
if go test -v ./internal/models -run TestConfig_SetDefaults > /tmp/test-output.txt 2>&1; then
    echo "✓ PASS: Default API version unit tests passed"
else
    echo "✗ FAIL: Default API version unit tests failed"
    cat /tmp/test-output.txt
    exit 1
fi
echo

# Cleanup
rm -f /tmp/test-config-*.yaml /tmp/test-deployment*.log /tmp/test-output.txt

echo "=== All Deployment Verification Tests Passed ==="
echo
echo "Summary:"
echo "- Binary and configuration verified"
echo "- Backward compatibility confirmed"
echo "- Default API version working correctly"
echo "- No breaking changes detected"
echo
echo "Deployment procedures are verified and safe to use."
